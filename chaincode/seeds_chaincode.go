package micolec

import (
	"encoding/json"
	"fmt"
	"micolec/chaincode/models"
	"net/http"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

func (s *AuctionSmartContract) SeedParcel(stub shim.ChaincodeStubInterface, parcels []models.Parcel) pb.Response {
	fmt.Println("Seed Parcels invoke")
	// Create and store parcel entity

	for _, parcel := range parcels {

		parcelKey, err := s.CreateCompositeKey(stub, EntityParcel, []string{fmt.Sprint(parcel.ID)})
		if err != nil {
			return shim.Error(err.Error())
		}

		if parcel.State == models.State(models.ParcelStatePending) ||
			parcel.State == models.State(models.ParcelStateAuction) ||
			parcel.State == models.State(models.ParcelStateDelivery) ||
			parcel.State == models.State(models.ParcelStateDelivered) {
			// Check if parcel already exists
			if parcelRecordExists, err := s.EntityRecordExists(stub, parcelKey); err != nil {
				return shim.Error(err.Error())
			} else if parcelRecordExists {
				return shim.Error("Record Already Exists")
			}

			// Insert Parcel on the blockchain
			jsonData, err := json.Marshal(parcel)
			if err != nil {
				return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
			}
			_, err = s.UpsertEntityRecord(stub, parcelKey, jsonData)
			if err != nil {
				return shim.Error(err.Error())
			}
		} else {
			return shim.Error(fmt.Sprint("Parcel ", parcel.ID, " have an invalid State: ", parcel.State))
		}
	}

	return shim.Success(nil)
}

func (s *AuctionSmartContract) SeedAuction(stub shim.ChaincodeStubInterface, auctions []models.Auction, auctionsHasParcels []models.AuctionHasParcel) pb.Response {
	fmt.Println("Seed Auctions invoke")
	for _, auction := range auctions {
		var auctionKey string
		auctionKey, err := s.CreateCompositeKey(stub, EntityAuction, []string{fmt.Sprint(auction.ID)})
		if err != nil {
			return shim.Error(err.Error())
		}

		dataAuction, err := json.Marshal(auction)
		if err != nil {
			return shim.Error(err.Error())
		}

		_, err = s.UpsertEntityRecord(stub, auctionKey, dataAuction)
		if err != nil {
			return shim.Error(err.Error())
		}
	}
	for _, auctionHasParcel := range auctionsHasParcels {
		var auctionHasParcelKey string
		auctionHasParcelKey, err := s.CreateCompositeKey(stub, EntityAuctionHasParcel, []string{fmt.Sprint(auctionHasParcel.AuctionID), fmt.Sprint(auctionHasParcel.ParcelID)})
		if err != nil {
			return shim.Error(err.Error())
		}
		jsonDataAuctionHasParcel, err := json.Marshal(auctionHasParcel)
		if err != nil {
			return shim.Error(err.Error())
		}
		_, err = s.UpsertEntityRecord(stub, auctionHasParcelKey, jsonDataAuctionHasParcel)
		if err != nil {
			return shim.Error(err.Error())
		}
	}

	return shim.Success(nil)
}

func (s *AuctionSmartContract) SeedWinningBid(stub shim.ChaincodeStubInterface, bids []models.Bid) pb.Response {
	fmt.Println("Seed Bids invoke")
	for _, bid := range bids {
		var bidKey string
		bidKey, err := s.CreateCompositeKey(stub, EntityBid, []string{fmt.Sprint(bid.ID), fmt.Sprint(bid.AuctionID)})
		if err != nil {
			return shim.Error(err.Error())
		}

		dataBid, err := json.Marshal(bid)
		if err != nil {
			return shim.Error(err.Error())
		}

		_, err = s.UpsertEntityRecord(stub, bidKey, dataBid)
		if err != nil {
			return shim.Error(err.Error())
		}
	}
	return shim.Success(nil)
}
