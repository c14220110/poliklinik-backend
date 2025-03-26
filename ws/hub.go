package ws

//bertanggung jawab untuk:

// Menyimpan koneksi client.

// Menerima pesan dari API endpoint.

// Melakukan broadcast pesan ke seluruh client yang terhubung.

import (
	"fmt"

	"github.com/gorilla/websocket"
)

var HubInstance = NewHub()

func init() {
	go HubInstance.Run()
}


// Client mewakili koneksi WebSocket
type Client struct {
	Conn *websocket.Conn
	Send chan []byte
}

// Hub mengelola semua koneksi client
type Hub struct {
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
			// Debug log
			fmt.Println("Client registered")
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
				fmt.Println("Client unregistered")
			}
		case message := <-h.Broadcast:
			fmt.Println("Broadcasting message:", string(message))
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		}
	}
}