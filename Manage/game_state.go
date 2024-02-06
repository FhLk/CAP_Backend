package Manage

type Gamestate struct {
	Board       [][]Block `json:"board"`
	PlayerTurn  int       `json:"player_turn"`
	Players     []Player  `json:"players"`
	CurrentDice int       `json:"current_dice"`
	DiceRolled  bool      `json:"dice_rolled"`
}

type Block struct {
	IsBlocked bool  // Indicates if the block is blocked (e.g., by a wall or obstacle)
	Item      *Item // Represents the item contained in the block, if any
}

type Item struct {
	Type string `json:"type"`
	// Additional fields based on your game's requirements
	Position struct {
		Row int `json:"row"`
		Col int `json:"col"`
	}
	Value    int `json:"value,omitempty"`    // Value for health item
	Duration int `json:"duration,omitempty"` // Duration for shield item
	Damage   int `json:"damage,omitempty"`
}
