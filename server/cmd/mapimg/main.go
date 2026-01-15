package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/AmeenAhmed/hackathon/game"
)

const (
	TileSize       = 16
	TilesPerRow    = 11
	SpritesheetRel = "../../assets/terrain.png" // Relative to cmd/mapimg
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// Find spritesheet
	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	spritesheetPath := filepath.Join(filepath.Dir(execPath), SpritesheetRel)

	// Try relative path from current working directory if executable path doesn't work
	if _, err := os.Stat(spritesheetPath); os.IsNotExist(err) {
		spritesheetPath = "../../assets/terrain.png"
	}
	if _, err := os.Stat(spritesheetPath); os.IsNotExist(err) {
		spritesheetPath = "../assets/terrain.png"
	}
	if _, err := os.Stat(spritesheetPath); os.IsNotExist(err) {
		spritesheetPath = "assets/terrain.png"
	}

	// Load spritesheet
	sprites, err := loadSpritesheet(spritesheetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading spritesheet: %v\n", err)
		fmt.Fprintf(os.Stderr, "Tried path: %s\n", spritesheetPath)
		os.Exit(1)
	}

	// Generate map
	mapData := game.GenerateMap()

	// Create output image
	imgWidth := game.MapSize * TileSize
	imgHeight := game.MapSize * TileSize
	output := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

	// Build object lookup for quick access
	objects := make(map[string]int) // key -> object ID as int
	for _, obj := range mapData.MapObjects {
		key := fmt.Sprintf("%d,%d", obj.X, obj.Y)
		id := 0
		fmt.Sscanf(obj.ID, "%d", &id)
		objects[key] = id
	}

	// Draw terrain layer
	for y := 0; y < game.MapSize; y++ {
		for x := 0; x < game.MapSize; x++ {
			terrainType := mapData.Terrain[y][x]
			if terrainType >= 0 && terrainType < len(sprites) {
				drawTile(output, sprites[terrainType], x, y)
			}
		}
	}

	// Draw objects on top
	for y := 0; y < game.MapSize; y++ {
		for x := 0; x < game.MapSize; x++ {
			key := fmt.Sprintf("%d,%d", x, y)
			if objID, exists := objects[key]; exists {
				if objID >= 0 && objID < len(sprites) {
					drawTile(output, sprites[objID], x, y)
				}
			}
		}
	}

	// Determine output filename
	outputPath := "map.png"
	if len(os.Args) > 1 {
		outputPath = os.Args[1]
	}

	// Save output
	outFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	if err := png.Encode(outFile, output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding PNG: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated map image: %s (%dx%d pixels)\n", outputPath, imgWidth, imgHeight)
	fmt.Printf("  Terrain tiles: %d\n", game.MapSize*game.MapSize)
	fmt.Printf("  Objects: %d\n", len(mapData.MapObjects))
}

func loadSpritesheet(path string) ([]image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return nil, err
	}

	sprites := make([]image.Image, TilesPerRow)
	for i := 0; i < TilesPerRow; i++ {
		rect := image.Rect(i*TileSize, 0, (i+1)*TileSize, TileSize)
		sprite := image.NewRGBA(image.Rect(0, 0, TileSize, TileSize))
		draw.Draw(sprite, sprite.Bounds(), img, rect.Min, draw.Src)
		sprites[i] = sprite
	}

	return sprites, nil
}

func drawTile(dst *image.RGBA, src image.Image, tileX, tileY int) {
	dstRect := image.Rect(
		tileX*TileSize,
		tileY*TileSize,
		(tileX+1)*TileSize,
		(tileY+1)*TileSize,
	)
	draw.Draw(dst, dstRect, src, image.Point{0, 0}, draw.Over)
}
