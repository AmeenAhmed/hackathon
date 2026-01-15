package game

// MapSize increased to 200 for larger battle royale maps
// This gives us 40,000 total tiles for 50-100 players
const MapSize = 200

type MapObject struct {
	ID       string `json:"id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	IsPicked bool   `json:"isPicked"`
}

type MapData struct {
	Width      int                   `json:"width"`
	Height     int                   `json:"height"`
	MapObjects []MapObject           `json:"mapObjects"`
	Terrain    [MapSize][MapSize]int `json:"terrain"`
}
