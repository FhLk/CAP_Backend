package Manage

type Gamestate struct {
	Board      [][]Block `json:"board"`
	PlayerTurn int       `json:"player_turn"`
	Players    []Player  `json:"players"`
}

type Block struct {
	IsBlocked bool  `json:"is_blocked"`
	Item      *Item `json:"item"`
}

type Item struct {
	Type     string `json:"type"`
	Position struct {
		Row int `json:"row"`
		Col int `json:"col"`
	} `json:"position"`
}
