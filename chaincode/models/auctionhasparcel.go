package models

type AuctionHasParcel struct {
	AuctionID string `json:"auction_id"`
	ParcelID  int    `json:"parcel_id"`
}
