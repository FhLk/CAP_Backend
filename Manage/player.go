package Manage

// Design Player Struct
//and Method for manage player (optional)
//json format

type Player struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Color  string `json:"color"`
	Status bool   `json:"status"`
	Hearts int    `json:"hearts"`
	Shield int    `json:"shield"`
}