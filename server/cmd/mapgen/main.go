package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/AmeenAhmed/hackathon/game"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// Uses the exact same GenerateMap() as CreateRoom()
	mapData := game.GenerateMap()

	terrainChars := []string{"·", "░", "▒", "▓", "~", "≈", "█"}

	objects := make(map[string]string)
	for _, obj := range mapData.MapObjects {
		key := fmt.Sprintf("%d,%d", obj.X, obj.Y)
		objects[key] = obj.ID
	}

	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Println("  MAP GENERATOR - Uses game.GenerateMap() (same as server)")
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Println("  Terrain: · ░ ▒ ▓ ~ ≈ █  (types 0-6)")
	fmt.Println("  Objects: ─ (7:h-wall)  │ (8:v-wall)  ♣ (9:cactus)  ◆ (10:chest)")
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Println()

	for y := 0; y < game.MapSize; y++ {
		for x := 0; x < game.MapSize; x++ {
			key := fmt.Sprintf("%d,%d", x, y)
			if objID, exists := objects[key]; exists {
				switch objID {
				case "7":
					fmt.Print("─")
				case "8":
					fmt.Print("│")
				case "9":
					fmt.Print("♣")
				case "10":
					fmt.Print("◆")
				}
			} else {
				fmt.Print(terrainChars[mapData.Terrain[y][x]])
			}
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("Total objects: %d\n", len(mapData.MapObjects))

	counts := map[string]int{"7": 0, "8": 0, "9": 0, "10": 0}
	for _, obj := range mapData.MapObjects {
		counts[obj.ID]++
	}
	fmt.Printf("  Horizontal walls (7): %d\n", counts["7"])
	fmt.Printf("  Vertical walls (8):   %d\n", counts["8"])
	fmt.Printf("  Cacti (9):            %d\n", counts["9"])
	fmt.Printf("  Chests (10):          %d\n", counts["10"])
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════════════════════════════")
}
