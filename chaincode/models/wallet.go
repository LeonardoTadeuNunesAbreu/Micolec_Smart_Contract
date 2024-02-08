package models

import "time"

type Wallet struct {
	ParticipantId int       `json:"participant_id"`
	Balance       int       `json:"balance"`
	UsableBalance int       `json:"usable_balance"`
	LastMovement  time.Time `json:"last_movement"`
}
