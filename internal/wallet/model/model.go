package model

type DebitBalanceEvent struct {
	Amount    float64
	UserID    string
	Reference string
}
