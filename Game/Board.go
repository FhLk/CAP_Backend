package Game

import (
	"fmt"
	"math/rand"
)

type Board struct {
	//Hex    [][]Tile `json:"hex"`
	Height int `json:"height"`
	Width  int `json:"width"`
}

type Tile struct {
	X     int `json:"x"`
	Y     int `json:"y"`
	Type  int `json:"type"`
	Index int `json:"tile_index"`
}

func RandomBomb(w, h, bombCount int) []Tile {
	bombRange := make([]int, w*h)
	for i, _ := range bombRange {
		bombRange[i] = i
	}
	rand.Shuffle(len(bombRange), func(i, j int) {
		bombRange[i], bombRange[j] = bombRange[j], bombRange[i]
	})
	bomb := make([]Tile, bombCount) // สร้าง array 5 ช่อง เก็บค่า int เริ่มต้นด้วยค่า 0
	// สร้าง array
	for i := 0; i < len(bomb); i++ {
		randomIndex := bombRange[i]
		x := randomIndex / 10
		y := randomIndex % 10
		bomb[i].X = x
		bomb[i].Y = y
		bomb[i].Type = 1
		if x == 0 {
			bomb[i].Index = y
		} else if x != 0 {
			bomb[i].Index = (x * 10) + y
		}
	}
	fmt.Println(bomb)
	return bomb
}

func RandomLadder(w, h, ladderCount int) [][]Tile {
	ladderRange := make([]int, (w*h)-2)
	for i, _ := range ladderRange {
		ladderRange[i] = i + 1
	}

	rand.Shuffle(len(ladderRange), func(i, j int) {
		ladderRange[i], ladderRange[j] = ladderRange[j], ladderRange[i]
	})

	ladder := make([][]Tile, ladderCount)
	for i := 0; i < len(ladder); i++ {
		ladder[i] = make([]Tile, 2)
	}

	for i := 0; i < len(ladder); i++ {
		for j := 0; j < len(ladder[i]); j++ {
			randomIndex := ladderRange[i*2+j] // สุ่มจาก ladderRange ที่สับแล้ว
			x := randomIndex / 10
			y := randomIndex % 10
			ladder[i][j].X = x
			ladder[i][j].Y = y
			ladder[i][j].Type = 3
			if x == 0 {
				ladder[i][j].Index = y
			} else if x != 0 {
				ladder[i][j].Index = (x * 10) + y
			}
		}
	}
	fmt.Println(ladder)
	return ladder
}

func GenerateTile(x, y, t int) Tile {
	tile := Tile{
		X:    x,
		Y:    y,
		Type: t,
	}
	return tile
}
