package game

const MapSize = 100

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
