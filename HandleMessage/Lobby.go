package HandleMessage

import (
	"Unity_Websocket/Client"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
)

func HandleLobby(conn *Client.Connection, messageType int, messageByte []byte, s *Client.Subscription) {
	if messageType == websocket.TextMessage {
		var msg map[string]interface{}
		if err := json.Unmarshal(messageByte, &msg); err != nil {
			fmt.Println("Error parsing message:", err)
			return
		}

		checkType := msg["type"].(string)
		switch checkType {
		case "(type_request)":
			//Design follow planning
		}
	}
}
