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
		TickRate:   time.Second / 30, // 30Hz
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
		log.Printf("Player %s joined room %s", client.ID, r.Code)
	}

	// Send initial game state to new client
	r.sendGameStateToClient(client)
}

func (r *Room) removeClient(client *Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if client.IsDashboard {
		r.Dashboard = nil
		log.Printf("Dashboard disconnected from room %s", r.Code)
	} else {
		delete(r.Players, client.ID)
		delete(r.GameState.Players, client.ID)
		log.Printf("Player %s left room %s", client.ID, r.Code)
	}

	close(client.Send)
}

func (r *Room) broadcastGameState() {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	state := struct {
		Type      string        `json:"type"`
		GameState GameState     `json:"gameState"`
		MapData   game.MapData  `json:"mapData"`
		Timestamp int64         `json:"timestamp"`
	}{
		Type:      "gameUpdate",
		GameState: r.GameState,
		MapData:   r.MapData,
		Timestamp: time.Now().UnixMilli(),
	}

	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("Error marshaling game state: %v", err)
		return
	}

	r.broadcast <- data
}

func (r *Room) sendGameStateToClient(client *Client) {
	state := struct {
		Type      string        `json:"type"`
		RoomCode  string        `json:"roomCode"`
		GameState GameState     `json:"gameState"`
		MapData   game.MapData  `json:"mapData"`
		Timestamp int64         `json:"timestamp"`
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

func (r *Room) updatePlayerPosition(playerID string, x, y float64, animation string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if player, exists := r.GameState.Players[playerID]; exists {
		player.X = x
		player.Y = y
		player.Animation = animation
		r.LastUpdate = time.Now()
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

	case "joinRoom":
		var data struct {
			Code string `json:"code"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(msg.Content, &data); err != nil {
			log.Printf("Error parsing joinRoom message: %v", err)
			return
		}
		c.handleJoinRoom(data.Code, data.Name)

	case "updatePosition":
		var data struct {
			X         float64 `json:"x"`
			Y         float64 `json:"y"`
			Animation string  `json:"animation"`
		}
		if err := json.Unmarshal(msg.Content, &data); err != nil {
			log.Printf("Error parsing updatePosition message: %v", err)
			return
		}
		c.handleUpdatePosition(data.X, data.Y, data.Animation)

	case "updateGamePhase":
		var data struct {
			Phase string `json:"phase"`
		}
		if err := json.Unmarshal(msg.Content, &data); err != nil {
			log.Printf("Error parsing updateGamePhase message: %v", err)
			return
		}
		c.handleUpdateGamePhase(data.Phase)
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

	// Create player
	c.Player = &Player{
		ID:        c.ID,
		Name:      playerName,
		Color:     playerColors[rand.Intn(len(playerColors))],
		X:         float64(rand.Intn(room.MapData.Width)),
		Y:         float64(rand.Intn(room.MapData.Height)),
		Animation: "idle",
	}

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

func (c *Client) handleUpdatePosition(x, y float64, animation string) {
	if c.RoomCode == "" || c.Player == nil {
		return
	}

	room, exists := roomManager.GetRoom(c.RoomCode)
	if !exists {
		return
	}

	room.updatePlayerPosition(c.ID, x, y, animation)
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
	log.Println("30Hz tick rate enabled")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
