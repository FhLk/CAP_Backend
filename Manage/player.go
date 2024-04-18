package Manage

type Player struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Pending bool   `json:"pending"`
}
