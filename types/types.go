package types

import "time"

type SensorData struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	AccelX    float64   `json:"accel_x"`
	AccelY    float64   `json:"accel_y"`
	AccelZ    float64   `json:"accel_z"`
	Latitude  *float64  `json:"latitude"`
	Longitude *float64  `json:"longitude"`
}

type SensorDataRequest struct {
	Size   int       `json:"size" form:"size"`
	Order  string    `json:"order" form:"order"`
	Offset *time.Time `json:"offset" form:"offset"`
}
