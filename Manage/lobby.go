package Manage

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Lobby struct {
	ID      string   `json:"id"`
	Players []Player `json:"players"`
}

var ActiveLobbies = make(map[string]*Lobby)

func GenerateRandomID() string {
	rand.Seed(time.Now().UnixNano())
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

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

var lobbyMutex sync.Mutex

const MaxPlayersInLobby = 4

func JoinLobby(newPlayer Player, lobbyID string) (*Lobby, error) {
	lobbyMutex.Lock()
	defer lobbyMutex.Unlock()

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

	if len(lobby.Players) >= MaxPlayersInLobby {
		fmt.Println("Lobby is full")
		return nil, errors.New("Lobby is full")
	}

	lobby.Players = append(lobby.Players, newPlayer)
	return lobby, nil
}

func FindLobby(lobbyID string) (*Lobby, error) {
	lobbyMutex.Lock()
	defer lobbyMutex.Unlock()

	lobby, exists := ActiveLobbies[lobbyID]
	if !exists {
		fmt.Println("Lobby", lobbyID, "is not found")
		return nil, errors.New("Lobby is not found")
	}

	return lobby, nil
}

func LeaveLobby(newPlayer Player, lobbyID string) (*Lobby, error) {
	lobby, exists := ActiveLobbies[lobbyID]
	if !exists {
		fmt.Println("Lobby", lobbyID, "is not found")
		return nil, errors.New("Lobby not found")
	}
	playerIndex := -1
	for i, existingPlayer := range lobby.Players {
		if existingPlayer.ID == newPlayer.ID {
			playerIndex = i
			break
		}
	}

	if playerIndex == -1 {
		fmt.Println("Player", newPlayer.ID, "is not in the lobby")
		return nil, errors.New("Player is not in the lobby")
	}

	// Remove the player from the slice using slice deletion
	lobby.Players = append(lobby.Players[:playerIndex], lobby.Players[playerIndex+1:]...)

	return lobby, nil
}

func DeleteLobby(lobbyID string) error {
	lobbyMutex.Lock()
	defer lobbyMutex.Unlock()

	_, exists := ActiveLobbies[lobbyID]
	if !exists {
		fmt.Println("Lobby", lobbyID, "is not found")
		return errors.New("Lobby not found")
	}

	delete(ActiveLobbies, lobbyID)

	return nil
}
