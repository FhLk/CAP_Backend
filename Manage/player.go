package Manage

// Design Player Struct
//and Method for manage player (optional)
//json format

type player struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Color  string `json:"color"`
	Status bool   `json:"status"`
}
