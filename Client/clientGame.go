package Client

import (
	"POC_Unity_Websocket/Manage"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

const (

	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type subscription struct {
	conn *connection
	room string
}

type connection struct {
	// The websocket connection.
	ws *websocket.Conn
	// Buffered channel of outbound messages.
	send chan []byte
}

func (s *subscription) readPump() {
	c := s.conn
	defer func() {
		//Unregister
		H.unregister <- *s
		c.ws.Close()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		//Reading incoming message...
		messageType, msg, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}
		handleMessageLobby(c, messageType, msg, s)
	}
}
func (s *subscription) writePump() {
	c := s.conn
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	for {
		select {
		//Listerning message when it comes will write it into writer and then send it to the client
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (c *connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

func ServeWsLobby(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	//Get room's id from client...
	queryValues := r.URL.Query()
	roomId := queryValues.Get("roomId")
	if roomId == "" {
		//log.Println(err)
	} else if err != nil {
		log.Println(err)
		return
	}
	c := &connection{send: make(chan []byte, 256), ws: ws}
	s := subscription{c, roomId}
	H.register <- s
	go s.writePump()
	go s.readPump()
}

func handleMessageLobby(conn *connection, messageType int, messageByte []byte, s *subscription) {
	if messageType == websocket.TextMessage {
		var msg map[string]interface{}
		if err := json.Unmarshal(messageByte, &msg); err != nil {
			fmt.Println("Error parsing message:", err)
			return
		}

		checkType := msg["type"].(string)
		switch checkType {
		case "req_create":
			fmt.Println("Request Create.")
			playerId, _ := msg["playerId"].(string)

			lobby := Manage.CreateLobby(playerId)

			response := map[string]interface{}{
				"type":  "res_created",
				"lobby": lobby,
			}

			responseBytes, err := json.Marshal(response)
			if err != nil {
				fmt.Println("Error marshaling response:", err)
				return
			}

			conn.send <- responseBytes
		case "req_join":
			fmt.Println("Request Join.")
			lobbyId, _ := msg["id"].(string)
			playerId, _ := msg["playerId"].(string)

			lobby := Manage.JoinLobby(playerId, lobbyId)

			response := map[string]interface{}{
				"type":  "res_joined",
				"lobby": lobby,
			}

			responseBytes, err := json.Marshal(response)
			if err != nil {
				fmt.Println("Error marshaling response:", err)
				return
			}

			responseBytes = bytes.TrimSpace(bytes.Replace(responseBytes, newline, space, -1))
			m := message{s.room, responseBytes}
			H.broadcast <- m
		}

	}
}
