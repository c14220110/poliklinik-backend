package ws

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Sesuaikan policy CORS jika diperlukan
		return true
	},
}

func ServeWS(hub *Hub) echo.HandlerFunc {
	return func(c echo.Context) error {
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		client := &Client{Conn: conn, Send: make(chan []byte, 256)}
		hub.Register <- client

		// Jalankan goroutine untuk membaca dan menulis pesan
		go client.writePump()
		go client.readPump(hub)
		return nil
	}
}

// Implementasi sederhana fungsi readPump dan writePump
func (c *Client) readPump(hub *Hub) {
	defer func() {
		hub.Unregister <- c
		c.Conn.Close()
	}()
	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		// Boleh tambahkan logika jika client mengirim pesan
	}
}

func (c *Client) writePump() {
	for message := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
	c.Conn.Close()
}