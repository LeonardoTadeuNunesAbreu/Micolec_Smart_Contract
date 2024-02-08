package models

import "time"

type State string

//state (Pending/Auction/Delivery/Delivered)

const (
	ParcelStatePending   Status = "Pending"
	ParcelStateAuction   Status = "Auction"
	ParcelStateDelivery  Status = "Delivery"
	ParcelStateDelivered Status = "Delivered"
)

type Parcel struct {
	ID                   int       `json:"id"`
	State                State     `json:"state"`
	AddedToPlatform      time.Time `json:"added_to_platform"`
	RequiredDeliveryDate time.Time `json:"required_delivery_date"`
	PickupPostalArea     string    `json:"pickup_postal_area"`
	DeliveryPostalArea   string    `json:"delivery_postal_area"`
	NotifiedCeOption     bool      `json:"notified_ce_option"`
	BitcircleReward      int       `json:"bitcircle_reward"`
	Weight               string    `json:"weight"`
	Volumes              int       `json:"volume"`
	LogisticOperatorId   int       `json:"logistic_operator_id"`
	EndCustomerId        int       `json:"end_customer_id"`
}
