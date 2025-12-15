package model

type DebitBalanceEvent struct {
	Amount    float64
	UserID    string
	Reference string
}

type LocationEvent struct {
	UserID    string  `json:"user_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp string  `json:"timestamp"`
}
