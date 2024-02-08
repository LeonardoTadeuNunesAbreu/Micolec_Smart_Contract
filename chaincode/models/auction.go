package models

import "time"

type AuctionState string

const (
	AuctionClosedNoBids Status = "CLOSED NO BIDS"
	AuctionClosedBids   Status = "CLOSED"
	AuctionOpen         Status = "OPEN"
)

type Auction struct {
	ID                        string       `json:"id"`
	StartDate                 time.Time    `json:"start_date"`
	EndDate                   time.Time    `json:"end_date"`
	MaximumAcceptedLicitation float32      `json:"maximum_accepted_licitation,omitempty"`
	State                     AuctionState `json:"state"`
	ParticipantId             int          `json:"participant_id"`
}
