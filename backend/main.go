package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Estructura para representar a un usuario
type User struct {
	Name      string
	Conn      *websocket.Conn
	IsMaster  bool
	Speaking  bool
	TurnTime  time.Duration
	JoinedAt  time.Time
	HasSpoken bool
	PressedAt time.Time
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
	meetingStarted = false
	semaphoreGreen = false
	turnOrder      = []*User{}
	currentSpeaker *User
	meetingMutex   = &sync.Mutex{}
)

// Configuración del upgrader para WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Ajustar según sea necesario
	},
}

func main() {
	router := gin.New()

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
	log.Print("userMutex on")
	if master == nil {
		user.IsMaster = true
		master = user
		log.Printf("El usuario %s ha sido asignado como master.", user.Name)
	}
	users[conn] = user
	log.Print("userMutex off")
	usersMutex.Unlock()

	for u := range users {
		log.Printf("Usuario %s conectado.", users[u].Name)
	}

	// Notificar al usuario si es master
	initialResponse := Message{
		Type: "initial_role",
		Payload: struct {
			IsMaster bool `json:"isMaster"`
		}{
			IsMaster: user.IsMaster,
		},
	}
	log.Printf("Enviando mensaje initial_role a %s: %+v", user.Name, initialResponse)
	if err := user.Conn.WriteJSON(initialResponse); err != nil {
		log.Printf("Error al enviar initial_role a %s: %v", user.Name, err)
	}

	// Enviar el estado actual de la reunión al usuario
	sendMeetingState(user)

	// Enviar la lista actualizada de usuarios al usuario
	sendUserList(user)

	// Difundir la lista actualizada de usuarios a todos
	broadcastUserList()

	// Escuchar mensajes del usuario
	go listenToUser(user)
}

func listenToUser(user *User) {
	defer func() {
		user.Conn.Close()
		removeUser(user.Conn)
	}()

	for {
		_, msg, err := user.Conn.ReadMessage()
		if err != nil {
			log.Println("Usuario desconectado:", user.Name)
			break
		}

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
	if err := user.Conn.WriteJSON(msg); err != nil {
		log.Printf("Error al enviar meeting_state a %s: %v", user.Name, err)
	} else {
		log.Printf("Enviado meeting_state a %s", user.Name)
	}
}

func sendUserList(user *User) {
	var userList []string

	// Bloquear el mutex y recopilar la lista de usuarios
	usersMutex.Lock()
	for _, u := range users {
		userList = append(userList, u.Name)
	}
	usersMutex.Unlock()

	// Enviar el mensaje fuera del bloqueo
	msg := Message{
		Type:    "user_list",
		Payload: userList,
	}
	if err := user.Conn.WriteJSON(msg); err != nil {
		log.Printf("Error al enviar user_list a %s: %v", user.Name, err)
	} else {
		log.Printf("Enviado user_list a %s", user.Name)
	}
}

// Función para difundir el estado actual de la reunión a todos los usuarios
func broadcastMeetingState() {
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
	broadcast(msg)
}

// Función para manejar los mensajes entrantes
func handleMessage(user *User, msg Message) {
	switch msg.Type {
	case "start_meeting":
		if user.IsMaster && !meetingStarted {
			startMeeting()
		}
	case "start_semaphore":
		if user.IsMaster && meetingStarted {
			startSemaphore()
		}
	case "press_button":
		if semaphoreGreen && !user.HasSpoken {
			user.PressedAt = time.Now()
			addToTurnOrder(user)
		}
	case "end_turn":
		if user == currentSpeaker {
			endTurn()
		}
	case "force_end_turn":
		if user.IsMaster {
			endTurn()
		}
	case "skip_turn":
		if user.IsMaster {
			endTurn()
		}
	case "reset_meeting":
		if user.IsMaster {
			resetMeetingState()
		}
	case "add_virtual_user":
		if user.IsMaster {
			addVirtualUser()
		}
	default:
		log.Printf("Mensaje desconocido de %s: %s", user.Name, msg.Type)
	}
}

// Función para iniciar la reunión
func startMeeting() {
	meetingMutex.Lock()
	defer meetingMutex.Unlock()
	meetingStarted = true
	broadcastMeetingState()
}

// Función para iniciar el semáforo con tiempo aleatorio
func startSemaphore() {
	meetingMutex.Lock()
	defer meetingMutex.Unlock()

	if semaphoreGreen {
		return
	}

	// Cambiar el semáforo a rojo y notificar a todos
	semaphoreGreen = false
	broadcastMeetingState()

	// Esperar un tiempo aleatorio entre 2 y 5 segundos
	randomTime := time.Duration(rand.Intn(3)+2) * time.Second
	time.AfterFunc(randomTime, func() {
		meetingMutex.Lock()
		defer meetingMutex.Unlock()
		// Cambiar el semáforo a verde y notificar a todos
		semaphoreGreen = true
		broadcastMeetingState()
	})
}

// Función para agregar un usuario al orden de turnos
func addToTurnOrder(user *User) {
	meetingMutex.Lock()
	defer meetingMutex.Unlock()

	// Verificar si el usuario ya está en el orden de turnos
	for _, u := range turnOrder {
		if u == user {
			return
		}
	}

	turnOrder = append(turnOrder, user)
	broadcastTurnOrder()

	// Si nadie más ha presionado el botón, iniciar el turno
	if currentSpeaker == nil {
		advanceTurn()
	}
}

// Función para avanzar al siguiente turno
func advanceTurn() {
	if len(turnOrder) == 0 {
		// Si no hay más usuarios en la lista, agregar un usuario virtual
		addVirtualUser()
	}

	currentSpeaker = turnOrder[0]
	currentSpeaker.HasSpoken = true
	turnOrder = turnOrder[1:]
	broadcastTurnOrder()

	// Notificar a todos quién es el siguiente orador
	msg := Message{
		Type:    "next_speaker",
		Payload: currentSpeaker.Name,
	}
	broadcast(msg)
}

// Función para finalizar el turno actual
func endTurn() {
	currentSpeaker = nil
	advanceTurn()
}

// Función para agregar un usuario virtual
func addVirtualUser() {
	virtualUser := &User{
		Name:      "Usuario Virtual",
		IsMaster:  false,
		HasSpoken: true,
	}
	turnOrder = append(turnOrder, virtualUser)
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

	broadcast(Message{
		Type:    "meeting_reset",
		Payload: nil,
	})
	broadcastMeetingState()
	broadcastTurnOrder()
}

// Función para eliminar un usuario
func removeUser(conn *websocket.Conn) {
	var userName string
	var needToBroadcast bool

	usersMutex.Lock()
	user, exists := users[conn]

	if exists {
		if user == master {
			master = nil
			// Asignar nuevo master si hay usuarios conectados
			if len(users) > 0 {
				for _, u := range users {
					master = u
					u.IsMaster = true
					break
				}
			}
		}
		userName = user.Name
		delete(users, conn)
		needToBroadcast = true
	}
	usersMutex.Unlock()

	// Remover de turnOrder si está
	for i, u := range turnOrder {
		if u == user {
			turnOrder = append(turnOrder[:i], turnOrder[i+1:]...)
			break
		}
	}
	// Enviar la lista actualizada de usuarios
	if needToBroadcast {
		log.Printf("Usuario %s eliminado", userName)
		broadcastUserList()
		broadcastTurnOrder()
	}

}

// Función para difundir mensajes a todos los usuarios
func broadcast(msg Message) {
	var userConns []*websocket.Conn

	// Bloquear el mutex y copiar las conexiones
	usersMutex.Lock()
	for _, user := range users {
		userConns = append(userConns, user.Conn)
	}
	usersMutex.Unlock()

	// Enviar el mensaje a cada conexión fuera del bloqueo
	for _, conn := range userConns {
		err := conn.WriteJSON(msg)
		if err != nil {
			log.Printf("Error al enviar mensaje a un usuario: %v", err)
			// Opcional: manejar la desconexión del usuario
		}
	}
}

// Función para difundir el orden de turnos
func broadcastTurnOrder() {
	var order []string
	for _, user := range turnOrder {
		order = append(order, user.Name)
	}
	msg := Message{
		Type:    "turn_order",
		Payload: order,
	}
	broadcast(msg)
}

// Función para difundir la lista de usuarios conectados
func broadcastUserList() {
	var userList []string

	// Bloquear el mutex y recopilar la lista de usuarios
	usersMutex.Lock()
	for _, user := range users {
		userList = append(userList, user.Name)
	}
	usersMutex.Unlock()

	msg := Message{
		Type:    "user_list",
		Payload: userList,
	}
	broadcast(msg)
}

// Handler para restablecer la reunión vía HTTP
func resetMeeting(c *gin.Context) {
	resetMeetingState()
	c.JSON(http.StatusOK, gin.H{
		"message": "Reunión restablecida",
	})
}
