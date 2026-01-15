package game

import (
	"fmt"
	"math"
	"math/rand"
)

// =============================================================================
// NUCLEAR THRONE-STYLE MAP GENERATION
// Multi-Walker "Drunkard's Walk" Algorithm for 50-100 Player Battle Royale
//
// Algorithm Overview:
// 1. Multiple "walkers" start from center and carve floor tiles randomly
// 2. Walkers can spawn children and despawn, creating organic cave shapes
// 3. Flood-fill ensures all areas are connected
// 4. Cover objects and chests placed at strategic locations
// 5. Spawn zones distributed around map perimeter
// =============================================================================

// -----------------------------------------------------------------------------
// CONFIGURATION CONSTANTS
// These control the "feel" of generated maps
// -----------------------------------------------------------------------------
const (
	// Target floor coverage - how much of the map should be walkable
	// For 200x200 = 40,000 tiles, we want ~2000-2500 floors for good density
	TargetFloorTiles = 2200
	MinFloorTiles    = 1800
	MaxFloorTiles    = 2600

	// Walker behavior probabilities (percentages)
	// Lower turn chances = longer corridors
	ChanceTurn90       = 12 // % chance to turn 90 degrees (reduced for longer corridors)
	ChanceTurn180      = 4  // % chance to turn 180 degrees (rare, creates enclosed dead ends)
	ChanceSpawnWalker  = 8  // % chance to spawn a child walker each step
	ChanceDespawnBase  = 8  // Base % chance to despawn when too many walkers
	Chance2x2Floor     = 55 // % chance to carve 2x2 (increased for denser combat areas)
	Chance3x3Floor     = 15 // % chance to carve 3x3 (more mini arenas for combat)

	// Walker limits
	InitialWalkers  = 6  // Starting walker count
	MaxActiveWalker = 12 // Maximum concurrent walkers
	MinActiveWalker = 3  // Don't despawn below this count

	// Object placement
	CoverDensity      = 5   // % of floor tiles that get cover objects
	MinCoverSpacing   = 4   // Minimum tiles between cover objects
	ChestCount        = 5   // Rare chests - only in enclosures
	LootCount         = 15  // Number of ammo/health pickups
	MinChestSpacing   = 40  // Large spacing - chests are rare finds
	MinLootSpacing    = 15  // Minimum tiles between loot

	// Spawn zones
	SpawnZoneCount   = 12 // Number of spawn zones around perimeter
	MinSpawnDistance = 40 // Minimum distance from center for spawns
	SpawnZoneBuffer  = 10 // Tiles from map edge for spawn zone
)

// Tile types for the generation grid
const (
	TileWall  = 0
	TileFloor = 1
)

// Directions for walker movement
const (
	DirUp    = 0
	DirRight = 1
	DirDown  = 2
	DirLeft  = 3
)

// Direction vectors for movement
var dirVectors = [4][2]int{
	{0, -1},  // Up
	{1, 0},   // Right
	{0, 1},   // Down
	{-1, 0},  // Left
}

// -----------------------------------------------------------------------------
// WALKER STRUCT
// Each walker is a "drunk" entity that carves paths through solid rock
// -----------------------------------------------------------------------------
type Walker struct {
	X, Y      int  // Current position
	Direction int  // Current facing direction (0-3)
	Steps     int  // Total steps taken
	Active    bool // Whether this walker is still carving
}

// Point represents a 2D coordinate - used for chest/loot locations
type Point struct {
	X, Y int
}

// -----------------------------------------------------------------------------
// MAIN GENERATION FUNCTION
// Orchestrates all phases of map generation
// -----------------------------------------------------------------------------
func GenerateMap() MapData {
	mapData := MapData{
		Width:      MapSize * 16, // Convert to pixel dimensions
		Height:     MapSize * 16,
		MapObjects: []MapObject{},
		Terrain:    [MapSize][MapSize]int{},
	}

	// Initialize terrain texture variation (cosmetic only)
	generateTerrainTexture(&mapData)

	// Create the carving grid (0 = wall, 1 = floor)
	grid := make([][]int, MapSize)
	for i := range grid {
		grid[i] = make([]int, MapSize)
	}

	// =========================================================================
	// PHASE 1: MULTI-WALKER FLOOR CARVING
	// This is the core Nuclear Throne algorithm
	// =========================================================================
	chestLocations, lootLocations := carveWithWalkers(grid)

	// =========================================================================
	// PHASE 2: ENSURE CONNECTIVITY
	// Flood-fill to verify all floor tiles are reachable
	// =========================================================================
	ensureConnectivity(grid)

	// =========================================================================
	// PHASE 3: COVER & OBSTACLES
	// Add tactical cover points throughout the map
	// =========================================================================
	occupiedTiles := make(map[string]bool)
	placeCoverObjects(&mapData, grid, occupiedTiles)

	// =========================================================================
	// PHASE 4: CHEST & LOOT PLACEMENT
	// Place chests at dead ends, loot at walker death points
	// =========================================================================
	placeChests(&mapData, grid, chestLocations, occupiedTiles)
	placeLoot(&mapData, grid, lootLocations, occupiedTiles)

	// =========================================================================
	// PHASE 5: BUILD WALL OBJECTS
	// Convert grid edges to renderable wall objects
	// =========================================================================
	buildWallObjects(&mapData, grid, occupiedTiles)

	return mapData
}

// -----------------------------------------------------------------------------
// PHASE 1: MULTI-WALKER CARVING SYSTEM
// The heart of Nuclear Throne-style generation
// -----------------------------------------------------------------------------
func carveWithWalkers(grid [][]int) ([]Point, []Point) {
	var chestLocations []Point  // Marked at 180-degree turns
	var lootLocations []Point   // Marked at walker death points

	// Initialize walkers at center, facing random directions
	center := MapSize / 2
	walkers := make([]*Walker, 0, MaxActiveWalker)

	for i := 0; i < InitialWalkers; i++ {
		// Offset each initial walker slightly from center
		offsetX := rand.Intn(5) - 2
		offsetY := rand.Intn(5) - 2
		walkers = append(walkers, &Walker{
			X:         center + offsetX,
			Y:         center + offsetY,
			Direction: rand.Intn(4),
			Steps:     0,
			Active:    true,
		})
	}

	// Carve the starting area (3x3 around center)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			grid[center+dy][center+dx] = TileFloor
		}
	}

	floorCount := 9 // Starting 3x3

	// Main carving loop - continue until we hit target floor count
	iteration := 0
	maxIterations := TargetFloorTiles * 3 // Safety limit

	for floorCount < TargetFloorTiles && iteration < maxIterations {
		iteration++

		// Process each active walker
		for _, walker := range walkers {
			if !walker.Active {
				continue
			}

			// ------------------------------------------------------------------
			// STEP 1: Decide on direction change
			// ------------------------------------------------------------------
			turnRoll := rand.Intn(100)

			if turnRoll < ChanceTurn180 {
				// 180-degree turn - mark this as a potential chest location!
				walker.Direction = (walker.Direction + 2) % 4
				chestLocations = append(chestLocations, Point{walker.X, walker.Y})
			} else if turnRoll < ChanceTurn180+ChanceTurn90 {
				// 90-degree turn (randomly left or right)
				if rand.Intn(2) == 0 {
					walker.Direction = (walker.Direction + 1) % 4
				} else {
					walker.Direction = (walker.Direction + 3) % 4
				}
			}
			// else: continue straight

			// ------------------------------------------------------------------
			// STEP 2: Move forward and carve
			// ------------------------------------------------------------------
			newX := walker.X + dirVectors[walker.Direction][0]
			newY := walker.Y + dirVectors[walker.Direction][1]

			// Keep walker within bounds (leave 5-tile border)
			if newX < 5 || newX >= MapSize-5 || newY < 5 || newY >= MapSize-5 {
				// Hit boundary - turn around
				walker.Direction = (walker.Direction + 2) % 4
				continue
			}

			walker.X = newX
			walker.Y = newY
			walker.Steps++

			// ------------------------------------------------------------------
			// STEP 3: Carve floor tiles (1x1, 2x2, or 3x3)
			// ------------------------------------------------------------------
			floorCount += carveFloorArea(grid, walker.X, walker.Y)

			// ------------------------------------------------------------------
			// STEP 4: Possibly spawn a child walker
			// ------------------------------------------------------------------
			activeCount := countActiveWalkers(walkers)
			if activeCount < MaxActiveWalker && rand.Intn(100) < ChanceSpawnWalker {
				// Spawn new walker at current position, random direction
				newWalker := &Walker{
					X:         walker.X,
					Y:         walker.Y,
					Direction: rand.Intn(4),
					Steps:     0,
					Active:    true,
				}
				walkers = append(walkers, newWalker)
			}

			// ------------------------------------------------------------------
			// STEP 5: Possibly despawn this walker (if too many)
			// ------------------------------------------------------------------
			if activeCount > MinActiveWalker {
				// Despawn chance increases with more walkers
				despawnChance := ChanceDespawnBase + (activeCount-MinActiveWalker)*5
				if rand.Intn(100) < despawnChance {
					walker.Active = false
					// Mark death location for loot
					lootLocations = append(lootLocations, Point{walker.X, walker.Y})
				}
			}
		}

		// If all walkers dead and not enough floors, spawn a new one
		if countActiveWalkers(walkers) == 0 && floorCount < MinFloorTiles {
			// Find a random floor tile to spawn new walker
			for attempts := 0; attempts < 100; attempts++ {
				x := 10 + rand.Intn(MapSize-20)
				y := 10 + rand.Intn(MapSize-20)
				if grid[y][x] == TileFloor {
					walkers = append(walkers, &Walker{
						X:         x,
						Y:         y,
						Direction: rand.Intn(4),
						Steps:     0,
						Active:    true,
					})
					break
				}
			}
		}

		// Safety check - stop if we have way too many floors
		if floorCount >= MaxFloorTiles {
			break
		}
	}

	return chestLocations, lootLocations
}

// carveFloorArea carves floor tiles at position, returns count of new tiles carved
func carveFloorArea(grid [][]int, x, y int) int {
	carved := 0

	// Determine carve size
	sizeRoll := rand.Intn(100)
	var size int

	if sizeRoll < Chance3x3Floor {
		size = 3 // 3x3 mini arena
	} else if sizeRoll < Chance3x3Floor+Chance2x2Floor {
		size = 2 // 2x2 open area (Nuclear Throne desert style)
	} else {
		size = 1 // 1x1 standard
	}

	// Carve the area
	halfSize := size / 2
	for dy := -halfSize; dy <= halfSize; dy++ {
		for dx := -halfSize; dx <= halfSize; dx++ {
			nx, ny := x+dx, y+dy
			if nx >= 2 && nx < MapSize-2 && ny >= 2 && ny < MapSize-2 {
				if grid[ny][nx] == TileWall {
					grid[ny][nx] = TileFloor
					carved++
				}
			}
		}
	}

	return carved
}

// countActiveWalkers returns number of currently active walkers
func countActiveWalkers(walkers []*Walker) int {
	count := 0
	for _, w := range walkers {
		if w.Active {
			count++
		}
	}
	return count
}

// -----------------------------------------------------------------------------
// PHASE 2: CONNECTIVITY VERIFICATION
// Uses flood-fill to ensure all floor tiles are reachable
// Optimized: simply removes disconnected floor tiles instead of tunneling
// -----------------------------------------------------------------------------
func ensureConnectivity(grid [][]int) {
	// Find the center floor tile to start flood fill
	center := MapSize / 2
	startX, startY := -1, -1

	// Search outward from center for a floor tile
	for radius := 0; radius < MapSize/2 && startX == -1; radius++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				x, y := center+dx, center+dy
				if x >= 0 && x < MapSize && y >= 0 && y < MapSize {
					if grid[y][x] == TileFloor {
						startX, startY = x, y
						break
					}
				}
			}
			if startX != -1 {
				break
			}
		}
	}

	if startX == -1 {
		return // No floor found (shouldn't happen)
	}

	// Flood fill from start point using 2D array (faster than map)
	visited := make([][]bool, MapSize)
	for i := range visited {
		visited[i] = make([]bool, MapSize)
	}

	floodFill(grid, visited, startX, startY)

	// Any floor tile not visited is disconnected - convert to wall
	// This is faster than trying to connect them
	for y := 0; y < MapSize; y++ {
		for x := 0; x < MapSize; x++ {
			if grid[y][x] == TileFloor && !visited[y][x] {
				grid[y][x] = TileWall
			}
		}
	}
}

// floodFill marks all connected floor tiles as visited
func floodFill(grid [][]int, visited [][]bool, startX, startY int) {
	stack := []Point{{startX, startY}}

	for len(stack) > 0 {
		// Pop from stack
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if p.X < 0 || p.X >= MapSize || p.Y < 0 || p.Y >= MapSize {
			continue
		}
		if visited[p.Y][p.X] || grid[p.Y][p.X] != TileFloor {
			continue
		}

		visited[p.Y][p.X] = true

		// Add neighbors
		stack = append(stack, Point{p.X + 1, p.Y})
		stack = append(stack, Point{p.X - 1, p.Y})
		stack = append(stack, Point{p.X, p.Y + 1})
		stack = append(stack, Point{p.X, p.Y - 1})
	}
}

// -----------------------------------------------------------------------------
// PHASE 3: COVER & OBSTACLE PLACEMENT
// Adds tactical cover throughout the map for combat
// -----------------------------------------------------------------------------
func placeCoverObjects(mapData *MapData, grid [][]int, occupied map[string]bool) {
	// Collect all valid floor positions
	var floorTiles []Point
	for y := 5; y < MapSize-5; y++ {
		for x := 5; x < MapSize-5; x++ {
			if grid[y][x] == TileFloor {
				// Check if has enough floor neighbors (not in narrow corridor)
				floorNeighbors := countFloorNeighbors(grid, x, y)
				if floorNeighbors >= 3 {
					floorTiles = append(floorTiles, Point{x, y})
				}
			}
		}
	}

	// Calculate target cover count
	targetCover := len(floorTiles) * CoverDensity / 100
	placed := 0

	// Shuffle floor tiles for random placement
	rand.Shuffle(len(floorTiles), func(i, j int) {
		floorTiles[i], floorTiles[j] = floorTiles[j], floorTiles[i]
	})

	for _, tile := range floorTiles {
		if placed >= targetCover {
			break
		}

		// Check minimum spacing from other cover
		if !checkSpacing(occupied, tile.X, tile.Y, MinCoverSpacing) {
			continue
		}

		// Place a cactus (cover object)
		key := fmt.Sprintf("%d,%d", tile.X, tile.Y)
		occupied[key] = true

		mapData.MapObjects = append(mapData.MapObjects, MapObject{
			ID: "9", // Cactus ID
			X:  tile.X,
			Y:  tile.Y,
		})
		placed++
	}
}

// countFloorNeighbors counts floor tiles in 8 directions
func countFloorNeighbors(grid [][]int, x, y int) int {
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx >= 0 && nx < MapSize && ny >= 0 && ny < MapSize {
				if grid[ny][nx] == TileFloor {
					count++
				}
			}
		}
	}
	return count
}

// checkSpacing verifies minimum distance from occupied tiles
func checkSpacing(occupied map[string]bool, x, y, minDist int) bool {
	for dy := -minDist; dy <= minDist; dy++ {
		for dx := -minDist; dx <= minDist; dx++ {
			key := fmt.Sprintf("%d,%d", x+dx, y+dy)
			if occupied[key] {
				return false
			}
		}
	}
	return true
}

// -----------------------------------------------------------------------------
// PHASE 4: CHEST & LOOT PLACEMENT
// Chests ONLY in enclosures (dead ends with limited exits) - rare finds!
// Loot at walker death points
// -----------------------------------------------------------------------------
func placeChests(mapData *MapData, grid [][]int, locations []Point, occupied map[string]bool) {
	// Filter locations to only include true enclosures
	var enclosedLocations []Point

	for _, loc := range locations {
		if loc.X < 5 || loc.X >= MapSize-5 || loc.Y < 5 || loc.Y >= MapSize-5 {
			continue
		}
		if grid[loc.Y][loc.X] != TileFloor {
			continue
		}

		// Check if this is an enclosure - count nearby walls vs floors
		// An enclosure has mostly walls around it (limited escape routes)
		enclosureScore := calculateEnclosureScore(grid, loc.X, loc.Y)

		// Only consider locations with high enclosure scores (3+ walls in immediate vicinity)
		if enclosureScore >= 5 {
			enclosedLocations = append(enclosedLocations, loc)
		}
	}

	// If we don't have enough enclosed locations, find some manually
	if len(enclosedLocations) < ChestCount {
		enclosedLocations = append(enclosedLocations, findEnclosedSpots(grid, occupied, ChestCount-len(enclosedLocations))...)
	}

	// Filter and space out chest locations
	validLocations := filterLocations(grid, enclosedLocations, occupied, MinChestSpacing, ChestCount)

	for _, loc := range validLocations {
		key := fmt.Sprintf("%d,%d", loc.X, loc.Y)
		occupied[key] = true

		mapData.MapObjects = append(mapData.MapObjects, MapObject{
			ID: "10", // Chest ID
			X:  loc.X,
			Y:  loc.Y,
		})
	}
}

// calculateEnclosureScore returns how "enclosed" a position is (higher = more enclosed)
func calculateEnclosureScore(grid [][]int, x, y int) int {
	wallCount := 0

	// Check in a 5x5 area around the point
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx < 0 || nx >= MapSize || ny < 0 || ny >= MapSize {
				wallCount++ // Out of bounds counts as wall
				continue
			}
			if grid[ny][nx] == TileWall {
				wallCount++
			}
		}
	}

	return wallCount
}

// findEnclosedSpots searches the map for naturally enclosed areas
func findEnclosedSpots(grid [][]int, occupied map[string]bool, count int) []Point {
	var spots []Point
	type scoredPoint struct {
		point Point
		score int
	}
	var candidates []scoredPoint

	// Scan the map for enclosed spots (sample to avoid scanning everything)
	step := 3 // Check every 3rd tile for speed
	for y := 10; y < MapSize-10; y += step {
		for x := 10; x < MapSize-10; x += step {
			if grid[y][x] != TileFloor {
				continue
			}
			key := fmt.Sprintf("%d,%d", x, y)
			if occupied[key] {
				continue
			}

			score := calculateEnclosureScore(grid, x, y)
			if score >= 10 { // Reasonably enclosed
				candidates = append(candidates, scoredPoint{Point{x, y}, score})
			}

			// Early exit if we have enough candidates
			if len(candidates) > count*10 {
				break
			}
		}
		if len(candidates) > count*10 {
			break
		}
	}

	// Sort by enclosure score using efficient sort
	for i := 0; i < len(candidates) && i < count*3; i++ {
		maxIdx := i
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].score > candidates[maxIdx].score {
				maxIdx = j
			}
		}
		candidates[i], candidates[maxIdx] = candidates[maxIdx], candidates[i]
	}

	// Take the most enclosed spots
	for i := 0; i < len(candidates) && len(spots) < count; i++ {
		spots = append(spots, candidates[i].point)
	}

	return spots
}

func placeLoot(mapData *MapData, grid [][]int, locations []Point, occupied map[string]bool) {
	// Filter and space out loot locations
	validLocations := filterLocations(grid, locations, occupied, MinLootSpacing, LootCount)

	for _, loc := range validLocations {
		key := fmt.Sprintf("%d,%d", loc.X, loc.Y)
		occupied[key] = true

		// Alternate between different loot types (11 = ammo, 12 = health)
		lootID := "11"
		if rand.Intn(2) == 0 {
			lootID = "12"
		}

		mapData.MapObjects = append(mapData.MapObjects, MapObject{
			ID: lootID,
			X:  loc.X,
			Y:  loc.Y,
		})
	}
}

// filterLocations filters points by validity, spacing, and count
func filterLocations(grid [][]int, locations []Point, occupied map[string]bool, minSpacing, maxCount int) []Point {
	var result []Point

	// Shuffle for randomness
	rand.Shuffle(len(locations), func(i, j int) {
		locations[i], locations[j] = locations[j], locations[i]
	})

	for _, loc := range locations {
		if len(result) >= maxCount {
			break
		}

		// Must be on floor
		if loc.X < 5 || loc.X >= MapSize-5 || loc.Y < 5 || loc.Y >= MapSize-5 {
			continue
		}
		if grid[loc.Y][loc.X] != TileFloor {
			continue
		}

		// Check spacing from existing placements
		tooClose := false
		for _, existing := range result {
			dist := math.Sqrt(float64((loc.X-existing.X)*(loc.X-existing.X) + (loc.Y-existing.Y)*(loc.Y-existing.Y)))
			if dist < float64(minSpacing) {
				tooClose = true
				break
			}
		}
		if tooClose {
			continue
		}

		// Check not already occupied
		key := fmt.Sprintf("%d,%d", loc.X, loc.Y)
		if occupied[key] {
			continue
		}

		result = append(result, loc)
	}

	// If we don't have enough from walker events, fill with random positions
	// Limit attempts to avoid infinite loop
	maxAttempts := maxCount * 100
	attempts := 0
	for len(result) < maxCount && attempts < maxAttempts {
		attempts++
		x := 10 + rand.Intn(MapSize-20)
		y := 10 + rand.Intn(MapSize-20)

		if grid[y][x] != TileFloor {
			continue
		}

		key := fmt.Sprintf("%d,%d", x, y)
		if occupied[key] {
			continue
		}

		// Check spacing
		tooClose := false
		for _, existing := range result {
			dist := math.Sqrt(float64((x-existing.X)*(x-existing.X) + (y-existing.Y)*(y-existing.Y)))
			if dist < float64(minSpacing) {
				tooClose = true
				break
			}
		}
		if tooClose {
			continue
		}

		result = append(result, Point{x, y})
	}

	return result
}

// -----------------------------------------------------------------------------
// PHASE 5: WALL OBJECT BUILDING
// Converts grid boundaries to renderable wall objects
// -----------------------------------------------------------------------------
func buildWallObjects(mapData *MapData, grid [][]int, occupied map[string]bool) {
	for y := 0; y < MapSize; y++ {
		for x := 0; x < MapSize; x++ {
			// Skip floor tiles
			if grid[y][x] == TileFloor {
				continue
			}

			// Check if this wall is adjacent to floor (visible wall)
			adjacentToFloor := false
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					ny, nx := y+dy, x+dx
					if ny >= 0 && ny < MapSize && nx >= 0 && nx < MapSize {
						if grid[ny][nx] == TileFloor {
							adjacentToFloor = true
							break
						}
					}
				}
				if adjacentToFloor {
					break
				}
			}

			if !adjacentToFloor {
				continue
			}

			// Mark as occupied
			key := fmt.Sprintf("%d,%d", x, y)
			occupied[key] = true

			// Determine wall type based on adjacent floor direction
			left := x > 0 && grid[y][x-1] == TileFloor
			right := x < MapSize-1 && grid[y][x+1] == TileFloor
			up := y > 0 && grid[y-1][x] == TileFloor
			down := y < MapSize-1 && grid[y+1][x] == TileFloor

			// Wall type: "7" = default, "8" = horizontal-facing
			wallType := "7"
			if (left || right) && !up && !down {
				wallType = "8"
			}

			mapData.MapObjects = append(mapData.MapObjects, MapObject{
				ID: wallType,
				X:  x,
				Y:  y,
			})
		}
	}
}

// -----------------------------------------------------------------------------
// TERRAIN TEXTURE GENERATION
// Cosmetic variation for the ground tiles
// -----------------------------------------------------------------------------
func generateTerrainTexture(mapData *MapData) {
	for y := 0; y < MapSize; y++ {
		for x := 0; x < MapSize; x++ {
			// 90% base tile, 10% variation tiles
			if rand.Float64() < 0.9 {
				mapData.Terrain[y][x] = 0
			} else {
				mapData.Terrain[y][x] = 1 + rand.Intn(6)
			}
		}
	}
}
