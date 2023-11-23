package Manage

// Design Lobby Struct
//and Method for manage Lobby (Create and Join)
//json format
import (
	"fmt"
)

type lobby struct {
	ID      string   `json:"id"`
	Players []player `json:"players"`
}

var ActiveLobbies map[string]*lobby

func CreateLobby(host player, lobbyID string) *lobby {
	newLobby, exists := ActiveLobbies[lobbyID]
	if !exists {
		newLobby = &lobby{
			ID:      lobbyID,
			Players: make([]player, 0),
		}
		ActiveLobbies[lobbyID] = newLobby
	}

	newLobby.Players = append(newLobby.Players, host)
	return newLobby
}

func JoinLobby(newPlayer player, lobbyID string) *lobby {
	lobby, exists := ActiveLobbies[lobbyID]
	if !exists {
		fmt.Println("Lobby not found:", lobbyID)
		return nil
	}

	lobby.Players = append(lobby.Players, newPlayer)
	return lobby
}
