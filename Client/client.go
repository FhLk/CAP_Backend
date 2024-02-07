package Client

import (
	"Unity_Websocket/Manage"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
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
	RequestFind           = "20"
	ResponseFindSuccess   = "21"
	ResponseFindError     = "22"
	PlayerAction          = "30"
	PlayerActionError     = "31"
	RollDice              = "40"
	RollDiceResponse      = "41"
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
			newPlayer.Hearts = playerData["hearts"].(int)
			newPlayer.Shield = playerData["shield"].(int)

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
				sendResponse(conn, ResponseJoinError, "Invalid player information")
				return
			}

			newPlayer.ID = playerData["id"].(string)
			newPlayer.Name = playerData["name"].(string)
			newPlayer.Color = playerData["color"].(string)
			newPlayer.Status = playerData["status"].(bool)
			newPlayer.Hearts = playerData["hearts"].(int)
			newPlayer.Shield = playerData["shield"].(int)

			lobbyID, ok := msg["lobbyId"].(string)
			if !ok {
				sendResponse(conn, ResponseJoinError, "Invalid lobbyID")
				return
			}

			lobby, err := Manage.JoinLobby(newPlayer, lobbyID)

			if err != nil {
				sendResponse(conn, ResponseJoinError, "Error joining lobby: "+err.Error())
				return
			}

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
		case RequestFind:
			fmt.Println("Request Find")
			lobbyID, ok := msg["lobbyId"].(string)
			if !ok {
				sendResponse(conn, ResponseFindError, "Invalid lobbyID")
				return
			}

			response, err := Manage.FindLobby(lobbyID)
			if err != nil {
				sendResponse(conn, ResponseFindError, "Error finding lobby: "+err.Error())
				return
			}

			sendResponse(conn, ResponseFindSuccess, response)
		case PlayerAction: // New case for player action
			fmt.Println("Player Action")
			playerIndex, ok := msg["playerIndex"].(int)
			if !ok {
				sendResponse(conn, PlayerActionError, "Invalid player index")
				return
			}

			x, ok := msg["x"].(int)
			if !ok {
				sendResponse(conn, PlayerActionError, "Invalid x coordinate")
				return
			}

			y, ok := msg["y"].(int)
			if !ok {
				sendResponse(conn, PlayerActionError, "Invalid y coordinate")
				return
			}

			// Call HandlePlayerAction function with the provided parameters
			HandlePlayerAction(playerIndex, x, y, &Manage.Gamestate{})
		case RollDice:
			randomNumber := rand.Intn(6) + 1
			fmt.Println("Player rolled the dice:", randomNumber)

			// Prepare the response data
			response := map[string]interface{}{
				"type": RollDiceResponse,
				"data": randomNumber,
			}

			// Marshal the response data
			responseBytes, err := json.Marshal(response)
			if err != nil {
				fmt.Println("Error marshaling response:", err)
				return
			}

			// Create a message with the room identifier as "gameState" and the JSON data
			message := message{Room: "gameState", Data: responseBytes}

			// Broadcast the message to all connected clients
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

func HandlePlayerAction(playerIndex int, x, y int, gameState *Manage.Gamestate) {
	// Get the current player
	player := &gameState.Players[playerIndex]

	// Check if the new position is within the bounds of the board
	if x < 0 || x >= len(gameState.Board) || y < 0 || y >= len(gameState.Board[0]) {
		fmt.Println("Invalid click: out of bounds")
		return
	}

	// Check if the cell is blocked (e.g., by a wall or obstacle)
	if gameState.Board[x][y].IsBlocked {
		fmt.Println("Invalid click: position blocked")
		return
	}

	// Perform action based on the cell's content
	item := gameState.Board[x][y].Item
	switch item.Type {
	case "Health":
		if player.Hearts < 3 {
			player.Hearts++ // Increase player's hearts
			fmt.Println("Player picked up a health item")
		} else {
			fmt.Println("Player's hearts are already at maximum")
		}
	case "Shield":
		if player.Hearts < 1 {
			player.Shield++ // Give player a shield
			fmt.Println("Player picked up a shield item")
		} else {
			fmt.Println("Player already has a shield")
		}
	case "Bomb":
		if player.Shield > 0 {
			player.Shield-- // Reduce player's shield
		} else {
			player.Hearts-- // Reduce player's hearts if no shield
		}
		fmt.Println("Player encountered a bomb")
	default:
		fmt.Println("Player clicked an empty cell")
	}

	// Print updated player information
	fmt.Printf("Player %s moved to (%d, %d). Hearts: %d, Shield: %d\n", player.ID, x, y, player.Hearts, player.Shield)

	// Broadcast the updated game state to all connected clients

	// Convert game state to JSON
	jsonData, err := json.Marshal(gameState)
	if err != nil {
		fmt.Println("Error marshaling game state:", err)
		return
	}

	// Create a message with the room identifier as "gameState" and the JSON data
	message := message{Room: "gameState", Data: jsonData}

	// Broadcast the message to all connected clients
	H.broadcast <- message
}
