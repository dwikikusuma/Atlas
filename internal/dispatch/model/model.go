package model

type RideDispatchedEvent struct {
	RideID      string  `json:"ride_id"`
	PassengerID string  `json:"passenger_id"`
	DriverID    string  `json:"driver_id"`
	PickupLat   float64 `json:"pickup_lat"`
	PickupLong  float64 `json:"pickup_long"`
	Timestamp   int64   `json:"timestamp"`
}
