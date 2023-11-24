package Manage

// Design Lobby Struct
//and Method for manage Lobby (Create and Join)
//json format
import (
	"errors"
	"fmt"
)

type Lobby struct {
	ID      string   `json:"id"`
	Players []Player `json:"players"`
}

var ActiveLobbies = make(map[string]*Lobby)

func CreateLobby(host Player, lobbyID string) (*Lobby, error) {
	newLobby, exists := ActiveLobbies[lobbyID]
	if exists {
		fmt.Println("Lobby", lobbyID, "is already exists")
		return nil, errors.New("Lobby already exists")
	}
	newLobby = &Lobby{
		ID:      lobbyID,
		Players: []Player{host},
	}
	ActiveLobbies[lobbyID] = newLobby

	return newLobby, nil
}

func JoinLobby(newPlayer Player, lobbyID string) (*Lobby, error) {
	lobby, exists := ActiveLobbies[lobbyID]
	if !exists {
		fmt.Println("Lobby", lobbyID, "is not found")
		return nil, errors.New("Lobby not found")
	}

	for _, existingPlayer := range lobby.Players {
		if existingPlayer.ID == newPlayer.ID {
			fmt.Println("Player is already in the lobby:", newPlayer.ID)
			return nil, errors.New("Player is already in the lobby")
		}
	}

	lobby.Players = append(lobby.Players, newPlayer)
	return lobby, nil
}
