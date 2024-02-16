package Manage

type Gamestate struct {
	Board      [][]Block `json:"board"`
	PlayerTurn int       `json:"turn"`
	Players    []Player  `json:"players"`
}

type Block struct {
	Destroy bool  `json:"destroy"`
	Item    *Item `json:"item"`
}

type Item struct {
	Type string `json:"type"`
}
