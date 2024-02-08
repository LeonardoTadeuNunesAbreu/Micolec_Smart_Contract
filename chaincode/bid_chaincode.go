package micolec

import (
	"encoding/json"
	"fmt"
	"micolec/chaincode/models"
	"net/http"
	"sort"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

// ** -----------------------------------------------------
// ** BID
// ** -> START
// ** -----------------------------------------------------

/*
CORE_PEER_ADDRESS=127.0.0.1:7051 peer chaincode invoke -o 127.0.0.1:7050 -C ch1 -n mycc -c '{"Args":["ParcelDeliveryBidingRequest", "1", "1", "500.50", "100.25"]}'
*/

func GetBids(stub shim.ChaincodeStubInterface) ([]models.Bid, error) {
	bidsIterator, err := stub.GetStateByPartialCompositeKey(string(EntityBid), []string{})
	if err != nil {
		return nil, err
	}
	defer bidsIterator.Close()

	var bids []models.Bid
	for bidsIterator.HasNext() {
		response, err := bidsIterator.Next()
		if err != nil {
			return nil, err
		}
		var bid models.Bid
		err = json.Unmarshal(response.Value, &bid)
		if err != nil {
			return nil, err
		}
		bids = append(bids, bid)
	}

	return bids, nil
}

func (s *AuctionSmartContract) GetLastBids(stub shim.ChaincodeStubInterface) ([]models.Bid, error) {
	allBids, err := GetBids(stub)
	if err != nil {
		return nil, err
	}

	lastBids := make([]models.Bid, 0)

	// Sort bids by date in descending order
	sort.SliceStable(allBids, func(i, j int) bool {
		return allBids[i].Date.After(allBids[j].Date)
	})

	// Get the last bids (assuming you want the top N bids)
	N := 5 // Number of last bids to retrieve
	if len(allBids) < N {
		N = len(allBids)
	}

	lastBids = allBids[:N]

	return lastBids, nil
}

func (s *AuctionSmartContract) GetBidsForAuction(stub shim.ChaincodeStubInterface, auctionID string) pb.Response {
	auctionKey, err := s.CreateCompositeKey(stub, EntityAuction, []string{auctionID})
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	exists, err := s.EntityRecordExists(stub, auctionKey)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	if !exists {
		return shim.Success(createErrorResponse(http.StatusNotFound, fmt.Sprintf("auction %v does not exist", auctionID)))
	}

	allBids, err := GetBids(stub)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	// Create a slice to hold the bid objects
	var bids []models.Bid
	for _, bid := range allBids {
		if bid.AuctionID == auctionID {
			bids = append(bids, bid)
		}
	}

	// Convert the slice of bids to JSON
	bidJSON, err := json.Marshal(bids)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(bidJSON)
}

func (s *AuctionSmartContract) GetParticipantBids(stub shim.ChaincodeStubInterface, userID int) pb.Response {
	allBids, err := GetBids(stub)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	var bids []models.Bid
	for _, bid := range allBids {
		if bid.CourierID == userID {
			bids = append(bids, bid)
		}
	}

	// Convert the slice of bids to JSON
	bidsJSON, err := json.Marshal(bids)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(bidsJSON)
}

func (s *AuctionSmartContract) ParcelDeliveryBidingRequest(stub shim.ChaincodeStubInterface, bidID string, auctionID string, moneyAmount float32, bitcircleAmount int, participantId int, date time.Time) pb.Response {
	transactionError := false

	// Start a new transaction
	transactionId, err := s.StartTransaction(stub)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	// Roolback in case of error Or Commit in case of success
	defer s.CloseTransaction(stub, transactionId, transactionError)

	if moneyAmount < 0 {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusNotFound, fmt.Sprintf("The money ammount most be higher or equal than 0")))
	}

	if bitcircleAmount < 0 {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusNotFound, fmt.Sprintf("The bitcircle ammount most be higher or equal than 0")))
	}

	auctionKey, err := s.CreateCompositeKey(stub, EntityAuction, []string{fmt.Sprint(auctionID)})
	if err != nil {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	auctionJSON, err := s.ReadEntity(stub, auctionKey)
	if err != nil {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusNotFound, err.Error()))
	}

	var auction models.Auction
	err = json.Unmarshal(auctionJSON, &auction)
	if err != nil {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusNotFound, err.Error()))
	}

	if moneyAmount > auction.MaximumAcceptedLicitation {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusNotFound, fmt.Sprintf("The bid amount cannot exceed the maximum limit set for this auction. Please enter a lower bid amount.")))
	}

	// Get the current time
	currentTime := getCurrentTime()
	if auction.State != models.AuctionState(models.AuctionOpen) || auction.EndDate.Before(currentTime) {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusNotFound, fmt.Sprintf("This auction is already closed")))
	}

	// Check if the new bid is the lowest bid
	bidsIterator, err := stub.GetStateByPartialCompositeKey(string(EntityBid), []string{})
	if err != nil {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	defer bidsIterator.Close()

	var lowestBid models.Bid
	var lowestBidKey string
	for bidsIterator.HasNext() {
		bidResponse, err := bidsIterator.Next()
		if err != nil {
			transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		var bid models.Bid
		err = json.Unmarshal(bidResponse.Value, &bid)
		if err != nil {
			transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		fmt.Println("Iterate Bid: ", bid.ID)
		fmt.Println("Auction: ", bid.AuctionID)

		// Checks if this bid is the current winner and if input money amount is less than or equal to winner bid amount
		if bid.AuctionID == auctionID && bid.Status == models.BitStatusLowerBid {
			lowestBid = bid
			lowestBidKey = bidResponse.Key
			// If it finds the winner bid, there is no need to continue iterating
			break
		}
	}

	fmt.Println("lowestBid: ", lowestBidKey)

	// Check if new bid is lower than lowest bid
	if lowestBid.ID != "" && (moneyAmount > lowestBid.MoneyAmount || (moneyAmount == lowestBid.MoneyAmount && bitcircleAmount <= lowestBid.BitcircleAmount)) {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusBadRequest, fmt.Sprint("The current winner bid have ", lowestBid.MoneyAmount, "€ and ", lowestBid.BitcircleAmount, " bitcircles. The bid amount cannot be higher than the current lowest bid. If the bid amount is equal to the lowest bid, please increase the number of Bitcircles instead.")))
	}

	if lowestBid.CourierID == participantId {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusBadRequest, fmt.Sprint("You cannot place a new bid because you are the owner of the current winning bid.")))
	}

	// Set previous lowest bid to "Outbidded" status and not a winner
	if lowestBidKey != "" {
		var prevBid models.Bid
		prevBidByte, err := stub.GetState(lowestBidKey)
		if err != nil {
			transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}
		err = json.Unmarshal(prevBidByte, &prevBid)
		if err != nil {
			transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		prevBid.Status = models.BitStatusOutBidded

		err = s.RefundBitcirclesForAuction(stub, prevBid.CourierID, prevBid.BitcircleAmount, prevBid.MoneyAmount, prevBid.AuctionID, false)
		if err != nil {
			transactionError = true
			return shim.Success(createErrorResponse(http.StatusBadRequest, err.Error()))
		}

		jsonDataPrevBid, err := json.Marshal(prevBid)
		if err != nil {
			transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}
		fmt.Println("Update Bid")
		_, err = s.UpsertEntityRecord(stub, lowestBidKey, jsonDataPrevBid)
		if err != nil {
			transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

	}

	// Create new bid
	bid := models.Bid{
		ID:              bidID,
		Date:            date,
		BitcircleAmount: bitcircleAmount,
		MoneyAmount:     moneyAmount,
		Status:          models.BitStatusLowerBid,
		Winner:          false,
		AuctionID:       auctionID,
		CourierID:       participantId,
	}

	bidCompositeKey, err := s.CreateCompositeKey(stub, EntityBid, []string{fmt.Sprint(bidID), fmt.Sprint(auctionID)})
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	jsonDataBid, err := json.Marshal(bid)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	err = s.ReserveBitcirclesForBid(stub, participantId, bid.BitcircleAmount, bid.MoneyAmount, bid.AuctionID, false)
	if err != nil {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusBadRequest, err.Error()))
	}

	_, err = s.UpsertEntityRecord(stub, bidCompositeKey, jsonDataBid)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(jsonDataBid)
}

func (s *AuctionSmartContract) ReadBids(stub shim.ChaincodeStubInterface) pb.Response {
	// Create iterator for all bid entities
	iterator, err := stub.GetStateByPartialCompositeKey(string(EntityBid), []string{})
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	defer iterator.Close()

	// Create a slice to hold the bid objects
	var bids []models.Bid

	// Loop through all bids and append them to the slice
	for iterator.HasNext() {
		response, err := iterator.Next()
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		var bid models.Bid
		err = json.Unmarshal(response.Value, &bid)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		bids = append(bids, bid)
	}

	// Convert the slice of bids to JSON
	bidJSON, err := json.Marshal(bids)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(bidJSON)
}

func (s *AuctionSmartContract) DeleteAllBids(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("DeleteAllBids Invoke")

	// Create an iterator for all bid entities
	iterator, err := stub.GetStateByPartialCompositeKey(string(EntityBid), []string{})
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	defer iterator.Close()

	// Iterate through all bids and delete them
	for iterator.HasNext() {
		response, err := iterator.Next()
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		// Delete the bid by its composite key
		err = stub.DelState(response.Key)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}
	}

	return shim.Success(nil)
}

// ** -----------------------------------------------------
// ** BID
// ** -> END
// ** -----------------------------------------------------
