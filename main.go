package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type room struct {
	ID     string   `json:"id"`
	HostID string   `json:"hostId"`
	People []string `json:"people"`
	Lock   sync.Mutex
}

type connection struct {
	ws   *websocket.Conn
	send chan []byte
	room *room
}

type message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type joinRequest struct {
	connection *connection
	roomID     string
	peopleID   string
}

type subscription struct {
	conn *connection
	room string
}

type hub struct {
	rooms       map[string]map[*connection]bool
	broadcast   chan message
	register    chan subscription
	unregister  chan subscription
	connections map[*connection]bool
	joinRoom    chan joinRequest
	sync.RWMutex
}

func newHub() *hub {
	return &hub{
		connections: make(map[*connection]bool),
		register:    make(chan subscription),
		unregister:  make(chan subscription),
		broadcast:   make(chan message),
		joinRoom:    make(chan joinRequest),
		rooms:       make(map[string]map[*connection]bool),
	}
}

func (h *hub) run() {
	for {
		select {
		case s := <-h.register:
			connections := h.rooms[s.room]
			if connections == nil {
				connections = make(map[*connection]bool)
				h.rooms[s.room] = connections
			}
			h.rooms[s.room][s.conn] = true

		case s := <-h.unregister:
			connections := h.rooms[s.room]
			if connections != nil {
				if _, ok := connections[s.conn]; ok {
					delete(connections, s.conn)
					close(s.conn.send)
					if len(connections) == 0 {
						delete(h.rooms, s.room)
					}
				}
			}

		case m := <-h.broadcast:
			connections := h.rooms[m.Type]
			for c := range connections {
				select {
				case c.send <- m.Data:
				default:
					close(c.send)
					delete(connections, c)
					if len(connections) == 0 {
						delete(h.rooms, m.Type)
					}
				}
			}
		}
	}
}

func (c *connection) readPump() {
	defer func() {
		H.unregister <- subscription{conn: c, room: "default"} // Assuming a default room for now
		c.ws.Close()
	}()

	for {
		_, msg, err := c.ws.ReadMessage()
		if err != nil {
			break
		}

		var m message
		if err := json.Unmarshal(msg, &m); err != nil {
			fmt.Println("Error parsing message:", err)
			continue
		}

		switch m.Type {
		case "create":
			c.handleCreateRoom(m.Data)
		case "join":
			c.handleJoinRoom(m.Data)
		}
	}
}

func (c *connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

func (c *connection) writePump() {
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

func (c *connection) handleCreateRoom(data json.RawMessage) {
	var createData struct {
		HostID string `json:"hostId"`
	}
	if err := json.Unmarshal(data, &createData); err != nil {
		fmt.Println("Error parsing create data:", err)
		return
	}

	newRoom := &room{
		ID:     generateRoomID(),
		HostID: createData.HostID,
		People: []string{createData.HostID},
	}

	c.room = newRoom
	roomMessage := message{"created", toJSON(newRoom)}

	// Send the room details back to the client
	c.send <- toJSON(roomMessage)

	// Broadcast the room creation to others
	H.broadcast <- roomMessage
}

func (c *connection) handleJoinRoom(data json.RawMessage) {
	var joinData struct {
		RoomID string `json:"roomId"`
	}
	if err := json.Unmarshal(data, &joinData); err != nil {
		fmt.Println("Error parsing join data:", err)
		return
	}

	// Generate a random people ID
	peopleID := generatePeopleID()

	// Send the join request with the random people ID
	H.joinRoom <- joinRequest{connection: c, roomID: joinData.RoomID, peopleID: peopleID}
}

var (
	H                = newHub()
	generatePeopleID = func() string { return generateRandomID(6) }
)

func generateRoomID() string {
	rand.Seed(time.Now().UnixNano())
	return generateRandomID(8)
}

func generateRandomID(length int) string {
	var characters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	id := make([]rune, length)
	for i := range id {
		id[i] = characters[rand.Intn(len(characters))]
	}
	return string(id)
}

func main() {
	go H.run()

	http.HandleFunc("/lobby", serveWs)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	c := &connection{send: make(chan []byte, 256), ws: ws}
	H.register <- subscription{conn: c, room: "default"} // Assuming a default room for now

	go c.writePump()
	c.readPump()
}

func toJSON(data interface{}) []byte {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Println("Error marshaling JSON:", err)
		return nil
	}
	return jsonData
}
