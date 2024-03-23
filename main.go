package main

import (
	"Unity_Websocket/Client"
	"log"
	"net/http"
)

func main() {
	//_ = Game.RandomBomb(10, 10, 7)
	hub := Client.H
	go hub.Run()

	http.HandleFunc("/ws/lobby", func(w http.ResponseWriter, r *http.Request) {
		Client.ServeWsLobby(w, r)
	})

	log.Fatal(http.ListenAndServe(":80", nil))
}
