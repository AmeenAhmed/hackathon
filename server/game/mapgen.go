package game

import (
	"fmt"
	"math/rand"
)

// =============================================================================
// MAP GENERATION CONFIGURATION
// Designed for 50-100 player maps with open areas and scattered cover
// =============================================================================

// Short winding wall segments (cover, not corridors)
const (
	MinWallStructures = 12 // Scattered wall structures across map
	MaxWallStructures = 18
	WallMinLength     = 6  // Short segments
	WallMaxLength     = 15 // Not too long
	MinTurns          = 1  // Each structure has 1-3 turns
	MaxTurns          = 3
)

// Alcoves for chests (small, spread out)
const (
	MinAlcoves    = 8
	MaxAlcoves    = 14
	AlcoveMinSize = 3
	AlcoveMaxSize = 5
)

// Isolated structures (L/U shapes for cover)
const (
	MinIsolatedStructures = 6
	MaxIsolatedStructures = 10
)

// Cacti and chests
const (
	MinCacti  = 30
	MaxCacti  = 50
	MinChests = 6
	MaxChests = 12
)

// Spacing - minimum distance between structures
const (
	MinStructureSpacing = 12 // Keep structures spread out
)

// =============================================================================

type point struct {
	X, Y int
}

type direction int

const (
	dirUp direction = iota
	dirRight
	dirDown
	dirLeft
)

func GenerateMap() MapData {
	mapData := MapData{
		Width:      1920,
		Height:     1080,
		MapObjects: []MapObject{},
		Terrain:    [MapSize][MapSize]int{},
	}

	// Generate terrain - 90% type 0, 10% scattered types 1-6
	for y := 0; y < MapSize; y++ {
		for x := 0; x < MapSize; x++ {
			if rand.Float64() < 0.9 {
				mapData.Terrain[y][x] = 0 // 90% base terrain
			} else {
				mapData.Terrain[y][x] = 1 + rand.Intn(6) // 10% types 1-6
			}
		}
	}

	walls := make(map[string]bool)
	structureCenters := []point{} // Track structure centers for spacing

	// Phase 1: Generate scattered winding wall structures
	generateScatteredWalls(&mapData, walls, &structureCenters)

	// Phase 2: Add alcoves for chests
	alcoveChestSpots := generateAlcoves(&mapData, walls, &structureCenters)

	// Phase 3: Add isolated L/U structures
	isolatedChestSpots := generateIsolatedStructures(&mapData, walls, &structureCenters)

	// Phase 4: Place chests
	allChestSpots := append(alcoveChestSpots, isolatedChestSpots...)
	placeChestsLimited(&mapData, allChestSpots, walls)

	// Phase 5: Place cacti in open areas
	placeCacti(&mapData, walls)

	return mapData
}

// generateScatteredWalls creates short winding wall segments spread across the map
func generateScatteredWalls(mapData *MapData, walls map[string]bool, centers *[]point) {
	numStructures := MinWallStructures + rand.Intn(MaxWallStructures-MinWallStructures+1)

	for i := 0; i < numStructures; i++ {
		// Find a position that's far enough from existing structures
		var x, y int
		found := false
		for attempts := 0; attempts < 50; attempts++ {
			x = 10 + rand.Intn(MapSize-20)
			y = 10 + rand.Intn(MapSize-20)

			if isFarFromStructures(x, y, *centers) {
				found = true
				break
			}
		}

		if !found {
			continue
		}

		// Create a short winding wall structure
		createWindingStructure(mapData, x, y, walls)
		*centers = append(*centers, point{x, y})
	}
}

// createWindingStructure creates a short wall that turns 1-3 times
func createWindingStructure(mapData *MapData, startX, startY int, walls map[string]bool) {
	x, y := startX, startY
	dir := direction(rand.Intn(4))
	numTurns := MinTurns + rand.Intn(MaxTurns-MinTurns+1)

	for turn := 0; turn <= numTurns; turn++ {
		// Random segment length
		segLen := WallMinLength + rand.Intn(WallMaxLength-WallMinLength+1)

		dx, dy := directionDelta(dir)
		wallType := "7" // horizontal
		if dir == dirUp || dir == dirDown {
			wallType = "8" // vertical
		}

		// Place wall segment
		for i := 0; i < segLen; i++ {
			if x < 3 || x >= MapSize-3 || y < 3 || y >= MapSize-3 {
				return
			}

			key := fmt.Sprintf("%d,%d", x, y)
			if !walls[key] {
				mapData.MapObjects = append(mapData.MapObjects, MapObject{ID: wallType, X: x, Y: y})
				walls[key] = true
			}

			x += dx
			y += dy
		}

		// Turn for next segment (if not last)
		if turn < numTurns {
			// Turn left or right
			if rand.Intn(2) == 0 {
				dir = (dir + 1) % 4
			} else {
				dir = (dir + 3) % 4
			}
		}
	}
}

// generateAlcoves creates small rooms with one opening containing chests
func generateAlcoves(mapData *MapData, walls map[string]bool, centers *[]point) []point {
	var chestSpots []point

	numAlcoves := MinAlcoves + rand.Intn(MaxAlcoves-MinAlcoves+1)

	for i := 0; i < numAlcoves; i++ {
		// Find position far from other structures
		var x, y int
		found := false
		for attempts := 0; attempts < 50; attempts++ {
			x = 8 + rand.Intn(MapSize-20)
			y = 8 + rand.Intn(MapSize-20)

			if isFarFromStructures(x, y, *centers) {
				found = true
				break
			}
		}

		if !found {
			continue
		}

		size := AlcoveMinSize + rand.Intn(AlcoveMaxSize-AlcoveMinSize+1)
		openSide := rand.Intn(4)

		spot := createAlcove(mapData, x, y, size, openSide, walls)
		if spot != nil {
			chestSpots = append(chestSpots, *spot)
			*centers = append(*centers, point{x + size/2, y + size/2})
		}
	}

	return chestSpots
}

// createAlcove creates a small room open on one side
func createAlcove(mapData *MapData, x, y, size, openSide int, walls map[string]bool) *point {
	if x < 3 || y < 3 || x+size > MapSize-3 || y+size > MapSize-3 {
		return nil
	}

	// Check if area is clear
	for dy := -1; dy <= size; dy++ {
		for dx := -1; dx <= size; dx++ {
			key := fmt.Sprintf("%d,%d", x+dx, y+dy)
			if walls[key] {
				return nil
			}
		}
	}

	switch openSide {
	case 0: // Open at top
		for i := 0; i < size; i++ {
			addWall(mapData, x+i, y+size-1, "7", walls)
		}
		for i := 1; i < size-1; i++ {
			addWall(mapData, x, y+i, "8", walls)
			addWall(mapData, x+size-1, y+i, "8", walls)
		}
		return &point{x + size/2, y + size - 2}

	case 1: // Open at right
		for i := 0; i < size; i++ {
			addWall(mapData, x, y+i, "8", walls)
		}
		for i := 1; i < size-1; i++ {
			addWall(mapData, x+i, y, "7", walls)
			addWall(mapData, x+i, y+size-1, "7", walls)
		}
		return &point{x + 1, y + size/2}

	case 2: // Open at bottom
		for i := 0; i < size; i++ {
			addWall(mapData, x+i, y, "7", walls)
		}
		for i := 1; i < size-1; i++ {
			addWall(mapData, x, y+i, "8", walls)
			addWall(mapData, x+size-1, y+i, "8", walls)
		}
		return &point{x + size/2, y + 1}

	case 3: // Open at left
		for i := 0; i < size; i++ {
			addWall(mapData, x+size-1, y+i, "8", walls)
		}
		for i := 1; i < size-1; i++ {
			addWall(mapData, x+i, y, "7", walls)
			addWall(mapData, x+i, y+size-1, "7", walls)
		}
		return &point{x + size - 2, y + size/2}
	}

	return nil
}

// generateIsolatedStructures creates L and U shapes spread across the map
func generateIsolatedStructures(mapData *MapData, walls map[string]bool, centers *[]point) []point {
	var chestSpots []point

	numStructures := MinIsolatedStructures + rand.Intn(MaxIsolatedStructures-MinIsolatedStructures+1)

	for i := 0; i < numStructures; i++ {
		var x, y int
		found := false
		for attempts := 0; attempts < 50; attempts++ {
			x = 10 + rand.Intn(MapSize-25)
			y = 10 + rand.Intn(MapSize-25)

			if isFarFromStructures(x, y, *centers) {
				found = true
				break
			}
		}

		if !found {
			continue
		}

		structType := rand.Intn(2)
		rotation := rand.Intn(4)

		var spot *point
		if structType == 0 {
			spot = placeIsolatedL(mapData, x, y, rotation, walls)
		} else {
			spot = placeIsolatedU(mapData, x, y, rotation, walls)
		}

		if spot != nil {
			chestSpots = append(chestSpots, *spot)
			*centers = append(*centers, point{x + 3, y + 3})
		}
	}

	return chestSpots
}

// placeIsolatedL creates an L-shaped structure
func placeIsolatedL(mapData *MapData, x, y, rotation int, walls map[string]bool) *point {
	hLen := 4 + rand.Intn(3) // 4-6
	vLen := 3 + rand.Intn(2) // 3-4

	// Check if area is clear
	for dy := -1; dy <= vLen+1; dy++ {
		for dx := -1; dx <= hLen+1; dx++ {
			key := fmt.Sprintf("%d,%d", x+dx, y+dy)
			if walls[key] {
				return nil
			}
		}
	}

	var chestX, chestY int

	switch rotation {
	case 0:
		for i := 0; i < hLen; i++ {
			addWall(mapData, x+i, y, "7", walls)
		}
		for i := 1; i < vLen; i++ {
			addWall(mapData, x, y+i, "8", walls)
		}
		chestX, chestY = x+1, y+1

	case 1:
		for i := 0; i < hLen; i++ {
			addWall(mapData, x+i, y, "7", walls)
		}
		for i := 1; i < vLen; i++ {
			addWall(mapData, x+hLen-1, y+i, "8", walls)
		}
		chestX, chestY = x+hLen-2, y+1

	case 2:
		for i := 0; i < hLen; i++ {
			addWall(mapData, x+i, y+vLen-1, "7", walls)
		}
		for i := 0; i < vLen-1; i++ {
			addWall(mapData, x+hLen-1, y+i, "8", walls)
		}
		chestX, chestY = x+hLen-2, y+vLen-2

	case 3:
		for i := 0; i < hLen; i++ {
			addWall(mapData, x+i, y+vLen-1, "7", walls)
		}
		for i := 0; i < vLen-1; i++ {
			addWall(mapData, x, y+i, "8", walls)
		}
		chestX, chestY = x+1, y+vLen-2
	}

	return &point{chestX, chestY}
}

// placeIsolatedU creates a U-shaped structure
func placeIsolatedU(mapData *MapData, x, y, rotation int, walls map[string]bool) *point {
	width := 4 + rand.Intn(2)  // 4-5
	height := 3 + rand.Intn(2) // 3-4

	// Check if area is clear
	for dy := -1; dy <= height+1; dy++ {
		for dx := -1; dx <= width+1; dx++ {
			key := fmt.Sprintf("%d,%d", x+dx, y+dy)
			if walls[key] {
				return nil
			}
		}
	}

	var chestX, chestY int

	switch rotation {
	case 0: // U opens upward
		for i := 0; i < width; i++ {
			addWall(mapData, x+i, y+height-1, "7", walls)
		}
		for i := 0; i < height-1; i++ {
			addWall(mapData, x, y+i, "8", walls)
			addWall(mapData, x+width-1, y+i, "8", walls)
		}
		chestX, chestY = x+width/2, y+height-2

	case 1: // U opens right
		for i := 0; i < width-1; i++ {
			addWall(mapData, x+i, y, "7", walls)
			addWall(mapData, x+i, y+height-1, "7", walls)
		}
		for i := 1; i < height-1; i++ {
			addWall(mapData, x, y+i, "8", walls)
		}
		chestX, chestY = x+1, y+height/2

	case 2: // U opens downward
		for i := 0; i < width; i++ {
			addWall(mapData, x+i, y, "7", walls)
		}
		for i := 1; i < height; i++ {
			addWall(mapData, x, y+i, "8", walls)
			addWall(mapData, x+width-1, y+i, "8", walls)
		}
		chestX, chestY = x+width/2, y+1

	case 3: // U opens left
		for i := 1; i < width; i++ {
			addWall(mapData, x+i, y, "7", walls)
			addWall(mapData, x+i, y+height-1, "7", walls)
		}
		for i := 1; i < height-1; i++ {
			addWall(mapData, x+width-1, y+i, "8", walls)
		}
		chestX, chestY = x+width-2, y+height/2
	}

	return &point{chestX, chestY}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func directionDelta(dir direction) (int, int) {
	switch dir {
	case dirUp:
		return 0, -1
	case dirRight:
		return 1, 0
	case dirDown:
		return 0, 1
	case dirLeft:
		return -1, 0
	}
	return 0, 0
}

func isFarFromStructures(x, y int, centers []point) bool {
	for _, c := range centers {
		dx := x - c.X
		dy := y - c.Y
		distSq := dx*dx + dy*dy
		if distSq < MinStructureSpacing*MinStructureSpacing {
			return false
		}
	}
	return true
}

func addWall(mapData *MapData, x, y int, wallType string, walls map[string]bool) {
	if x < 0 || x >= MapSize || y < 0 || y >= MapSize {
		return
	}
	key := fmt.Sprintf("%d,%d", x, y)
	if walls[key] {
		return
	}
	mapData.MapObjects = append(mapData.MapObjects, MapObject{ID: wallType, X: x, Y: y})
	walls[key] = true
}

func placeChestsLimited(mapData *MapData, spots []point, walls map[string]bool) {
	if len(spots) == 0 {
		return
	}

	numChests := MinChests + rand.Intn(MaxChests-MinChests+1)
	if numChests > len(spots) {
		numChests = len(spots)
	}

	rand.Shuffle(len(spots), func(i, j int) {
		spots[i], spots[j] = spots[j], spots[i]
	})

	placed := 0
	for _, spot := range spots {
		if placed >= numChests {
			break
		}
		key := fmt.Sprintf("%d,%d", spot.X, spot.Y)
		if !walls[key] && spot.X > 0 && spot.Y > 0 && spot.X < MapSize && spot.Y < MapSize {
			mapData.MapObjects = append(mapData.MapObjects, MapObject{ID: "10", X: spot.X, Y: spot.Y})
			walls[key] = true
			placed++
		}
	}
}

func placeCacti(mapData *MapData, walls map[string]bool) {
	numCacti := MinCacti + rand.Intn(MaxCacti-MinCacti+1)

	placed := 0
	attempts := 0
	maxAttempts := numCacti * 20

	for placed < numCacti && attempts < maxAttempts {
		x := 3 + rand.Intn(MapSize-6)
		y := 3 + rand.Intn(MapSize-6)
		key := fmt.Sprintf("%d,%d", x, y)

		if !walls[key] && !isNearWall(x, y, walls) {
			mapData.MapObjects = append(mapData.MapObjects, MapObject{ID: "9", X: x, Y: y})
			walls[key] = true
			placed++
		}
		attempts++
	}
}

func isNearWall(x, y int, walls map[string]bool) bool {
	for dy := -3; dy <= 3; dy++ {
		for dx := -3; dx <= 3; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			key := fmt.Sprintf("%d,%d", x+dx, y+dy)
			if walls[key] {
				return true
			}
		}
	}
	return false
}

// Noise functions for terrain
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
