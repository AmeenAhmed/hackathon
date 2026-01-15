package game

import (
	"fmt"
	"math/rand"
)

// =============================================================================
// MAP GENERATION CONFIGURATION
// =============================================================================

// Structure counts
const (
	MinStructures = 8  // Minimum number of wall structures
	MaxStructures = 13 // Maximum number of wall structures
)

// Cacti counts
const (
	MinCacti = 30 // Minimum number of cacti
	MaxCacti = 50 // Maximum number of cacti
)

// L-shaped wall dimensions
const (
	LShapeMinHLen = 6  // Minimum horizontal length
	LShapeMaxHLen = 13 // Maximum horizontal length
	LShapeMinVLen = 5  // Minimum vertical length
	LShapeMaxVLen = 10 // Maximum vertical length
)

// U-shaped wall dimensions
const (
	UShapeMinWidth  = 6  // Minimum width
	UShapeMaxWidth  = 11 // Maximum width
	UShapeMinHeight = 5  // Minimum height
	UShapeMaxHeight = 9  // Maximum height
)

// Room dimensions
const (
	RoomMinWidth  = 5 // Minimum width
	RoomMaxWidth  = 9 // Maximum width
	RoomMinHeight = 4 // Minimum height
	RoomMaxHeight = 7 // Maximum height
)

// Corridor dimensions
const (
	CorridorMinLength = 8  // Minimum length
	CorridorMaxLength = 17 // Maximum length
	CorridorMinGap    = 2  // Minimum width (gap between walls)
	CorridorMaxGap    = 3  // Maximum width (gap between walls)
)

// Single wall dimensions
const (
	SingleWallMinLen = 5  // Minimum length
	SingleWallMaxLen = 12 // Maximum length
)

// Chest counts (placed at interior corners of structures)
const (
	MinChests = 4 // Minimum number of chests
	MaxChests = 8 // Maximum number of chests
)

// =============================================================================

type point struct {
	X, Y int
}

// Wall structure types
const (
	structureL = iota      // L-shaped
	structureU             // U-shaped (3 walls)
	structureRoom          // Small room (open on one side)
	structureCorridor      // Corridor segment
	structureSingleWall    // Just a wall segment
)

func noise2D(x, y int, seed int) float64 {
	n := x + y*57 + seed*131
	n = (n << 13) ^ n
	return (1.0 - float64((n*(n*n*15731+789221)+1376312589)&0x7fffffff)/1073741824.0)
}

func smoothNoise2D(x, y int, seed int) float64 {
	corners := (noise2D(x-1, y-1, seed) + noise2D(x+1, y-1, seed) +
		noise2D(x-1, y+1, seed) + noise2D(x+1, y+1, seed)) / 16.0
	sides := (noise2D(x-1, y, seed) + noise2D(x+1, y, seed) +
		noise2D(x, y-1, seed) + noise2D(x, y+1, seed)) / 8.0
	center := noise2D(x, y, seed) / 4.0
	return corners + sides + center
}

func interpolatedNoise2D(x, y float64, seed int) float64 {
	intX := int(x)
	fracX := x - float64(intX)
	intY := int(y)
	fracY := y - float64(intY)

	v1 := smoothNoise2D(intX, intY, seed)
	v2 := smoothNoise2D(intX+1, intY, seed)
	v3 := smoothNoise2D(intX, intY+1, seed)
	v4 := smoothNoise2D(intX+1, intY+1, seed)

	i1 := v1*(1-fracX) + v2*fracX
	i2 := v3*(1-fracX) + v4*fracX

	return i1*(1-fracY) + i2*fracY
}

func terrainNoise(x, y int, seed int) float64 {
	total := 0.0
	frequency := 0.05
	amplitude := 1.0
	maxValue := 0.0

	for i := 0; i < 4; i++ {
		total += interpolatedNoise2D(float64(x)*frequency, float64(y)*frequency, seed+i) * amplitude
		maxValue += amplitude
		amplitude *= 0.5
		frequency *= 2
	}

	return total / maxValue
}

func GenerateMap() MapData {
	seed := rand.Intn(10000)

	mapData := MapData{
		Width:      1920,
		Height:     1080,
		MapObjects: []MapObject{},
		Terrain:    [MapSize][MapSize]int{},
	}

	// Generate terrain
	for y := 0; y < MapSize; y++ {
		for x := 0; x < MapSize; x++ {
			noiseVal := terrainNoise(x, y, seed)
			terrainType := int((noiseVal + 1) / 2 * 7)
			if terrainType > 6 {
				terrainType = 6
			}
			if terrainType < 0 {
				terrainType = 0
			}
			mapData.Terrain[y][x] = terrainType
		}
	}

	// Generate varied wall structures with chests inside
	occupied := make(map[string]bool)
	chestSpots := generateStructures(&mapData, occupied)

	// Place chests at interior corners
	placeChests(&mapData, chestSpots, occupied)

	// Place cacti in open areas
	placeCacti(&mapData, occupied)

	return mapData
}

// generateStructures creates various wall formations and returns interior corners for chests
func generateStructures(mapData *MapData, occupied map[string]bool) []point {
	var chestSpots []point
	numStructures := MinStructures + rand.Intn(MaxStructures-MinStructures+1)

	for i := 0; i < numStructures; i++ {
		structType := rand.Intn(5)
		rotation := rand.Intn(4) // 0=up, 1=right, 2=down, 3=left

		// Random position
		x := 5 + rand.Intn(MapSize-20)
		y := 5 + rand.Intn(MapSize-20)

		var spots []point
		switch structType {
		case structureL:
			spots = placeL(mapData, x, y, rotation, occupied)
		case structureU:
			spots = placeU(mapData, x, y, rotation, occupied)
		case structureRoom:
			spots = placeRoom(mapData, x, y, rotation, occupied)
		case structureCorridor:
			spots = placeCorridor(mapData, x, y, rotation, occupied)
		case structureSingleWall:
			placeSingleWall(mapData, x, y, rotation, occupied)
			// No chest for single walls
		}
		chestSpots = append(chestSpots, spots...)
	}

	return chestSpots
}

// placeL creates an L-shaped wall and returns the interior corner
//
//	Example (rotation=0):
//	  ────
//	  │
//	  │
func placeL(mapData *MapData, x, y int, rotation int, occupied map[string]bool) []point {
	hLen := LShapeMinHLen + rand.Intn(LShapeMaxHLen-LShapeMinHLen+1)
	vLen := LShapeMinVLen + rand.Intn(LShapeMaxVLen-LShapeMinVLen+1)

	if !canPlace(x, y, hLen+2, vLen+2, occupied) {
		return nil
	}

	var chestX, chestY int

	switch rotation {
	case 0: // L opens bottom-right
		// Horizontal wall going right
		for i := 0; i < hLen; i++ {
			addWall(mapData, x+i, y, "7", occupied)
		}
		// Vertical wall going down
		for i := 1; i < vLen; i++ {
			addWall(mapData, x, y+i, "8", occupied)
		}
		chestX, chestY = x+1, y+1 // Inside corner

	case 1: // L opens bottom-left
		for i := 0; i < hLen; i++ {
			addWall(mapData, x+i, y, "7", occupied)
		}
		for i := 1; i < vLen; i++ {
			addWall(mapData, x+hLen-1, y+i, "8", occupied)
		}
		chestX, chestY = x+hLen-2, y+1

	case 2: // L opens top-left
		for i := 0; i < hLen; i++ {
			addWall(mapData, x+i, y+vLen-1, "7", occupied)
		}
		for i := 0; i < vLen-1; i++ {
			addWall(mapData, x+hLen-1, y+i, "8", occupied)
		}
		chestX, chestY = x+hLen-2, y+vLen-2

	case 3: // L opens top-right
		for i := 0; i < hLen; i++ {
			addWall(mapData, x+i, y+vLen-1, "7", occupied)
		}
		for i := 0; i < vLen-1; i++ {
			addWall(mapData, x, y+i, "8", occupied)
		}
		chestX, chestY = x+1, y+vLen-2
	}

	return []point{{chestX, chestY}}
}

// placeU creates a U-shaped wall (3 walls) and returns the interior corner
//
//	Example (rotation=0):
//	  │     │
//	  │     │
//	  ───────
func placeU(mapData *MapData, x, y int, rotation int, occupied map[string]bool) []point {
	width := UShapeMinWidth + rand.Intn(UShapeMaxWidth-UShapeMinWidth+1)
	height := UShapeMinHeight + rand.Intn(UShapeMaxHeight-UShapeMinHeight+1)

	if !canPlace(x, y, width+2, height+2, occupied) {
		return nil
	}

	var chestX, chestY int

	switch rotation {
	case 0: // U opens upward
		// Bottom horizontal
		for i := 0; i < width; i++ {
			addWall(mapData, x+i, y+height-1, "7", occupied)
		}
		// Left vertical
		for i := 0; i < height-1; i++ {
			addWall(mapData, x, y+i, "8", occupied)
		}
		// Right vertical
		for i := 0; i < height-1; i++ {
			addWall(mapData, x+width-1, y+i, "8", occupied)
		}
		chestX, chestY = x+width/2, y+height-2 // Inside bottom

	case 1: // U opens right
		for i := 0; i < width; i++ {
			addWall(mapData, x+i, y, "7", occupied)
			addWall(mapData, x+i, y+height-1, "7", occupied)
		}
		for i := 1; i < height-1; i++ {
			addWall(mapData, x, y+i, "8", occupied)
		}
		chestX, chestY = x+1, y+height/2

	case 2: // U opens downward
		for i := 0; i < width; i++ {
			addWall(mapData, x+i, y, "7", occupied)
		}
		for i := 1; i < height; i++ {
			addWall(mapData, x, y+i, "8", occupied)
			addWall(mapData, x+width-1, y+i, "8", occupied)
		}
		chestX, chestY = x+width/2, y+1

	case 3: // U opens left
		for i := 0; i < width; i++ {
			addWall(mapData, x+i, y, "7", occupied)
			addWall(mapData, x+i, y+height-1, "7", occupied)
		}
		for i := 1; i < height-1; i++ {
			addWall(mapData, x+width-1, y+i, "8", occupied)
		}
		chestX, chestY = x+width-2, y+height/2
	}

	return []point{{chestX, chestY}}
}

// placeRoom creates a small room open on one side
//
//	Example (rotation=0, open on right):
//	  ─────
//	  │
//	  │
//	  ─────
func placeRoom(mapData *MapData, x, y int, rotation int, occupied map[string]bool) []point {
	width := RoomMinWidth + rand.Intn(RoomMaxWidth-RoomMinWidth+1)
	height := RoomMinHeight + rand.Intn(RoomMaxHeight-RoomMinHeight+1)

	if !canPlace(x, y, width+2, height+2, occupied) {
		return nil
	}

	var chestX, chestY int

	switch rotation {
	case 0: // Open on right
		for i := 0; i < width; i++ {
			addWall(mapData, x+i, y, "7", occupied)
			addWall(mapData, x+i, y+height-1, "7", occupied)
		}
		for i := 1; i < height-1; i++ {
			addWall(mapData, x, y+i, "8", occupied)
		}
		chestX, chestY = x+1, y+1

	case 1: // Open on bottom
		for i := 1; i < height-1; i++ {
			addWall(mapData, x, y+i, "8", occupied)
			addWall(mapData, x+width-1, y+i, "8", occupied)
		}
		for i := 0; i < width; i++ {
			addWall(mapData, x+i, y, "7", occupied)
		}
		chestX, chestY = x+1, y+1

	case 2: // Open on left
		for i := 0; i < width; i++ {
			addWall(mapData, x+i, y, "7", occupied)
			addWall(mapData, x+i, y+height-1, "7", occupied)
		}
		for i := 1; i < height-1; i++ {
			addWall(mapData, x+width-1, y+i, "8", occupied)
		}
		chestX, chestY = x+width-2, y+1

	case 3: // Open on top
		for i := 1; i < height-1; i++ {
			addWall(mapData, x, y+i, "8", occupied)
			addWall(mapData, x+width-1, y+i, "8", occupied)
		}
		for i := 0; i < width; i++ {
			addWall(mapData, x+i, y+height-1, "7", occupied)
		}
		chestX, chestY = x+1, y+height-2
	}

	return []point{{chestX, chestY}}
}

// placeCorridor creates a corridor segment
func placeCorridor(mapData *MapData, x, y int, rotation int, occupied map[string]bool) []point {
	length := CorridorMinLength + rand.Intn(CorridorMaxLength-CorridorMinLength+1)
	gap := CorridorMinGap + rand.Intn(CorridorMaxGap-CorridorMinGap+1)

	if !canPlace(x, y, length+2, gap+4, occupied) {
		return nil
	}

	if rotation%2 == 0 { // Horizontal corridor
		for i := 0; i < length; i++ {
			addWall(mapData, x+i, y, "7", occupied)
			addWall(mapData, x+i, y+gap+1, "7", occupied)
		}
		// Chest at one end inside corridor
		return []point{{x + 1, y + 1}}
	} else { // Vertical corridor
		for i := 0; i < length; i++ {
			addWall(mapData, x, y+i, "8", occupied)
			addWall(mapData, x+gap+1, y+i, "8", occupied)
		}
		return []point{{x + 1, y + 1}}
	}
}

// placeSingleWall creates a simple wall segment (no chest)
func placeSingleWall(mapData *MapData, x, y int, rotation int, occupied map[string]bool) {
	length := SingleWallMinLen + rand.Intn(SingleWallMaxLen-SingleWallMinLen+1)

	if rotation%2 == 0 { // Horizontal
		if !canPlace(x, y, length+2, 3, occupied) {
			return
		}
		for i := 0; i < length; i++ {
			addWall(mapData, x+i, y, "7", occupied)
		}
	} else { // Vertical
		if !canPlace(x, y, 3, length+2, occupied) {
			return
		}
		for i := 0; i < length; i++ {
			addWall(mapData, x, y+i, "8", occupied)
		}
	}
}

// Helper to add a wall and mark it occupied
func addWall(mapData *MapData, x, y int, wallType string, occupied map[string]bool) {
	if x < 0 || x >= MapSize || y < 0 || y >= MapSize {
		return
	}
	key := fmt.Sprintf("%d,%d", x, y)
	if occupied[key] {
		return
	}
	mapData.MapObjects = append(mapData.MapObjects, MapObject{ID: wallType, X: x, Y: y})
	occupied[key] = true
}

// canPlace checks if an area is free
func canPlace(x, y, width, height int, occupied map[string]bool) bool {
	if x < 2 || y < 2 || x+width > MapSize-2 || y+height > MapSize-2 {
		return false
	}
	for dy := -1; dy <= height; dy++ {
		for dx := -1; dx <= width; dx++ {
			key := fmt.Sprintf("%d,%d", x+dx, y+dy)
			if occupied[key] {
				return false
			}
		}
	}
	return true
}

func placeChests(mapData *MapData, spots []point, occupied map[string]bool) {
	if len(spots) == 0 {
		return
	}

	// Determine how many chests to place
	numChests := MinChests + rand.Intn(MaxChests-MinChests+1)
	if numChests > len(spots) {
		numChests = len(spots)
	}

	// Shuffle spots and take the first numChests
	rand.Shuffle(len(spots), func(i, j int) {
		spots[i], spots[j] = spots[j], spots[i]
	})

	placed := 0
	for _, spot := range spots {
		if placed >= numChests {
			break
		}
		key := fmt.Sprintf("%d,%d", spot.X, spot.Y)
		if !occupied[key] && spot.X > 0 && spot.Y > 0 && spot.X < MapSize && spot.Y < MapSize {
			mapData.MapObjects = append(mapData.MapObjects, MapObject{ID: "10", X: spot.X, Y: spot.Y})
			occupied[key] = true
			placed++
		}
	}
}

func placeCacti(mapData *MapData, occupied map[string]bool) {
	numCacti := MinCacti + rand.Intn(MaxCacti-MinCacti+1)

	placed := 0
	attempts := 0
	maxAttempts := numCacti * 10

	for placed < numCacti && attempts < maxAttempts {
		x := 2 + rand.Intn(MapSize-4)
		y := 2 + rand.Intn(MapSize-4)
		key := fmt.Sprintf("%d,%d", x, y)

		if !occupied[key] && !isNearOccupied(x, y, occupied) {
			mapData.MapObjects = append(mapData.MapObjects, MapObject{ID: "9", X: x, Y: y})
			occupied[key] = true
			placed++
		}
		attempts++
	}
}

func isNearOccupied(x, y int, occupied map[string]bool) bool {
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			key := fmt.Sprintf("%d,%d", x+dx, y+dy)
			if occupied[key] {
				return true
			}
		}
	}
	return false
}
