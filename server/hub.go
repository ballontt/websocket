package main

import (
	"time"
	"os"
	"log"
)

// hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *Hub) logMonitor() {
	startTime := time.Now()

	logFile, err := os.OpenFile("./websocket.log", os.O_WRONLY | os.O_TRUNC |os.O_CREATE , 0777 )
	if err != nil {
		log.Println("open logfile error")
	}
	logger := log.New(logFile,"", log.Ldate | log.Ltime )

	defer logFile.Close()

	for {
		time.Sleep(10*time.Second)
		connNums := len(h.clients)
		currTime := time.Now()
		intervalTime := currTime.Sub(startTime).Seconds()
		logger.Printf("alive time: %.2f seconds, connection nums: %v", intervalTime, connNums)
	}
}
