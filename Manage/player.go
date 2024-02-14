package Manage

type Player struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Pending bool   `json:"pending"`
	Hearts  int    `json:"hearts"`
	Shield  int    `json:"shield"`
}
