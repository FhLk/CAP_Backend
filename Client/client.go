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
	maxMessageSize = 4096
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
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

//	type GameContext struct {
//		GameState Manage.Gamestate
//	}
var gameState Manage.Gamestate

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

		// var data map[string]interface{}
		// if err := json.Unmarshal(msg, &data); err != nil {
		// 	log.Printf("error decoding message: %v", err)
		// 	continue
		// }
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
	RequestCreate              = "00"
	ResponseCreateSuccess      = "01"
	ResponseCreateError        = "02"
	RequestJoin                = "10"
	ResponseJoinSuccess        = "11"
	ResponseJoinError          = "12"
	RequestFind                = "20"
	ResponseFindSuccess        = "21"
	ResponseFindError          = "22"
	PlayerAction               = "30"
	PlayerActionSuccess        = "31"
	PlayerActionError          = "32"
	RollDice                   = "40"
	RollDiceResponse           = "41"
	RequestBoardUpdate         = "50"
	ResponseBoardUpdateSuccess = "51"
	ResponseBoardUpdateError   = "52"
)

func HandleLobby(conn *Connection, messageType int, messageByte []byte, s *Subscription) {
	if messageType == websocket.TextMessage {
		var msg map[string]interface{}
		var newPlayer Manage.Player
		// var gameState GameContext
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
			newPlayer.Pending = playerData["pending"].(bool)
			newPlayer.Hearts = 3
			newPlayer.Shield = 0

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
			newPlayer.Pending = playerData["pending"].(bool)
			newPlayer.Hearts = 3
			newPlayer.Shield = 0

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
		case RequestBoardUpdate:
			fmt.Println("Request Board Update")
			var gameStateData Manage.Gamestate
			if err := json.Unmarshal(messageByte, &gameStateData); err != nil {
				fmt.Println("Error parsing updated game state:", err)
				return
			}

			// Update the server's game state with the received board
			gameState.Board = gameStateData.Board
			gameState.PlayerTurn = gameStateData.PlayerTurn
			gameState.Players = gameStateData.Players

			sendResponse(conn, ResponseBoardUpdateSuccess, gameState)
		case PlayerAction: // New case for player action
			fmt.Println("Player Action")
			playerIndexFloat, ok := msg["playerIndex"].(float64)
			if !ok {
				sendResponse(conn, PlayerActionError, "Invalid player index")
				return
			}
			playerIndex := int(playerIndexFloat)

			xFloat, ok := msg["x"].(float64)
			if !ok {
				sendResponse(conn, PlayerActionError, "Invalid x coordinate")
				return
			}
			x := int(xFloat)

			yFloat, ok := msg["y"].(float64)
			if !ok {
				sendResponse(conn, PlayerActionError, "Invalid y coordinate")
				return
			}
			y := int(yFloat)
			fmt.Println(gameState)

			HandlePlayerAction(playerIndex, x, y, gameState, conn, s)
		case RollDice:
			response := map[string]interface{}{
				"type":        RollDiceResponse,
				"Dice Number": msg["Dice Number"], // Use the received dice number from the message
			}

			// Marshal the response data
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

func HandlePlayerAction(playerIndex int, x, y int, gameState Manage.Gamestate, conn *Connection, s *Subscription) {
	// Get the current player
	player := gameState.Players[playerIndex]

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
	case "Heart":
		sendActionBroadcast(conn, s, playerIndex, x, y, "picked up a heart item")
		if player.Hearts < 3 {
			player.Hearts++
			fmt.Println("Player picked up a heart item")

			sendActionBroadcast(conn, s, playerIndex, x, y, "heart++")
		} else {
			fmt.Println("Player's hearts are already at maximum")

			sendActionBroadcast(conn, s, playerIndex, x, y, "max")
		}
	case "Shield":
		sendActionBroadcast(conn, s, playerIndex, x, y, "picked up a shield item")
		if player.Hearts < 1 {
			player.Hearts++
			fmt.Println("Player picked up a shield item")

			sendActionBroadcast(conn, s, playerIndex, x, y, "shield++")
		} else {
			fmt.Println("Player already has a shield")

			sendActionBroadcast(conn, s, playerIndex, x, y, "max")
		}
	case "Bomb":
		sendActionBroadcast(conn, s, playerIndex, x, y, "encountered a bomb")
		if player.Shield > 0 {
			player.Shield--

			sendActionBroadcast(conn, s, playerIndex, x, y, "shield--")
		} else {
			player.Hearts--

			sendActionBroadcast(conn, s, playerIndex, x, y, "heart--")
		}
		fmt.Println("Player encountered a bomb")

	default:
		fmt.Println("Player clicked an empty cell")

		sendActionBroadcast(conn, s, playerIndex, x, y, "clicked an empty cell")
	}

	// Print updated player information
	fmt.Printf("Player %s moved to (%d, %d). Hearts: %d, Shield: %d\n", player.ID, x, y, player.Hearts, player.Shield)

}

func sendActionBroadcast(conn *Connection, s *Subscription, playerIndex int, x, y int, action string) {
	response := map[string]interface{}{
		"action":      action,
		"playerIndex": playerIndex,
		"x":           x,
		"y":           y,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		fmt.Println("Error marshaling response:", err)
		return
	}

	responseBytes = bytes.TrimSpace(bytes.Replace(responseBytes, newline, space, -1))
	message := message{s.room, responseBytes}
	H.broadcast <- message
}
