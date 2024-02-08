package models

import "time"

type BitcircleTransaction struct {
	ID                    int       `json:"id"`
	SenderParticipantId   int       `json:"sender_participant_id"`
	ReceiverParticipantId int       `json:"receiver_participant_id"`
	BitcircleAmount       int       `json:"bitcircle_amount"`
	Description           string    `json:"description"`
	Date                  time.Time `json:"date"`
	IsReward              bool      `json:"is_reward"`
}
