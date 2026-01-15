package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/AmeenAhmed/hackathon/game"
	"github.com/gorilla/websocket"
)

// Message types
type Message struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}

// Player represents a player in the game
type Player struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Color              string    `json:"color"`
	X                  float64   `json:"x"`
	Y                  float64   `json:"y"`
	Animation          string    `json:"animation"`
	Direction          string    `json:"direction"`
	GunRotation        float64   `json:"gunRotation"`
	GunFlipped         bool      `json:"gunFlipped"`
	CurrentGun         int       `json:"currentGun"`
	IsProtected        bool      `json:"isProtected"`
	ProtectionExpiry   time.Time `json:"-"` // Don't send to client
	CorrectAnswers     int       `json:"correctAnswers"`
	QuestionsAttempted int       `json:"questionsAttempted"`
	Kills              int       `json:"kills"`
}

// Client represents a connected websocket client
type Client struct {
	ID          string
	Conn        *websocket.Conn
	RoomCode    string
	IsDashboard bool
	Player      *Player
	Send        chan []byte
}

// Room represents a game room
type Room struct {
	Code       string
	Dashboard  *Client
	Players    map[string]*Client
	GameState  GameState
	MapData    game.MapData
	Created    time.Time
	LastUpdate time.Time
	TickRate   time.Duration
	mutex      sync.RWMutex
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	stopTicker chan bool
}

// GameState holds the current state of the game
type GameState struct {
	Players   map[string]*Player `json:"players"`
	GamePhase string             `json:"gamePhase"` // "waiting", "playing", "ended"
	Timer     int                `json:"timer"`
	Score     map[string]int     `json:"score"`
}

// RoomManager manages all active rooms
type RoomManager struct {
	rooms map[string]*Room
	mutex sync.RWMutex
}

// Global variables
var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	roomManager = &RoomManager{
		rooms: make(map[string]*Room),
	}

	playerColors = []string{"#FF6B6B", "#4ECDC4", "#45B7D1", "#FFA07A", "#98D8C8", "#6C5CE7", "#A8E6CF", "#FFD3B6"}
)

// Initialize random seed
func init() {
	rand.Seed(time.Now().UnixNano())
}

// Generate a 6-letter room code
func generateRoomCode() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	code := make([]byte, 6)
	for i := range code {
		code[i] = letters[rand.Intn(len(letters))]
	}
	return string(code)
}

// RoomManager methods
func (rm *RoomManager) CreateRoom() *Room {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	// Generate unique room code
	var code string
	for {
		code = generateRoomCode()
		if _, exists := rm.rooms[code]; !exists {
			break
		}
	}

	// Create new room with 30Hz tick rate
	mapData := game.GenerateMap()

	// Debug: Count different types of objects
	var wallCount, chestCount, cactusCount, lootCount int
	for _, obj := range mapData.MapObjects {
		switch obj.ID {
		case "7", "8":
			wallCount++
		case "9":
			cactusCount++
		case "10":
			chestCount++
		case "11", "12":
			lootCount++
		}
	}
	log.Printf("Generated map with %d total objects: %d walls, %d cacti, %d chests, %d loot items",
		len(mapData.MapObjects), wallCount, cactusCount, chestCount, lootCount)

	room := &Room{
		Code:       code,
		Players:    make(map[string]*Client),
		Created:    time.Now(),
		TickRate:   time.Second / 60, // 60Hz for smoother updates
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stopTicker: make(chan bool),
		GameState: GameState{
			Players:   make(map[string]*Player),
			GamePhase: "waiting",
			Score:     make(map[string]int),
		},
		MapData: mapData,
	}

	rm.rooms[code] = room

	// Start room goroutines
	go room.run()
	go room.ticker()

	log.Printf("Room created with code: %s", code)
	return room
}

func (rm *RoomManager) GetRoom(code string) (*Room, bool) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	room, exists := rm.rooms[code]
	return room, exists
}

func (rm *RoomManager) RemoveRoom(code string) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	if room, exists := rm.rooms[code]; exists {
		room.stopTicker <- true
		close(room.broadcast)
		delete(rm.rooms, code)
		log.Printf("Room %s removed", code)
	}
}

// Room methods
func (r *Room) run() {
	for {
		select {
		case client := <-r.register:
			r.addClient(client)

		case client := <-r.unregister:
			r.removeClient(client)

		case message := <-r.broadcast:
			r.broadcastToClients(message)
		}
	}
}

func (r *Room) ticker() {
	ticker := time.NewTicker(r.TickRate)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.checkSpawnProtection()
			r.broadcastGameState()
		case <-r.stopTicker:
			return
		}
	}
}

// checkSpawnProtection removes expired spawn protection
func (r *Room) checkSpawnProtection() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	for _, player := range r.GameState.Players {
		if player.IsProtected && !player.ProtectionExpiry.IsZero() && now.After(player.ProtectionExpiry) {
			player.IsProtected = false
			player.ProtectionExpiry = time.Time{} // Reset to zero value
		}
	}
}

func (r *Room) addClient(client *Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if client.IsDashboard {
		r.Dashboard = client
	} else {
		r.Players[client.ID] = client
		r.GameState.Players[client.ID] = client.Player
	}
}

func (r *Room) removeClient(client *Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if client.IsDashboard {
		r.Dashboard = nil
		log.Printf("Dashboard disconnected from room %s", r.Code)
	} else {
		// Remove from active players but keep in game state for rejoin
		delete(r.Players, client.ID)
		// Keep player data in GameState so they can rejoin with same name/color/position
		// Only remove from GameState after a timeout or when room is destroyed
		log.Printf("Player %s disconnected from room %s (data preserved for rejoin)", client.ID, r.Code)
	}

	close(client.Send)
}

func (r *Room) broadcastGameState() {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Don't send MapData in regular updates - only GameState
	// MapData is static and only needs to be sent once on join/rejoin
	state := struct {
		Type      string    `json:"type"`
		GameState GameState `json:"gameState"`
		Timestamp int64     `json:"timestamp"`
	}{
		Type:      "gameUpdate",
		GameState: r.GameState,
		Timestamp: time.Now().UnixMilli(),
	}

	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("Error marshaling game state: %v", err)
		return
	}

	r.broadcast <- data
}

func (r *Room) broadcastToOthers(senderID string, message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Send to all players except the sender
	for id, client := range r.Players {
		if id != senderID && client != nil {
			select {
			case client.Send <- data:
			default:
				// Client's send channel is full, skip
			}
		}
	}

	// Also send to dashboard
	if r.Dashboard != nil {
		select {
		case r.Dashboard.Send <- data:
		default:
			// Dashboard's send channel is full, skip
		}
	}
}

func (r *Room) broadcastToAll(message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Send to dashboard
	if r.Dashboard != nil {
		select {
		case r.Dashboard.Send <- data:
		default:
			// Dashboard's send channel is full, skip
		}
	}

	// Send to all players
	for _, client := range r.Players {
		if client != nil {
			select {
			case client.Send <- data:
			default:
				// Client's send channel is full, skip
			}
		}
	}
}

func (r *Room) sendGameStateToClient(client *Client) {
	r.mutex.RLock()
	state := struct {
		Type      string       `json:"type"`
		RoomCode  string       `json:"roomCode"`
		GameState GameState    `json:"gameState"`
		MapData   game.MapData `json:"mapData"`
		Timestamp int64        `json:"timestamp"`
	}{
		Type:      "initialState",
		RoomCode:  r.Code,
		GameState: r.GameState,
		MapData:   r.MapData,
		Timestamp: time.Now().UnixMilli(),
	}
	r.mutex.RUnlock()

	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("Error marshaling initial state: %v", err)
		return
	}

	select {
	case client.Send <- data:
	default:
	}
}

func (r *Room) broadcastToClients(message []byte) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Send to dashboard
	if r.Dashboard != nil {
		select {
		case r.Dashboard.Send <- message:
		default:
			log.Printf("Dashboard send buffer full")
		}
	}

	// Send to all players
	for _, client := range r.Players {
		if client != nil {
			select {
			case client.Send <- message:
			default:
				log.Printf("Player %s send buffer full", client.ID)
			}
		}
	}
}

func (r *Room) updatePlayerPosition(playerID string, x, y float64, animation string, direction string, gunRotation float64, gunFlipped bool, currentGun int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if player, exists := r.GameState.Players[playerID]; exists {
		player.X = x
		player.Y = y
		player.Animation = animation
		player.Direction = direction
		player.GunRotation = gunRotation
		player.GunFlipped = gunFlipped
		player.CurrentGun = currentGun
		r.LastUpdate = time.Now()
	}
}

// getRandomSpawnPoint returns a random spawn point from floor tiles (terrain 0-6)
// Avoids spawning on tiles with terrain value -1 (outside map) or walls (7+)
func (r *Room) getRandomSpawnPoint() (float64, float64) {
	// Collect all valid floor positions (terrain values 0-6)
	type FloorTile struct {
		X, Y int
	}
	var floorTiles []FloorTile

	// Look through the terrain array for valid floor tiles
	for y := 0; y < game.MapSize; y++ {
		for x := 0; x < game.MapSize; x++ {
			terrainValue := r.MapData.Terrain[y][x]
			// Floor tiles have values 0-6, avoid -1 (outside map) and 7+ (walls/objects)
			if terrainValue >= 0 && terrainValue <= 6 {
				// Also check that there's no wall object at this position
				hasWall := false
				for _, obj := range r.MapData.MapObjects {
					if obj.X == x && obj.Y == y && (obj.ID == "7" || obj.ID == "8") {
						hasWall = true
						break
					}
				}
				if !hasWall {
					floorTiles = append(floorTiles, FloorTile{X: x, Y: y})
				}
			}
		}
	}

	log.Printf("Found %d floor tiles for spawning", len(floorTiles))

	// If we have floor tiles, pick a random one
	if len(floorTiles) > 0 {
		tile := floorTiles[rand.Intn(len(floorTiles))]
		// Convert tile coordinates to pixel coordinates (center of tile)
		x := float64(tile.X*16 + 8)
		y := float64(tile.Y*16 + 8)
		log.Printf("Spawning at floor tile: tile(%d, %d) -> pixel(%.0f, %.0f)", tile.X, tile.Y, x, y)
		return x, y
	}

	// Fallback: spawn at map center with some randomness
	log.Printf("No floor tiles found! Using fallback spawn at center")
	return float64(r.MapData.Width/2 + rand.Intn(200) - 100),
		float64(r.MapData.Height/2 + rand.Intn(200) - 100)
}

// CORS middleware
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// WebSocket handler
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	// Create client
	client := &Client{
		ID:   fmt.Sprintf("%d", time.Now().UnixNano()),
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	// Handle client messages
	go client.writePump()
	go client.readPump()

	log.Printf("New client connected: %s", client.ID)
}

// Client methods
func (c *Client) readPump() {
	defer func() {
		if c.RoomCode != "" {
			if room, exists := roomManager.GetRoom(c.RoomCode); exists {
				room.unregister <- c
			}
		}
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("Read error for client %s: %v", c.ID, err)
			break
		}

		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("Error parsing message from client %s: %v", c.ID, err)
			continue
		}

		c.handleMessage(msg)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.Conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(msg Message) {
	switch msg.Type {
	case "createRoom":
		c.handleCreateRoom()

	case "startGame":
		c.handleStartGame()

	case "joinRoom":
		var data struct {
			Code string `json:"code"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(msg.Content, &data); err != nil {
			log.Printf("Error parsing joinRoom message: %v", err)
			return
		}
		// log.Printf("Received joinRoom - Code: %s, Name: %s", data.Code, data.Name)
		c.handleJoinRoom(data.Code, data.Name)

	case "updatePosition":
		var data struct {
			X           float64 `json:"x"`
			Y           float64 `json:"y"`
			Animation   string  `json:"animation"`
			Direction   string  `json:"direction"`
			GunRotation float64 `json:"gunRotation"`
			GunFlipped  bool    `json:"gunFlipped"`
			CurrentGun  int     `json:"currentGun"`
		}
		if err := json.Unmarshal(msg.Content, &data); err != nil {
			log.Printf("Error parsing updatePosition message: %v", err)
			return
		}
		c.handleUpdatePosition(data.X, data.Y, data.Animation, data.Direction, data.GunRotation, data.GunFlipped, data.CurrentGun)

	case "bulletSpawn":
		var data struct {
			BulletID  string  `json:"bulletId"`
			X         float64 `json:"x"`
			Y         float64 `json:"y"`
			VelocityX float64 `json:"velocityX"`
			VelocityY float64 `json:"velocityY"`
			Angle     float64 `json:"angle"`
			GunType   int     `json:"gunType"`
		}
		if err := json.Unmarshal(msg.Content, &data); err != nil {
			log.Printf("Error parsing bulletSpawn message: %v", err)
			return
		}
		c.handleBulletSpawn(data.BulletID, data.X, data.Y, data.VelocityX, data.VelocityY, data.Angle, data.GunType)

	case "bulletDestroy":
		var data struct {
			BulletID string `json:"bulletId"`
		}
		if err := json.Unmarshal(msg.Content, &data); err != nil {
			log.Printf("Error parsing bulletDestroy message: %v", err)
			return
		}
		c.handleBulletDestroy(data.BulletID)

	case "playerHit":
		var data struct {
			BulletID       string  `json:"bulletId"`
			TargetPlayerID string  `json:"targetPlayerId"`
			Damage         int     `json:"damage"`
			Health         float64 `json:"health"`
			IsDead         bool    `json:"isDead"`
		}
		if err := json.Unmarshal(msg.Content, &data); err != nil {
			log.Printf("Error parsing playerHit message: %v", err)
			return
		}
		c.handlePlayerHit(data.BulletID, data.TargetPlayerID, data.Damage, data.Health, data.IsDead)

	case "playerDeath":
		var data struct {
			PlayerID string `json:"playerId"`
		}
		if err := json.Unmarshal(msg.Content, &data); err != nil {
			log.Printf("Error parsing playerDeath message: %v", err)
			return
		}
		c.handlePlayerDeath(data.PlayerID)

	case "playerRespawn":
		var data struct {
			PlayerID string  `json:"playerId"`
			X        float64 `json:"x"`
			Y        float64 `json:"y"`
		}
		if err := json.Unmarshal(msg.Content, &data); err != nil {
			log.Printf("Error parsing playerRespawn message: %v", err)
			return
		}
		c.handlePlayerRespawn(data.PlayerID, data.X, data.Y)

	case "updateGamePhase":
		var data struct {
			Phase string `json:"phase"`
		}
		if err := json.Unmarshal(msg.Content, &data); err != nil {
			log.Printf("Error parsing updateGamePhase message: %v", err)
			return
		}
		c.handleUpdateGamePhase(data.Phase)

	case "rejoinRoom":
		var data struct {
			Code     string `json:"code"`
			PlayerID string `json:"playerId"`
		}
		if err := json.Unmarshal(msg.Content, &data); err != nil {
			log.Printf("Error parsing rejoinRoom message: %v", err)
			return
		}
		c.handleRejoinRoom(data.Code, data.PlayerID)

	case "rejoinDashboard":
		var data struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal(msg.Content, &data); err != nil {
			log.Printf("Error parsing rejoinDashboard message: %v", err)
			return
		}
		c.handleRejoinDashboard(data.Code)

	case "getState":
		// Send current game state to the requesting client
		c.handleGetState()

	case "updateScore":
		var data struct {
			CorrectAnswers     int `json:"correctAnswers"`
			QuestionsAttempted int `json:"questionsAttempted"`
			Kills              int `json:"kills"`
		}
		if err := json.Unmarshal(msg.Content, &data); err != nil {
			log.Printf("Error parsing updateScore message: %v", err)
			return
		}
		c.handleUpdateScore(data.CorrectAnswers, data.QuestionsAttempted, data.Kills)

	case "endGame":
		c.handleEndGame()

	case "updateTimer":
		var data struct {
			Timer int `json:"timer"`
		}
		if err := json.Unmarshal(msg.Content, &data); err != nil {
			log.Printf("Error parsing updateTimer message: %v", err)
			return
		}
		c.handleUpdateTimer(data.Timer)
	}
}

func (c *Client) handleCreateRoom() {
	c.IsDashboard = true
	room := roomManager.CreateRoom()
	c.RoomCode = room.Code
	room.register <- c

	// Send room code to dashboard
	response := struct {
		Type     string `json:"type"`
		RoomCode string `json:"roomCode"`
	}{
		Type:     "roomCreated",
		RoomCode: room.Code,
	}

	data, _ := json.Marshal(response)
	c.Send <- data
}

func (c *Client) handleStartGame() {
	// Only dashboard can start the game
	if !c.IsDashboard {
		log.Printf("Non-dashboard client tried to start game")
		return
	}

	room, exists := roomManager.GetRoom(c.RoomCode)
	if !exists {
		log.Printf("Room not found for start game: %s", c.RoomCode)
		return
	}

	// Update game phase to playing
	room.mutex.Lock()
	room.GameState.GamePhase = "playing"
	room.mutex.Unlock()

	// Broadcast game started to all clients
	response := struct {
		Type      string `json:"type"`
		GamePhase string `json:"gamePhase"`
	}{
		Type:      "gameStarted",
		GamePhase: "playing",
	}

	data, _ := json.Marshal(response)
	room.broadcastToAll(data)

	log.Printf("Game started in room %s", c.RoomCode)
}

func (c *Client) handleJoinRoom(code string, playerName string) {
	room, exists := roomManager.GetRoom(code)
	if !exists {
		response := struct {
			Type  string `json:"type"`
			Error string `json:"error"`
		}{
			Type:  "error",
			Error: "Room not found",
		}
		data, _ := json.Marshal(response)
		c.Send <- data
		return
	}

	// Create player at a random chest spawn point
	spawnX, spawnY := room.getRandomSpawnPoint()
	c.Player = &Player{
		ID:          c.ID,
		Name:        playerName,
		Color:       playerColors[rand.Intn(len(playerColors))],
		X:           spawnX,
		Y:           spawnY,
		Animation:   "idle",
		Direction:   "right",
		IsProtected: false, // No spawn protection on initial join
	}

	log.Printf("Created player - ID: %s, Name: %s, Color: %s, Spawn: (%.0f, %.0f) [Chest spawn]",
		c.Player.ID, c.Player.Name, c.Player.Color, c.Player.X, c.Player.Y)

	c.RoomCode = code
	room.register <- c

	// Send success response with terrain data
	response := struct {
		Type     string       `json:"type"`
		PlayerID string       `json:"playerId"`
		Player   *Player      `json:"player"`
		MapData  game.MapData `json:"mapData"`
	}{
		Type:     "joinedRoom",
		PlayerID: c.ID,
		Player:   c.Player,
		MapData:  room.MapData,
	}

	data, _ := json.Marshal(response)
	c.Send <- data
}

func (c *Client) handleRejoinRoom(code string, playerID string) {
	room, exists := roomManager.GetRoom(code)
	if !exists {
		response := struct {
			Type  string `json:"type"`
			Error string `json:"error"`
		}{
			Type:  "error",
			Error: "Room not found",
		}
		data, _ := json.Marshal(response)
		c.Send <- data
		return
	}

	// Check if player exists in the room's game state
	room.mutex.RLock()
	existingPlayer, playerExists := room.GameState.Players[playerID]
	room.mutex.RUnlock()

	if playerExists {
		// Reuse existing player data but give them a new spawn point
		c.ID = playerID
		spawnX, spawnY := room.getRandomSpawnPoint()
		c.Player = &Player{
			ID:          existingPlayer.ID,
			Name:        existingPlayer.Name,
			Color:       existingPlayer.Color,
			X:           spawnX, // New spawn point instead of old position
			Y:           spawnY, // New spawn point instead of old position
			Animation:   "idle",
			Direction:   existingPlayer.Direction,
			IsProtected: false, // No spawn protection on rejoin
		}
		// Set default direction if empty
		if c.Player.Direction == "" {
			c.Player.Direction = "right"
		}
		log.Printf("Player %s rejoining room %s with existing data - Name: %s, Color: %s, New spawn: (%.0f, %.0f)",
			playerID, code, c.Player.Name, c.Player.Color, spawnX, spawnY)
	} else {
		// Player wasn't in the room before, create new player data at a random chest spawn point
		c.ID = playerID
		spawnX, spawnY := room.getRandomSpawnPoint()
		c.Player = &Player{
			ID:        playerID,
			Name:      "Player",
			Color:     playerColors[rand.Intn(len(playerColors))],
			X:         spawnX,
			Y:         spawnY,
			Animation: "idle",
			Direction: "right",
		}
		log.Printf("Player %s joining room %s as new player", playerID, code)
	}

	c.RoomCode = code

	// Update the game state with the new position if player existed
	if playerExists {
		room.mutex.Lock()
		if p, exists := room.GameState.Players[playerID]; exists {
			p.X = c.Player.X
			p.Y = c.Player.Y
			p.IsProtected = c.Player.IsProtected
			p.ProtectionExpiry = c.Player.ProtectionExpiry
		}
		room.mutex.Unlock()
	}

	room.register <- c

	// Send success response with player and terrain data
	response := struct {
		Type     string       `json:"type"`
		PlayerID string       `json:"playerId"`
		Player   *Player      `json:"player"`
		Rejoined bool         `json:"rejoined"`
		MapData  game.MapData `json:"mapData"`
	}{
		Type:     "rejoinedRoom",
		PlayerID: c.ID,
		Player:   c.Player,
		Rejoined: playerExists,
		MapData:  room.MapData,
	}

	data, _ := json.Marshal(response)
	c.Send <- data
}

func (c *Client) handleRejoinDashboard(code string) {
	room, exists := roomManager.GetRoom(code)
	if !exists {
		response := struct {
			Type  string `json:"type"`
			Error string `json:"error"`
		}{
			Type:  "error",
			Error: "Room not found",
		}
		data, _ := json.Marshal(response)
		c.Send <- data
		return
	}

	// Mark this client as a dashboard
	c.IsDashboard = true
	c.RoomCode = code

	// Register the dashboard with the room
	room.register <- c

	// Send success response
	response := struct {
		Type      string       `json:"type"`
		RoomCode  string       `json:"roomCode"`
		MapData   game.MapData `json:"mapData"`
		GameState GameState    `json:"gameState"`
	}{
		Type:      "rejoinedDashboard",
		RoomCode:  code,
		MapData:   room.MapData,
		GameState: room.GameState,
	}
	data, _ := json.Marshal(response)
	c.Send <- data

	// Send initial game state with MapData
	room.sendGameStateToClient(c)
}

func (c *Client) handleUpdatePosition(x, y float64, animation string, direction string, gunRotation float64, gunFlipped bool, currentGun int) {
	if c.RoomCode == "" || c.Player == nil {
		return
	}

	room, exists := roomManager.GetRoom(c.RoomCode)
	if !exists {
		return
	}

	room.updatePlayerPosition(c.ID, x, y, animation, direction, gunRotation, gunFlipped, currentGun)
}

func (c *Client) handleBulletSpawn(bulletID string, x, y, velocityX, velocityY, angle float64, gunType int) {
	if c.RoomCode == "" || c.Player == nil {
		return
	}

	room, exists := roomManager.GetRoom(c.RoomCode)
	if !exists {
		return
	}

	// Remove spawn protection when player shoots
	room.mutex.Lock()
	if player, exists := room.GameState.Players[c.ID]; exists && player.IsProtected {
		player.IsProtected = false
		player.ProtectionExpiry = time.Time{}
	}
	room.mutex.Unlock()

	// Broadcast bullet spawn to all other clients
	room.broadcastToOthers(c.ID, struct {
		Type      string  `json:"type"`
		BulletID  string  `json:"bulletId"`
		OwnerID   string  `json:"ownerId"`
		X         float64 `json:"x"`
		Y         float64 `json:"y"`
		VelocityX float64 `json:"velocityX"`
		VelocityY float64 `json:"velocityY"`
		Angle     float64 `json:"angle"`
		GunType   int     `json:"gunType"`
	}{
		Type:      "bulletSpawn",
		BulletID:  bulletID,
		OwnerID:   c.ID,
		X:         x,
		Y:         y,
		VelocityX: velocityX,
		VelocityY: velocityY,
		Angle:     angle,
		GunType:   gunType,
	})
}

func (c *Client) handleBulletDestroy(bulletID string) {
	if c.RoomCode == "" || c.Player == nil {
		return
	}

	room, exists := roomManager.GetRoom(c.RoomCode)
	if !exists {
		return
	}

	// Broadcast bullet destruction to all other clients
	room.broadcastToOthers(c.ID, struct {
		Type     string `json:"type"`
		BulletID string `json:"bulletId"`
	}{
		Type:     "bulletDestroy",
		BulletID: bulletID,
	})
}

func (c *Client) handlePlayerHit(bulletID, targetPlayerID string, damage int, health float64, isDead bool) {
	if c.RoomCode == "" || c.Player == nil {
		return
	}

	room, exists := roomManager.GetRoom(c.RoomCode)
	if !exists {
		return
	}

	// Broadcast hit to all clients for visual effects
	room.broadcastToAll(struct {
		Type           string  `json:"type"`
		BulletID       string  `json:"bulletId"`
		TargetPlayerID string  `json:"targetPlayerId"`
		Damage         int     `json:"damage"`
		Health         float64 `json:"health"`
		IsDead         bool    `json:"isDead"`
	}{
		Type:           "playerHit",
		BulletID:       bulletID,
		TargetPlayerID: targetPlayerID,
		Damage:         damage,
		Health:         health,
		IsDead:         isDead,
	})
}

func (c *Client) handlePlayerDeath(playerID string) {
	if c.RoomCode == "" || c.Player == nil {
		return
	}

	room, exists := roomManager.GetRoom(c.RoomCode)
	if !exists {
		return
	}

	// Broadcast death to all clients
	room.broadcastToAll(struct {
		Type     string `json:"type"`
		PlayerID string `json:"playerId"`
	}{
		Type:     "playerDeath",
		PlayerID: playerID,
	})
}

func (c *Client) handlePlayerRespawn(playerID string, x, y float64) {
	if c.RoomCode == "" || c.Player == nil {
		return
	}

	room, exists := roomManager.GetRoom(c.RoomCode)
	if !exists {
		return
	}

	// Use server-determined spawn point (ignore client's x, y)
	spawnX, spawnY := room.getRandomSpawnPoint()

	// Update player position on server with 3 seconds spawn protection
	room.mutex.Lock()
	if player, exists := room.GameState.Players[playerID]; exists {
		player.X = spawnX
		player.Y = spawnY
		player.IsProtected = true                                 // Give spawn protection
		player.ProtectionExpiry = time.Now().Add(3 * time.Second) // 3 seconds of protection
	}
	room.mutex.Unlock()

	// Broadcast respawn to all clients with server-determined position
	room.broadcastToAll(struct {
		Type     string  `json:"type"`
		PlayerID string  `json:"playerId"`
		X        float64 `json:"x"`
		Y        float64 `json:"y"`
	}{
		Type:     "playerRespawn",
		PlayerID: playerID,
		X:        spawnX,
		Y:        spawnY,
	})
}

func (c *Client) handleUpdateGamePhase(phase string) {
	if !c.IsDashboard || c.RoomCode == "" {
		return
	}

	room, exists := roomManager.GetRoom(c.RoomCode)
	if !exists {
		return
	}

	room.mutex.Lock()
	room.GameState.GamePhase = phase
	room.mutex.Unlock()

	log.Printf("Room %s game phase updated to: %s", c.RoomCode, phase)
}

func (c *Client) handleGetState() {
	if c.RoomCode == "" {
		log.Printf("Client %s requested state but not in a room", c.ID)
		return
	}

	room, exists := roomManager.GetRoom(c.RoomCode)
	if !exists {
		log.Printf("Client %s requested state but room %s not found", c.ID, c.RoomCode)
		return
	}

	log.Printf("Sending game state to client %s in room %s", c.ID, c.RoomCode)
	room.sendGameStateToClient(c)
}

func (c *Client) handleUpdateScore(correctAnswers int, questionsAttempted int, kills int) {
	if c.RoomCode == "" || c.Player == nil {
		return
	}

	room, exists := roomManager.GetRoom(c.RoomCode)
	if !exists {
		return
	}

	room.mutex.Lock()
	defer room.mutex.Unlock()

	// Update player stats
	if player, exists := room.GameState.Players[c.ID]; exists {
		player.CorrectAnswers = correctAnswers
		player.QuestionsAttempted = questionsAttempted
		player.Kills = kills

		// Calculate score: correctAnswers * 3 + kills
		score := correctAnswers*3 + kills
		room.GameState.Score[c.ID] = score

		log.Printf("Player %s score updated: correctAnswers=%d, questionsAttempted=%d, kills=%d, score=%d",
			c.ID, correctAnswers, questionsAttempted, kills, score)
	}
}

func (c *Client) handleEndGame() {
	// Only dashboard can end the game
	if !c.IsDashboard {
		log.Printf("Non-dashboard client tried to end game")
		return
	}

	room, exists := roomManager.GetRoom(c.RoomCode)
	if !exists {
		log.Printf("Room not found for end game: %s", c.RoomCode)
		return
	}

	// Update game phase to ended
	room.mutex.Lock()
	room.GameState.GamePhase = "ended"
	room.mutex.Unlock()

	// Broadcast game ended to all clients
	response := struct {
		Type      string         `json:"type"`
		GamePhase string         `json:"gamePhase"`
		Scores    map[string]int `json:"scores"`
	}{
		Type:      "gameEnded",
		GamePhase: "ended",
		Scores:    room.GameState.Score,
	}

	data, _ := json.Marshal(response)
	room.broadcastToAll(data)

	log.Printf("Game ended in room %s", c.RoomCode)
}

func (c *Client) handleUpdateTimer(timer int) {
	// Only dashboard can update timer
	if !c.IsDashboard {
		return
	}

	room, exists := roomManager.GetRoom(c.RoomCode)
	if !exists {
		return
	}

	// Update timer in game state
	room.mutex.Lock()
	room.GameState.Timer = timer
	room.mutex.Unlock()
}

func main() {
	// WebSocket endpoint
	http.HandleFunc("/ws", enableCORS(handleWebSocket))

	// Health check endpoint
	http.HandleFunc("/health", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	log.Println("Game server starting on :8080")
	log.Println("WebSocket endpoint: ws://localhost:8080/ws")
	log.Println("60Hz tick rate enabled")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
