package types

type TrainData struct {
	ID       int64 `json:"id" bsodn:"_id"`
	Position struct {
		Location   Location `json:"location"`
		LastUpdate int64    `json:"last_update"`
	} `json:"position"`
	ForceStop bool `json:"force_stop"`
}
