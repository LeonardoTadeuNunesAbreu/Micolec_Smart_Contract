package models

import "time"

type Status string

const (
	BitStatusLowerBid  Status = "LowerBid"
	BitStatusOutBidded Status = "OutBidded"
)

type Bid struct {
	ID              string    `json:"id"`
	Date            time.Time `json:"date"`
	BitcircleAmount int       `json:"bitcircle_amount,omitempty"`
	MoneyAmount     float32   `json:"money_amount"`
	Status          Status    `json:"status"`
	Winner          bool      `json:"winner"`
	AuctionID       string    `json:"auction_id"`
	CourierID       int       `json:"courier_id"`
}
