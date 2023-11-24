package Client

import (
	"Unity_Websocket/Manage"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
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

type Subscription struct {
	conn *Connection
	room string
}

type Connection struct {
	ws   *websocket.Conn
	send chan []byte
}

func (s *Subscription) readPump() {
	c := s.conn
	defer func() {
		H.unregister <- *s
		c.ws.Close()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		messageType, msg, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}
		HandleLobby(c, messageType, msg, s)
	}
}
func (s *Subscription) writePump() {
	c := s.conn
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	for {
		select {
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

func (c *Connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

func ServeWsLobby(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	queryValues := r.URL.Query()
	lobbyId := queryValues.Get("lobbyId")
	if lobbyId == "" {
		log.Println(err)
	} else if err != nil {
		log.Println(err)
		return
	}
	c := &Connection{send: make(chan []byte, 256), ws: ws}
	s := Subscription{c, lobbyId}
	H.register <- s
	go s.writePump()
	go s.readPump()
}

const (
	RequestCreate         = "00"
	ResponseCreateSuccess = "01"
	ResponseCreateError   = "02"
	RequestJoin           = "10"
	ResponseJoinSuccess   = "11"
	ResponseJoinError     = "12"
)

func HandleLobby(conn *Connection, messageType int, messageByte []byte, s *Subscription) {
	if messageType == websocket.TextMessage {
		var msg map[string]interface{}
		var newPlayer Manage.Player
		if err := json.Unmarshal(messageByte, &msg); err != nil {
			fmt.Println("Error parsing message:", err)
			return
		}

		checkType := msg["type"].(string)
		switch checkType {
		case RequestCreate:
			// Design follows planning
			fmt.Println("Request Create")
			playerData, ok := msg["player"].(map[string]interface{})
			if !ok {
				// Respond with an error if the player information is missing or not a valid map
				sendResponse(conn, ResponseCreateError, "Invalid player information")
				return
			}

			newPlayer.ID = playerData["id"].(string)
			newPlayer.Name = playerData["name"].(string)
			newPlayer.Color = playerData["color"].(string)
			newPlayer.Status = playerData["status"].(bool)

			lobbyID, ok := msg["lobbyId"].(string)
			if !ok {
				// Respond with an error if the lobbyID is missing or not a valid string
				sendResponse(conn, ResponseCreateError, "Invalid lobbyID")
				return
			}

			lobby, err := Manage.CreateLobby(newPlayer, lobbyID)

			if err != nil {
				sendResponse(conn, ResponseCreateError, "Error creating lobby: "+err.Error())
				return
			}

			sendResponse(conn, ResponseCreateSuccess, lobby)
		case RequestJoin:
			fmt.Println("Request Join")
			playerData, ok := msg["player"].(map[string]interface{})
			if !ok {
				// Respond with an error if the player information is missing or not a valid map
				sendResponse(conn, ResponseJoinError, "Invalid player information")
				return
			}

			newPlayer.ID = playerData["id"].(string)
			newPlayer.Name = playerData["name"].(string)
			newPlayer.Color = playerData["color"].(string)
			newPlayer.Status = playerData["status"].(bool)

			lobbyID, ok := msg["lobbyId"].(string)
			if !ok {
				// Respond with an error if the lobbyID is missing or not a valid string
				sendResponse(conn, ResponseJoinError, "Invalid lobbyID")
				return
			}

			lobby, err := Manage.JoinLobby(newPlayer, lobbyID)

			if err != nil {
				sendResponse(conn, ResponseJoinError, "Error joining lobby: "+err.Error())
				return
			}

			// Broadcast the updated lobby information to all participants
			response := map[string]interface{}{
				"type":  ResponseJoinSuccess,
				"lobby": lobby,
			}

			sendResponse(conn, ResponseJoinSuccess, lobby)

			responseBytes, err := json.Marshal(response)
			if err != nil {
				fmt.Println("Error marshaling response:", err)
				return
			}

			responseBytes = bytes.TrimSpace(bytes.Replace(responseBytes, newline, space, -1))
			message := message{s.room, responseBytes}
			H.broadcast <- message
		}
	}
}

func sendResponse(conn *Connection, responseType string, data interface{}) {
	response := map[string]interface{}{
		"type": responseType,
	}

	if data != nil {
		response["data"] = data
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		fmt.Println("Error marshaling response:", err)
		return
	}

	conn.send <- responseBytes
}
