package main

import (
	"POC_Unity_Websocket/Client"
	"log"
	"net/http"
)

func main() {
	hub := Client.H
	go hub.Run()

	http.HandleFunc("/ws/lobby", func(w http.ResponseWriter, r *http.Request) {
		Client.ServeWsLobby(w, r)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
