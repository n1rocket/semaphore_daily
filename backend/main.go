package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Estructura para representar a un usuario
type User struct {
	Name     string
	Conn     *websocket.Conn
	IsMaster bool
	Speaking bool
	TurnTime time.Duration
	JoinedAt time.Time
}

// Estructura para el mensaje WebSocket
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Mapa para almacenar los usuarios conectados
var users = make(map[*websocket.Conn]*User)
var usersMutex = &sync.Mutex{}
var master *User

// Variables para controlar el estado de la reunión
var (
	meetingStarted    = false
	semaphoreGreen    = false
	turnOrder         = []*User{}
	currentSpeaker    *User
	meetingMutex      = &sync.Mutex{}
	speakingStartTime time.Time
)

// Configuración del upgrader para WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Ajustar según sea necesario
	},
}

func main() {
	router := gin.Default()

	router.GET("/ws", handleWebSocket)
	router.POST("/reset", resetMeeting)
	router.Run(":8080")
}

// Handler para las conexiones WebSocket
func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error al actualizar a WebSocket:", err)
		return
	}
	// defer conn.Close() // No cerrar aquí, lo haremos cuando el usuario se desconecte

	log.Println("Nueva conexión WebSocket establecida.")

	// Leer el nombre del usuario
	_, msg, err := conn.ReadMessage()
	if err != nil {
		log.Println("Error al leer el mensaje inicial:", err)
		return
	}

	log.Printf("Mensaje inicial recibido: %s", msg)

	var initMsg struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(msg, &initMsg); err != nil {
		log.Println("Error al parsear el mensaje inicial:", err)
		return
	}

	log.Printf("Usuario conectado: %s", initMsg.Name)

	user := &User{
		Name:     initMsg.Name,
		Conn:     conn,
		IsMaster: false,
		JoinedAt: time.Now(),
	}

	usersMutex.Lock()
	if master == nil {
		user.IsMaster = true
		master = user
		log.Printf("El usuario %s ha sido asignado como master.", user.Name)
	}
	users[conn] = user
	usersMutex.Unlock()

	// Notificar al usuario si es master
	initialResponse := Message{
		Type: "initial_role",
		Payload: struct {
			IsMaster bool `json:"isMaster"`
		}{
			IsMaster: user.IsMaster,
		},
	}
	if err := conn.WriteJSON(initialResponse); err != nil {
		log.Println("Error al enviar initial_role:", err)
	} else {
		log.Printf("Enviado initial_role a %s: IsMaster=%v", user.Name, user.IsMaster)
	}

	// Enviar el estado actual de la reunión al usuario
	sendMeetingState(user)

	// Enviar la lista actualizada de usuarios a todos
	broadcastUserList()

	// Escuchar mensajes del usuario
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Usuario desconectado:", user.Name)
			removeUser(conn)
			break
		}

		log.Printf("Mensaje recibido de %s: %s", user.Name, msg)

		var wsMsg Message
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			log.Println("Error al parsear mensaje:", err)
			continue
		}

		handleMessage(user, wsMsg)
	}
}

// Función para enviar el estado actual de la reunión a un usuario específico
func sendMeetingState(user *User) {
	state := struct {
		MeetingStarted bool `json:"meetingStarted"`
		SemaphoreGreen bool `json:"semaphoreGreen"`
	}{
		MeetingStarted: meetingStarted,
		SemaphoreGreen: semaphoreGreen,
	}
	msg := Message{
		Type:    "meeting_state",
		Payload: state,
	}
	user.Conn.WriteJSON(msg)
}

// Función para difundir el estado actual de la reunión a todos los usuarios
func broadcastMeetingState() {
	usersMutex.Lock()
	defer usersMutex.Unlock()
	state := struct {
		MeetingStarted bool `json:"meetingStarted"`
		SemaphoreGreen bool `json:"semaphoreGreen"`
	}{
		MeetingStarted: meetingStarted,
		SemaphoreGreen: semaphoreGreen,
	}
	msg := Message{
		Type:    "meeting_state",
		Payload: state,
	}
	for _, user := range users {
		user.Conn.WriteJSON(msg)
	}
}

// Función para manejar los mensajes entrantes
func handleMessage(user *User, msg Message) {
	log.Printf("handleMessage: Usuario=%s, Tipo=%s, Payload=%v", user.Name, msg.Type, msg.Payload)

	switch msg.Type {
	case "start_meeting":
		if user.IsMaster {
			log.Printf("El master %s ha iniciado la reunión.", user.Name)
			startMeeting()
		} else {
			log.Printf("El usuario %s intentó iniciar la reunión sin ser master.", user.Name)
		}
	case "toggle_semaphore":
		if user.IsMaster {
			log.Printf("El master %s ha alternado el semáforo.", user.Name)
			toggleSemaphore()
		} else {
			log.Printf("El usuario %s intentó alternar el semáforo sin ser master.", user.Name)
		}
	case "press_button":
		if semaphoreGreen && !meetingStarted {
			usersMutex.Lock()
			// Evitar agregar al usuario más de una vez
			if !userInTurnOrder(user) {
				turnOrder = append(turnOrder, user)
				broadcastTurnOrder()
				log.Printf("Usuario %s ha presionado el botón y ha sido agregado al orden de turnos.", user.Name)
			} else {
				log.Printf("Usuario %s ya estaba en el orden de turnos.", user.Name)
			}
			usersMutex.Unlock()
		} else {
			log.Printf("Usuario %s intentó presionar el botón cuando el semáforo no estaba en verde o la reunión ya había comenzado.", user.Name)
		}
	case "start_turn":
		if user == currentSpeaker {
			user.Speaking = true
			speakingStartTime = time.Now()
			broadcast(Message{
				Type:    "turn_started",
				Payload: user.Name,
			})
			log.Printf("Usuario %s ha comenzado su turno.", user.Name)
		}
	case "end_turn":
		if user == currentSpeaker && user.Speaking {
			user.Speaking = false
			user.TurnTime = time.Since(speakingStartTime)
			log.Printf("Usuario %s ha terminado su turno. Tiempo utilizado: %v", user.Name, user.TurnTime)
			advanceTurn()
		}
	case "skip_turn":
		if user.IsMaster {
			log.Printf("El master %s ha saltado el turno.", user.Name)
			advanceTurn()
		} else {
			log.Printf("El usuario %s intentó saltar el turno sin ser master.", user.Name)
		}
	case "reorder_turn_order":
		if user.IsMaster {
			log.Printf("El master %s ha reordenado el orden de turnos.", user.Name)
			var newOrder []string
			if names, ok := msg.Payload.([]interface{}); ok {
				for _, name := range names {
					if nameStr, ok := name.(string); ok {
						newOrder = append(newOrder, nameStr)
					}
				}
				reorderTurnOrder(newOrder)
			}
		} else {
			log.Printf("El usuario %s intentó reordenar el orden de turnos sin ser master.", user.Name)
		}
	case "reset_meeting":
		if user.IsMaster {
			log.Printf("El master %s ha reiniciado la reunión.", user.Name)
			resetMeetingState()
		} else {
			log.Printf("El usuario %s intentó reiniciar la reunión sin ser master.", user.Name)
		}
	default:
		log.Printf("Tipo de mensaje desconocido recibido de %s: %s", user.Name, msg.Type)
	}
}

// Función para verificar si un usuario ya está en la lista de turnos
func userInTurnOrder(user *User) bool {
	for _, u := range turnOrder {
		if u == user {
			return true
		}
	}
	return false
}

func startMeeting() {
	meetingMutex.Lock()
	defer meetingMutex.Unlock()
	meetingStarted = true
	log.Println("La reunión ha comenzado.")
	broadcastMeetingState()
	advanceTurn()
}

func toggleSemaphore() {
	meetingMutex.Lock()
	defer meetingMutex.Unlock()
	semaphoreGreen = !semaphoreGreen
	log.Printf("Semáforo alternado: %v", semaphoreGreen)
	broadcastMeetingState()
}

// Función para avanzar al siguiente turno
func advanceTurn() {
	meetingMutex.Lock()
	defer meetingMutex.Unlock()

	if len(turnOrder) > 0 {
		currentSpeaker = turnOrder[0]
		turnOrder = turnOrder[1:]
		broadcastTurnOrder()
		broadcast(Message{
			Type:    "next_turn",
			Payload: currentSpeaker.Name,
		})
		// Enviar notificación TTS al frontend
		broadcast(Message{
			Type:    "tts",
			Payload: currentSpeaker.Name,
		})
	} else {
		currentSpeaker = nil
		// Finalizar reunión
		broadcast(Message{
			Type:    "meeting_finished",
			Payload: nil,
		})
	}
}

// Función para reordenar los turnos
func reorderTurnOrder(newOrder []string) {
	usersMutex.Lock()
	defer usersMutex.Unlock()

	var newTurnOrder []*User
	for _, name := range newOrder {
		for _, user := range turnOrder {
			if user.Name == name {
				newTurnOrder = append(newTurnOrder, user)
				break
			}
		}
	}
	turnOrder = newTurnOrder
	broadcastTurnOrder()
}

// Función para restablecer el estado de la reunión
func resetMeetingState() {
	meetingMutex.Lock()
	defer meetingMutex.Unlock()

	meetingStarted = false
	semaphoreGreen = false
	turnOrder = []*User{}
	currentSpeaker = nil
	speakingStartTime = time.Time{}

	broadcast(Message{
		Type:    "meeting_reset",
		Payload: nil,
	})
}

// Función para eliminar un usuario
func removeUser(conn *websocket.Conn) {
	usersMutex.Lock()
	defer usersMutex.Unlock()
	user := users[conn]
	delete(users, conn)
	if user == master {
		master = nil
		// Asignar nuevo master si hay usuarios conectados
		if len(users) > 0 {
			for _, u := range users {
				master = u
				master.IsMaster = true
				// Notificar al nuevo master
				master.Conn.WriteJSON(Message{
					Type:    "you_are_master",
					Payload: nil,
				})
				break
			}
		}
	}
	// Remover de turnOrder si está
	for i, u := range turnOrder {
		if u == user {
			turnOrder = append(turnOrder[:i], turnOrder[i+1:]...)
			break
		}
	}
	// Enviar la lista actualizada de usuarios
	broadcastUserList()
	broadcastTurnOrder()
}

// Función para difundir mensajes a todos los usuarios
func broadcast(msg Message) {
	usersMutex.Lock()
	defer usersMutex.Unlock()
	for conn := range users {
		conn.WriteJSON(msg)
	}
}

// Función para difundir el orden de turnos
func broadcastTurnOrder() {
	usersMutex.Lock()
	defer usersMutex.Unlock()
	var order []string
	for _, user := range turnOrder {
		order = append(order, user.Name)
	}
	broadcast(Message{
		Type:    "turn_order",
		Payload: order,
	})
}

// Función para difundir la lista de usuarios conectados
func broadcastUserList() {
	usersMutex.Lock()
	defer usersMutex.Unlock()
	var userList []string
	for _, user := range users {
		userList = append(userList, user.Name)
	}
	broadcast(Message{
		Type:    "user_list",
		Payload: userList,
	})
}

// Handler para restablecer la reunión vía HTTP
func resetMeeting(c *gin.Context) {
	resetMeetingState()
	c.JSON(http.StatusOK, gin.H{
		"message": "Reunión restablecida",
	})
}
