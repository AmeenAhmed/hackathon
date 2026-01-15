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
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Color       string  `json:"color"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Animation   string  `json:"animation"`
	Direction   string  `json:"direction"`
	GunRotation float64 `json:"gunRotation"`
	GunFlipped  bool    `json:"gunFlipped"`
	CurrentGun  int     `json:"currentGun"`
	IsProtected bool    `json:"isProtected"`
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
		MapData: game.GenerateMap(),
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
			r.broadcastGameState()
		case <-r.stopTicker:
			return
		}
	}
}

func (r *Room) addClient(client *Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if client.IsDashboard {
		r.Dashboard = client
		log.Printf("Dashboard connected to room %s", r.Code)
	} else {
		r.Players[client.ID] = client
		r.GameState.Players[client.ID] = client.Player
		log.Printf("Player %s joined room %s (total players: %d)", client.ID, r.Code, len(r.GameState.Players))
	}

	// Don't send initial state immediately - client will request it when ready via getState
	// This prevents timing issues where the client isn't ready to receive the state yet
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

	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("Error marshaling initial state: %v", err)
		return
	}

	select {
	case client.Send <- data:
	default:
		log.Printf("Client %s send buffer full", client.ID)
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
		select {
		case client.Send <- message:
		default:
			log.Printf("Player %s send buffer full", client.ID)
		}
	}
}

func (r *Room) updatePlayerPosition(playerID string, x, y float64, animation string, direction string, gunRotation float64, gunFlipped bool, currentGun int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if player, exists := r.GameState.Players[playerID]; exists {
		// Debug: Log before and after update
		oldX, oldY := player.X, player.Y
		player.X = x
		player.Y = y
		player.Animation = animation
		player.Direction = direction
		player.GunRotation = gunRotation
		player.GunFlipped = gunFlipped
		player.CurrentGun = currentGun
		r.LastUpdate = time.Now()

		// Verify the update actually happened
		if oldX != x || oldY != y {
			log.Printf("Player %s moved from (%.0f,%.0f) to (%.0f,%.0f), animation: %s",
				playerID, oldX, oldY, x, y, animation)
		}
	} else {
		log.Printf("WARNING: Player %s not found in GameState.Players", playerID)
	}
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

	// Create player at center with some randomness
	c.Player = &Player{
		ID:        c.ID,
		Name:      playerName,
		Color:     playerColors[rand.Intn(len(playerColors))],
		X:         float64(room.MapData.Width/2 + rand.Intn(200) - 100),
		Y:         float64(room.MapData.Height/2 + rand.Intn(200) - 100),
		Animation: "idle",
		Direction: "right",
	}

	// log.Printf("Created player - ID: %s, Name: %s, Color: %s, Position: (%.0f, %.0f)",
	// 	c.Player.ID, c.Player.Name, c.Player.Color, c.Player.X, c.Player.Y)

	c.RoomCode = code
	room.register <- c

	// Send success response
	response := struct {
		Type     string  `json:"type"`
		PlayerID string  `json:"playerId"`
		Player   *Player `json:"player"`
	}{
		Type:     "joinedRoom",
		PlayerID: c.ID,
		Player:   c.Player,
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
		// Reuse existing player data but update the client ID
		c.ID = playerID
		c.Player = &Player{
			ID:          existingPlayer.ID,
			Name:        existingPlayer.Name,
			Color:       existingPlayer.Color,
			X:           existingPlayer.X,
			Y:           existingPlayer.Y,
			Animation:   existingPlayer.Animation,
			Direction:   existingPlayer.Direction,
			IsProtected: existingPlayer.IsProtected,
		}
		// Set default direction if empty
		if c.Player.Direction == "" {
			c.Player.Direction = "right"
		}
		log.Printf("Player %s rejoining room %s with existing data - Name: %s, Color: %s",
			playerID, code, c.Player.Name, c.Player.Color)
	} else {
		// Player wasn't in the room before, create new player data at center
		c.ID = playerID
		c.Player = &Player{
			ID:        playerID,
			Name:      "Player",
			Color:     playerColors[rand.Intn(len(playerColors))],
			X:         float64(room.MapData.Width/2 + rand.Intn(200) - 100),
			Y:         float64(room.MapData.Height/2 + rand.Intn(200) - 100),
			Animation: "idle",
			Direction: "right",
		}
		log.Printf("Player %s joining room %s as new player", playerID, code)
	}

	c.RoomCode = code
	room.register <- c

	// Send success response with player data
	response := struct {
		Type     string  `json:"type"`
		PlayerID string  `json:"playerId"`
		Player   *Player `json:"player"`
		Rejoined bool    `json:"rejoined"`
	}{
		Type:     "rejoinedRoom",
		PlayerID: c.ID,
		Player:   c.Player,
		Rejoined: playerExists,
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

	// Send current game state to dashboard
	room.mutex.RLock()
	gameStateCopy := room.GameState
	room.mutex.RUnlock()

	response := struct {
		Type      string     `json:"type"`
		GameState *GameState `json:"gameState"`
	}{
		Type:      "rejoinedDashboard",
		GameState: &gameStateCopy,
	}

	data, _ := json.Marshal(response)
	c.Send <- data

	log.Printf("Dashboard rejoined room %s", code)
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

	// Broadcast respawn to all clients
	room.broadcastToAll(struct {
		Type     string  `json:"type"`
		PlayerID string  `json:"playerId"`
		X        float64 `json:"x"`
		Y        float64 `json:"y"`
	}{
		Type:     "playerRespawn",
		PlayerID: playerID,
		X:        x,
		Y:        y,
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
