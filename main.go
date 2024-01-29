package main

import (
	"Unity_Websocket/Client"
	"Unity_Websocket/Manage"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	hub := Client.H
	go hub.Run()

	http.HandleFunc("/ws/lobby", func(w http.ResponseWriter, r *http.Request) {
		Client.ServeWsLobby(w, r)
	})

	r := gin.Default()

	r.GET("/lobbyid", func(c *gin.Context) {
		lobbyID := Manage.GenerateRandomID()
		c.JSON(http.StatusOK, gin.H{"randomID": lobbyID})
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
