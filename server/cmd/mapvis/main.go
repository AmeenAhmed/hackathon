package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"

	"github.com/AmeenAhmed/hackathon/game"
)

// Sprite indices in the spritesheet
const (
	SpriteTerrain0 = 0  // Base terrain (90%)
	SpriteWallH    = 7  // Horizontal wall
	SpriteWallV    = 8  // Vertical wall
	SpriteCactus   = 9  // Cactus
	SpriteChest    = 10 // Chest
)

// Spritesheet holds the loaded tile images
type Spritesheet struct {
	tiles []*image.RGBA
}

func main() {
	// Load spritesheet
	spritesheet, err := loadSpritesheet("/Users/anask/code/personal/hackathon/assets/terrain.png")
	if err != nil {
		fmt.Printf("Error loading spritesheet: %v\n", err)
		return
	}
	fmt.Printf("Loaded spritesheet with %d tiles\n", len(spritesheet.tiles))
	fmt.Printf("Map size: %d x %d tiles\n\n", game.MapSize, game.MapSize)

	for i := 1; i <= 3; i++ {
		fmt.Printf("Generating map %d using game.GenerateMap()...\n", i)
		renderMap(i, spritesheet)
	}
	fmt.Println("Done!")
}

func loadSpritesheet(path string) (*Spritesheet, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	tileSize := 16
	numTiles := bounds.Dx() / tileSize

	ss := &Spritesheet{
		tiles: make([]*image.RGBA, numTiles),
	}

	// Extract each 16x16 tile
	for i := 0; i < numTiles; i++ {
		tile := image.NewRGBA(image.Rect(0, 0, tileSize, tileSize))
		srcRect := image.Rect(i*tileSize, 0, (i+1)*tileSize, tileSize)
		draw.Draw(tile, tile.Bounds(), img, srcRect.Min, draw.Src)
		ss.tiles[i] = tile
	}

	return ss, nil
}

func renderMap(mapNum int, ss *Spritesheet) {
	// Use the ACTUAL game map generation
	mapData := game.GenerateMap()

	// Build a lookup of objects by position
	objectMap := make(map[string]string) // "x,y" -> object ID
	for _, obj := range mapData.MapObjects {
		key := fmt.Sprintf("%d,%d", obj.X, obj.Y)
		objectMap[key] = obj.ID
	}

	// Reconstruct the floor grid from wall positions
	// Floor tiles are where walls are NOT and are adjacent to walls
	grid := make([][]int, game.MapSize)
	for y := range grid {
		grid[y] = make([]int, game.MapSize)
	}

	// First pass: mark all wall positions
	for _, obj := range mapData.MapObjects {
		if obj.ID == "7" || obj.ID == "8" {
			grid[obj.Y][obj.X] = 1 // Mark as wall
		}
	}

	// Second pass: infer floor tiles (tiles adjacent to walls that aren't walls)
	// and also any tile that has an object on it
	floorTiles := make(map[string]bool)

	// Objects (cacti, chests) are always on floor
	for _, obj := range mapData.MapObjects {
		if obj.ID != "7" && obj.ID != "8" {
			floorTiles[fmt.Sprintf("%d,%d", obj.X, obj.Y)] = true
		}
	}

	// Tiles adjacent to walls (that aren't walls themselves) are floor
	for _, obj := range mapData.MapObjects {
		if obj.ID == "7" || obj.ID == "8" {
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					nx, ny := obj.X+dx, obj.Y+dy
					if nx >= 0 && nx < game.MapSize && ny >= 0 && ny < game.MapSize {
						key := fmt.Sprintf("%d,%d", nx, ny)
						if grid[ny][nx] != 1 { // Not a wall
							floorTiles[key] = true
						}
					}
				}
			}
		}
	}

	// Flood fill from center to get all connected floor tiles
	visited := make(map[string]bool)
	center := game.MapSize / 2

	// Find starting floor tile near center
	var startX, startY int = -1, -1
	for radius := 0; radius < game.MapSize/2 && startX == -1; radius++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				x, y := center+dx, center+dy
				key := fmt.Sprintf("%d,%d", x, y)
				if floorTiles[key] {
					startX, startY = x, y
					break
				}
			}
			if startX != -1 {
				break
			}
		}
	}

	if startX != -1 {
		// Flood fill
		type point struct{ x, y int }
		stack := []point{{startX, startY}}

		for len(stack) > 0 {
			p := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			key := fmt.Sprintf("%d,%d", p.x, p.y)
			if visited[key] {
				continue
			}
			if p.x < 0 || p.x >= game.MapSize || p.y < 0 || p.y >= game.MapSize {
				continue
			}
			if grid[p.y][p.x] == 1 { // Wall
				continue
			}

			visited[key] = true

			// Add neighbors
			stack = append(stack, point{p.x + 1, p.y})
			stack = append(stack, point{p.x - 1, p.y})
			stack = append(stack, point{p.x, p.y + 1})
			stack = append(stack, point{p.x, p.y - 1})
		}
	}

	// Create image
	tileSize := 16
	img := image.NewRGBA(image.Rect(0, 0, game.MapSize*tileSize, game.MapSize*tileSize))

	// Stats
	floorCount := 0
	wallCount := 0
	cactusCount := 0
	chestCount := 0

	for y := 0; y < game.MapSize; y++ {
		for x := 0; x < game.MapSize; x++ {
			destRect := image.Rect(x*tileSize, y*tileSize, (x+1)*tileSize, (y+1)*tileSize)
			key := fmt.Sprintf("%d,%d", x, y)

			objID, hasObject := objectMap[key]

			if hasObject && (objID == "7" || objID == "8") {
				// Wall tile
				wallCount++
				spriteIdx := SpriteWallH
				if objID == "8" {
					spriteIdx = SpriteWallV
				}
				if spriteIdx < len(ss.tiles) {
					draw.Draw(img, destRect, ss.tiles[spriteIdx], image.Point{0, 0}, draw.Src)
				}
			} else if visited[key] || floorTiles[key] {
				// Floor tile
				floorCount++

				// Draw terrain from mapData.Terrain
				terrainIdx := mapData.Terrain[y][x]
				if terrainIdx >= 0 && terrainIdx < len(ss.tiles) {
					draw.Draw(img, destRect, ss.tiles[terrainIdx], image.Point{0, 0}, draw.Src)
				}

				// Draw object on top if present
				if hasObject {
					var spriteIdx int
					switch objID {
					case "9":
						spriteIdx = SpriteCactus
						cactusCount++
					case "10":
						spriteIdx = SpriteChest
						chestCount++
					case "11", "12":
						spriteIdx = SpriteChest // Use chest sprite for loot
						chestCount++
					default:
						continue
					}
					if spriteIdx < len(ss.tiles) {
						draw.Draw(img, destRect, ss.tiles[spriteIdx], image.Point{0, 0}, draw.Over)
					}
				}
			}
			// Else: empty space (transparent)
		}
	}

	// Save
	filename := fmt.Sprintf("/Users/anask/code/personal/hackathon/server/cmd/mapvis/map_%d.png", mapNum)
	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	png.Encode(f, img)
	f.Close()

	fmt.Printf("  Saved: %s\n", filename)
	fmt.Printf("  Stats: %d floors, %d walls, %d cacti, %d chests\n\n",
		floorCount, wallCount, cactusCount, chestCount)
}
