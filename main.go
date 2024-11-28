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
	Send      chan Message // Canal para enviar mensajes al usuario
	once      sync.Once
}

// Estructura para el mensaje WebSocket
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Event struct {
	User    *User
	Message Message
}

// Mapa para almacenar los usuarios conectados
var users = make(map[*websocket.Conn]*User)
var master *User

var eventChannel = make(chan Event)

// Variables para controlar el estado de la reunión
var (
	meetingStarted = false
	semaphoreGreen = false
	turnOrder      = []*User{}
	currentSpeaker *User
	// Mapa para rastrear quién ha presionado el botón
	buttonPressed = make(map[*User]bool)
)

// Configuración del upgrader para WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Ajustar según sea necesario
	},
}

func main() {
	go handleEvents()

	router := gin.New()

	router.GET("/ws", handleWebSocket)
	router.POST("/reset", resetMeeting)
	router.StaticFile("/", "./public/index.html")
	router.Static("/static", "./public")
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
		Send:     make(chan Message, 256), // Canal con buffer de 256 mensajes
	}

	if master == nil {
		user.IsMaster = true
		master = user
		log.Printf("El usuario %s ha sido asignado como master.", user.Name)
	}
	users[conn] = user

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

	go sendMessages(user)
}

func listenToUser(user *User) {
	defer func() {
		user.Conn.Close()
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

		eventChannel <- Event{User: user, Message: wsMsg}
	}
}

func sendMessages(user *User) {
	defer func() {
		user.Conn.Close()
		removeUser(user.Conn)
	}()

	for msg := range user.Send {
		if err := user.Conn.WriteJSON(msg); err != nil {
			log.Printf("Error al enviar mensaje a %s: %v", user.Name, err)
			break
		}
	}
}

func handleEvents() {
	for event := range eventChannel {
		handleMessage(event.User, event.Message)
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
	user.Send <- msg
}

func sendUserList(user *User) {
	var userList []string

	for _, u := range users {
		userList = append(userList, u.Name)
	}

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
	log.Printf("Mensaje de %s: %s", user.Name, msg.Type)
	switch msg.Type {
	case "start_meeting":
		if user.IsMaster && !meetingStarted {
			startMeeting()
		}
	case "start_semaphore":
		log.Printf("Mensaje de %s: %s", user.Name, msg.Type)
		log.Print("meetingStarted: ", meetingStarted)
		log.Print("user.IsMaster: ", user.IsMaster)
		if user.IsMaster && meetingStarted {
			startSemaphore()
		}
	case "press_button":
		addToTurnOrder(user)
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
	if meetingStarted {
		log.Println("La reunión ya ha sido iniciada.")
		return
	}

	meetingStarted = true
	log.Println("La reunión ha sido iniciada.")

	buttonPressed = make(map[*User]bool)
	broadcastMeetingState()
}

// Función para iniciar el semáforo con tiempo aleatorio
func startSemaphore() {
	if semaphoreGreen {
		return
	}

	// Cambiar el semáforo a rojo y notificar a todos
	semaphoreGreen = false
	broadcastMeetingState()

	// Esperar un tiempo aleatorio entre 2 y 5 segundos
	randomTime := time.Duration(rand.Intn(3)+2) * time.Second
	time.AfterFunc(randomTime, func() {
		// Cambiar el semáforo a verde y notificar a todos
		semaphoreGreen = true
		broadcastMeetingState()
	})
}

// Función para agregar un usuario al orden de turnos
func addToTurnOrder(user *User) {
	// Verificar si el usuario ya está en el orden de turnos
	for _, u := range turnOrder {
		if u == user {
			log.Printf("El usuario %s ya está en el orden de turnos.", user.Name)
			return
		}
	}

	turnOrder = append(turnOrder, user)
	log.Printf("Añadido al orden de turnos: %s", user.Name)

	log.Println("Estado actual de turnOrder:")
	for i, u := range turnOrder {
		log.Printf("turnOrder[%d]: %s", i, u.Name)
	}

	broadcastTurnOrder()

	// Marcar que el usuario ha presionado el botón
	buttonPressed[user] = true

	// Verificar si todos los usuarios han presionado el botón
	allPressed := len(buttonPressed) == len(users)

	if allPressed {
		log.Println("Todos los usuarios han presionado el botón. Avanzando al siguiente turno.")
		advanceTurn()
		// Reiniciar el mapa para la siguiente ronda
		buttonPressed = make(map[*User]bool)
	}
}

// Función para avanzar al siguiente turno
func advanceTurn() {
	log.Printf("advanceTurn: len(turnOrder) = %d", len(turnOrder))

	if len(turnOrder) == 0 {
		log.Println("No hay más usuarios en el orden de turnos.")
		// Enviar mensaje de fin de reunión
		msg := Message{
			Type:    "meeting_end",
			Payload: "La reunión ha finalizado. ¡Gracias por participar!",
		}
		broadcast(msg)
		log.Println("La reunión ha concluido. Se ha notificado a todos los usuarios.")
		resetMeetingState()
		return
	}

	log.Print("advanceTurn 2")

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
	log.Printf("Es el turno de: %s", currentSpeaker.Name)
}

// Función para finalizar el turno actual
func endTurn() {
	log.Print("endTurn -> launched")
	if currentSpeaker == nil {
		log.Println("No hay un orador actual para finalizar.")
		return
	}

	log.Printf("Turno finalizado de: %s", currentSpeaker.Name)
	currentSpeaker = nil

	log.Printf("Antes de advanceTurn, len(turnOrder) = %d", len(turnOrder))

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
	// Cerrar las conexiones y limpiar el mapa de usuarios
	for _, user := range users {
		// Enviar mensaje de reset antes de cerrar el canal
		user.Send <- Message{
			Type:    "meeting_reset",
			Payload: nil,
		}
		close(user.Send) // Cerrar el canal Send para finalizar sendMessages()
	}

	master = nil
	meetingStarted = false
	semaphoreGreen = false
	turnOrder = []*User{}
	currentSpeaker = nil
	buttonPressed = make(map[*User]bool)

	log.Println("Estado de la reunión ha sido reiniciado.")
}

// Función para eliminar un usuario
func removeUser(conn *websocket.Conn) {
	var userName string
	var needToBroadcast bool

	user, exists := users[conn]

	if !exists {
        return
    }

	user.once.Do(func() {
        userName = user.Name

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
		close(user.Send)
		delete(users, conn)
		needToBroadcast = true
		// Eliminar del mapa de botones presionados si estaba presente
		delete(buttonPressed, user)
    })

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
	for _, user := range users {
		select {
		case user.Send <- msg:
		default:
			log.Printf("El canal de mensajes de %s está lleno. Descartando mensaje.", user.Name)
		}
	}
}

// Función para difundir el orden de turnos
func broadcastTurnOrder() {
	log.Print("broadcastTurnOrder")
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

	for _, user := range users {
		userList = append(userList, user.Name)
	}

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
