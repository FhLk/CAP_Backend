package Client

import (
	"Unity_Websocket/Game"
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
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1000000
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
	fmt.Println(lobbyId)
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
	//In Lobby
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
	RequestBoardUpdate         = "50"
	ResponseBoardUpdateSuccess = "51"
	ResponseBoardUpdateError   = "52"
	RequestStartGame           = "60"
	ResponseStartGameSuccess   = "61"
	ResponseStartGameError     = "62"
	RequestNextPlayer          = "70"
	ResponseNextPlayerSuccess  = "71"
	ResponseNextPlayerError    = "72"
	RequestEndGame             = "80"
	ResponseEndGameSuccess     = "81"
	ResponseEndGameError       = "82"
	RequestRandom              = "90"
	ResponseRandomSuccess      = "91"
	ResponseRandomError        = "92"
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
			fmt.Println("Request Create")
			playerData, ok := msg["player"].(map[string]interface{})
			if !ok {
				sendResponse(conn, ResponseCreateError, "Invalid player information")
				return
			}

			newPlayer.ID = playerData["id"].(string)
			newPlayer.Name = playerData["name"].(string)
			//newPlayer.Pending = playerData["pending"].(bool)
			//newPlayer.Hearts = 3
			//newPlayer.Shield = 0

			lobbyID, ok := msg["lobbyId"].(string)
			if !ok {
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
		case RequestRandom:
			fmt.Println("Request Bomb")
			heightFloat, ok := msg["height"].(float64)
			height := int(heightFloat)
			widthFloat, ok := msg["width"].(float64)
			width := int(widthFloat)
			bombCountFloat, ok := msg["bomb"].(float64)
			bombCount := int(bombCountFloat)
			ladderCountFloat, ok := msg["ladder"].(float64)
			ladderCount := int(ladderCountFloat)
			if !ok {
				sendResponse(conn, ResponseRandomError, "Invalid Information")
				return
			}
			ladder := Game.RandomLadder(width, height, ladderCount)
			bomb := Game.RandomBomb(width, height, bombCount)

			response := map[string]interface{}{
				"type":   ResponseRandomSuccess,
				"bomb":   bomb,
				"ladder": ladder,
			}

			responseBytes, err := json.Marshal(response)
			if err != nil {
				fmt.Println("Error marshaling response:", err)
				return
			}

			responseBytes = bytes.TrimSpace(bytes.Replace(responseBytes, newline, space, -1))
			message := message{s.room, responseBytes}
			H.broadcast <- message
		case RequestBoardUpdate:
			fmt.Println("Request Board Update")
			var gameStateData Manage.Gamestate
			if err := json.Unmarshal(messageByte, &gameStateData); err != nil {
				fmt.Println("Error parsing updated game state:", err)
				return
			}

			gameState.Board = gameStateData.Board
			gameState.Round = gameStateData.Round
			gameState.Players = gameStateData.Players

			sendResponse(conn, ResponseBoardUpdateSuccess, gameState)
		case PlayerAction:
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
			tileTypeFloat, ok := msg["tile_type"].(float64)
			if !ok {
				sendResponse(conn, PlayerActionError, "Invalid tile type error")
				return
			}
			tileType := int(tileTypeFloat)
			HandlePlayerAction(playerIndex, x, y, tileType, gameState, conn, s)
		case RequestStartGame:
			fmt.Println("Request Start Game")
			response := map[string]interface{}{
				"type":    ResponseStartGameSuccess,
				"game":    true,
				"message": "Game starting",
			}

			responseBytes, err := json.Marshal(response)
			if err != nil {
				sendResponse(conn, ResponseStartGameError, "Error Start Game Lobby: "+err.Error())
				return
			}

			responseBytes = bytes.TrimSpace(bytes.Replace(responseBytes, newline, space, -1))
			message := message{s.room, responseBytes}
			H.broadcast <- message
		case RequestNextPlayer:
			fmt.Println("Request Next Player")
			playerIndex, ok := msg["playerIndex"].(float64)
			round, ok := msg["round"].(float64)

			if !ok {
				sendResponse(conn, ResponseNextPlayerError, "Invalid player index")
				return
			}

			if int(playerIndex) >= 1 {
				playerIndex = playerIndex - 1
			} else {
				playerIndex = playerIndex + 1
			}

			response := map[string]interface{}{
				"type":        ResponseNextPlayerSuccess,
				"playerIndex": playerIndex,
				"round":       round + 1,
				"message":     "Next Player",
			}

			responseBytes, err := json.Marshal(response)
			if err != nil {
				sendResponse(conn, ResponseNextPlayerError, "Error Change Player: "+err.Error())
				return
			}

			responseBytes = bytes.TrimSpace(bytes.Replace(responseBytes, newline, space, -1))
			message := message{s.room, responseBytes}
			H.broadcast <- message
		case RequestEndGame:
			fmt.Println("Request End Game")
			playerData, ok := msg["player"].(map[string]interface{})
			if !ok {
				sendResponse(conn, ResponseEndGameError, "Invalid player information")
				return
			}
			newPlayer.ID = playerData["id"].(string)
			newPlayer.Name = playerData["name"].(string)

			response := map[string]interface{}{
				"type":    ResponseEndGameSuccess,
				"message": "Game Ending",
				"player":  playerData,
			}

			responseBytes, err := json.Marshal(response)
			if err != nil {
				sendResponse(conn, ResponseNextPlayerError, "Error Change Player: "+err.Error())
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
		response["Lobby"] = data
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		fmt.Println("Error marshaling response:", err)
		return
	}

	conn.send <- responseBytes
}

func HandlePlayerAction(playerIndex, x, y, tile int, gameState Manage.Gamestate, conn *Connection, s *Subscription) {
	//player := gameState.Players[playerIndex]
	if x < 0 || x >= len(gameState.Board) || y < 0 || y >= len(gameState.Board[0]) {
		fmt.Println("Invalid click: out of bounds")
		return
	}

	if gameState.Board[x][y].Destroy {
		fmt.Println("Invalid click: position blocked")
		return
	}
	fmt.Println("Player clicked an empty cell")
	sendActionBroadcast(conn, s, playerIndex, x, y, tile, "clicked an empty cell")
	fmt.Printf("Player clicked to (%d, %d)", x, y)

}

func sendActionBroadcast(conn *Connection, s *Subscription, playerIndex, x, y, tile int, action string) {
	response := map[string]interface{}{
		"type":        PlayerActionSuccess,
		"action":      action,
		"playerIndex": playerIndex,
		"x":           x,
		"y":           y,
		"tile_type":   tile,
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
