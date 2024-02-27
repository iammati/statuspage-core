package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func Handle(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatalf("Failed to upgrade websocket connection: %v", err)
	}
	defer ws.Close() // Make sure we close the connection when the function returns

	clients[ws] = true

	// To read message from client
	messageType, message, err := ws.ReadMessage()
	if err != nil {
		log.Println("unable to read message from client")
	}

	log.Println("messageType: ", messageType)
	log.Println("message: ", string(message))

	if string(message) == "ping" {
		// To write message to client
		err = ws.WriteMessage(messageType, []byte("pong"))
	}
}

type Message struct {
	Message string `json:"message"`
}

var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan Message)           // broadcast channel
