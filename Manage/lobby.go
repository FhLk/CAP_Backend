package Manage

import (
	"fmt"
	"math/rand"
	"time"
)

type Lobby struct {
	ID           string   `json:"id"`
	Host         string   `json:"host"`
	Participants []string `json:"participants"`
}

var ActiveLobbies = make(map[string]*Lobby)

const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

const numbers = "0123456789"

func CreateLobby(playerId string) *Lobby {
	lobbyId := generateLobbyID()
	lobby := &Lobby{
		ID:   lobbyId,
		Host: playerId,
	}

	ActiveLobbies[lobbyId] = lobby
	lobby.Participants = append(lobby.Participants, playerId)
	return lobby
}

func generateLobbyID() string {
	rand.Seed(time.Now().UnixNano())
	var id string
	for i := 0; i < 6; i++ {
		choice := rand.Intn(2)
		if choice == 0 {
			id += string(letters[rand.Intn(len(letters))])
		} else {
			id += string(numbers[rand.Intn(len(numbers))])
		}
	}
	return id
}

func JoinLobby(playerId string, lobbyID string) *Lobby {
	lobby, exists := ActiveLobbies[lobbyID]
	if !exists {
		fmt.Println("Lobby not found:", lobbyID)
		return nil
	}

	lobby.Participants = append(lobby.Participants, playerId)

	return lobby
}

//func UpdateLobby(lobbyID string) *Lobby {
//	lobby, exists := ActiveLobbies[lobbyID]
//	if !exists {
//		fmt.Println("Lobby not found:", lobbyID)
//		return nil
//	}
//
//	update := &Lobby{
//		ID:           lobby.ID,
//		Host:         lobby.Host,
//		Participants: lobby.Participants,
//	}
//
//	return update
//}
