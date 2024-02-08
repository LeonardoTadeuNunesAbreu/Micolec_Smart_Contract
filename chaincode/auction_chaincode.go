package micolec

import (
	"encoding/json"
	"errors"
	"fmt"
	"micolec/chaincode/models"
	"net/http"
	"sort"
	"strings"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

func validateAuction(auction models.Auction) error {
	var errorMessages []string

	if !(auction.ID != "") {
		errorMessages = append(errorMessages, "ID cannot be empty")
	}

	if auction.State != models.AuctionState(models.AuctionOpen) {
		errorMessages = append(errorMessages, "Auction State when create most be 'OPEN'")
	}

	if auction.EndDate.Before(auction.StartDate) {
		return errors.New("End date cannot be before start date")
	}

	if !(auction.MaximumAcceptedLicitation > 0) {
		errorMessages = append(errorMessages, "MaximumAmount Higher than 0")
	}

	if len(errorMessages) > 0 {
		return errors.New(strings.Join(errorMessages, "\n"))
	}

	return nil
}

func GetAuctions(stub shim.ChaincodeStubInterface) ([]models.Auction, error) {
	auctionsIterator, err := stub.GetStateByPartialCompositeKey(string(EntityAuction), []string{})
	if err != nil {
		return nil, err
	}
	defer auctionsIterator.Close()

	var auctions []models.Auction
	for auctionsIterator.HasNext() {
		response, err := auctionsIterator.Next()
		if err != nil {
			return nil, err
		}
		var auction models.Auction
		err = json.Unmarshal(response.Value, &auction)
		if err != nil {
			return nil, err
		}
		auctions = append(auctions, auction)
	}

	return auctions, nil
}

func (s *AuctionSmartContract) GetLastAuctions(stub shim.ChaincodeStubInterface) ([]models.Auction, error) {
	allAuctions, err := GetAuctions(stub)
	if err != nil {
		return nil, err
	}

	lastAuctions := make([]models.Auction, 0)

	// Sort bids by date in descending order
	sort.SliceStable(allAuctions, func(i, j int) bool {
		return allAuctions[i].StartDate.After(allAuctions[j].StartDate)
	})

	// Get the last bids (assuming you want the top N bids)
	N := 5 // Number of last bids to retrieve
	if len(allAuctions) < N {
		N = len(allAuctions)
	}

	lastAuctions = allAuctions[:N]

	return lastAuctions, nil
}

func GetAuctionsHasParcel(stub shim.ChaincodeStubInterface) ([]models.AuctionHasParcel, error) {
	auctionsHasParcelIterator, err := stub.GetStateByPartialCompositeKey(string(EntityAuctionHasParcel), []string{})
	if err != nil {
		return nil, err
	}
	defer auctionsHasParcelIterator.Close()

	var auctionsHasParcels []models.AuctionHasParcel
	for auctionsHasParcelIterator.HasNext() {
		response, err := auctionsHasParcelIterator.Next()
		if err != nil {
			return nil, err
		}
		var auctionHasParcel models.AuctionHasParcel
		err = json.Unmarshal(response.Value, &auctionHasParcel)
		if err != nil {
			return nil, err
		}
		auctionsHasParcels = append(auctionsHasParcels, auctionHasParcel)
	}

	return auctionsHasParcels, nil
}

func (s *AuctionSmartContract) ParcelDeliveryAuctionStart(stub shim.ChaincodeStubInterface, parcels []models.AuctionHasParcel, auction models.Auction) pb.Response {
	fmt.Println("ParcelDeliveryAuctionStart Invoke")
	transactionError := false

	// Start a new transaction
	transactionId, err := s.StartTransaction(stub)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	// Rollback in case of error or Commit in case of success
	defer func() {
		if transactionError {
			if err := s.CloseTransaction(stub, transactionId, true); err != nil {
				fmt.Println("Error rolling back transaction:", err)
			}
		} else {
			if err := s.CloseTransaction(stub, transactionId, false); err != nil {
				fmt.Println("Error committing transaction:", err)
			}
		}
	}()

	// Validate auction
	err = validateAuction(auction)
	if err != nil {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusBadRequest, err.Error()))
	}

	if len(parcels) == 0 {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusBadRequest, "No parcel selected for the auction. Please choose a parcel to proceed."))
	}

	var auctionParcels []int
	// Process parcels
	for _, auctionHasParcel := range parcels {
		// Check if ParcelExists
		parcelKey, err := s.CreateCompositeKey(stub, EntityParcel, []string{fmt.Sprint(auctionHasParcel.ParcelID)})
		if err != nil {
			transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}
		entity, err := s.ReadEntity(stub, parcelKey)
		if err != nil {
			transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		var parcel models.Parcel
		json.Unmarshal(entity, &parcel)

		// ! Tem de ser adicionado assim que seja corrigido o tema a atomicidade das transações.
		// if parcel.LogisticOperatorId != auction.ParticipantId {
		// 	transactionError = true
		// 	return shim.Success(createErrorResponse(http.StatusInternalServerError, fmt.Sprint("The parcel with id ", auctionHasParcel.ParcelID, " do not belong to current user")))
		// }

		if parcel.State != models.State(models.ParcelStatePending) {
			transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, fmt.Sprint("The parcel with id ", auctionHasParcel.ParcelID, " is not on 'Pending' state.")))
		}

		parcel.State = models.State(models.ParcelStateAuction)
		dataParcel, err := json.Marshal(parcel)
		if err != nil {
			transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		s.UpsertEntityRecord(stub, parcelKey, dataParcel)
		// Create and store auctionHasParcel entity
		var auctionHasParcelKey string
		auctionHasParcelKey, err = s.CreateCompositeKey(stub, EntityAuctionHasParcel, []string{fmt.Sprint(auction.ID), fmt.Sprint(auctionHasParcel.ParcelID)})
		if err != nil {
			transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		jsonDataAuctionHasParcel, err := json.Marshal(auctionHasParcel)
		if err != nil {
			transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}
		_, err = s.UpsertEntityRecord(stub, auctionHasParcelKey, jsonDataAuctionHasParcel)
		if err != nil {
			transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		auctionParcels = append(auctionParcels, auctionHasParcel.ParcelID)
	}

	// Create and store auction entity
	var auctionKey string
	auctionKey, err = s.CreateCompositeKey(stub, EntityAuction, []string{fmt.Sprint(auction.ID)})
	if err != nil {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	dataAuction, err := json.Marshal(auction)
	if err != nil {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	_, err = s.UpsertEntityRecord(stub, auctionKey, dataAuction)
	if err != nil {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	var response struct {
		Auction models.Auction `json:"auction"`
		Parcels []int          `json:"parcels"`
	}

	response.Auction = auction
	response.Parcels = auctionParcels

	responseJson, err := json.Marshal(response)
	if err != nil {
		transactionError = true
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(responseJson)
}

func (s *AuctionSmartContract) GetAuctionByID(stub shim.ChaincodeStubInterface, id string) pb.Response {
	auctionKey, err := s.CreateCompositeKey(stub, EntityAuction, []string{fmt.Sprint(id)})
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	auctionJSON, err := s.ReadEntity(stub, auctionKey)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusNotFound, err.Error()))
	}

	var response struct {
		Auction models.Auction `json:"auction"`
		Parcels []int          `json:"parcels"`
		Bids    []models.Bid   `json:"bids"`
	}

	err = json.Unmarshal(auctionJSON, &response.Auction)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusNotFound, err.Error()))
	}

	response.Parcels, err = getParcelsForAuction(stub, response.Auction.ID)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	allBids, err := GetBids(stub)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	// Create a slice to hold the bid objects
	for _, bid := range allBids {
		if bid.AuctionID == response.Auction.ID {
			response.Bids = append(response.Bids, bid)
		}
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(responseJSON)
}

func (s *AuctionSmartContract) GetAuctionByParcelID(stub shim.ChaincodeStubInterface, parcelId int) pb.Response {
	parcelKey, err := s.CreateCompositeKey(stub, EntityParcel, []string{fmt.Sprint(parcelId)})

	exists, err := s.EntityRecordExists(stub, parcelKey)
	if !exists {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, "Parcel with id "+fmt.Sprint(parcelId)+" doesn't exist"))
	}

	allAuctionHasParcels, err := GetAuctionsHasParcel(stub)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	var filteredAuctionHasParcels []models.AuctionHasParcel
	for _, auctionHasParcel := range allAuctionHasParcels {
		if auctionHasParcel.ParcelID == parcelId {
			filteredAuctionHasParcels = append(filteredAuctionHasParcels, auctionHasParcel)
		}
	}

	var response []struct {
		Auction    models.Auction `json:"auction"`
		Parcels    []int          `json:"parcels"`
		WinningBid models.Bid     `json:"winning_bid"`
	}

	for _, filteredAuctionHasParcel := range filteredAuctionHasParcels {
		responseItem := struct {
			Auction    models.Auction `json:"auction"`
			Parcels    []int          `json:"parcels"`
			WinningBid models.Bid     `json:"winning_bid"`
		}{}
		auctionKey, err := s.CreateCompositeKey(stub, EntityAuction, []string{fmt.Sprint(filteredAuctionHasParcel.AuctionID)})
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		auctionJSON, err := s.ReadEntity(stub, auctionKey)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusNotFound, err.Error()))
		}

		err = json.Unmarshal(auctionJSON, &responseItem.Auction)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusNotFound, err.Error()))
		}

		responseItem.WinningBid, err = getWinningBidForAuction(stub, responseItem.Auction.ID)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}
		responseItem.Parcels, err = getParcelsForAuction(stub, responseItem.Auction.ID)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		response = append(response, responseItem)
	}

	// Convert the response slice to JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(responseJSON)
}

func (s *AuctionSmartContract) ReadAuctions(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("ReadAuctions Invoke")

	// Create iterator for all auction entities
	iterator, err := stub.GetStateByPartialCompositeKey(string(EntityAuction), []string{})
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	defer iterator.Close()

	// Create a slice to hold the auction and parcel data
	var response []struct {
		Auction models.Auction `json:"auction"`
		Parcels []int          `json:"parcels"`
		Bids    []models.Bid   `json:"bids"`
	}

	// Loop through all auctions and append them to the response slice
	for iterator.HasNext() {
		responseItem := struct {
			Auction models.Auction `json:"auction"`
			Parcels []int          `json:"parcels"`
			Bids    []models.Bid   `json:"bids"`
		}{}

		responseData, err := iterator.Next()
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		err = json.Unmarshal(responseData.Value, &responseItem.Auction)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		responseItem.Parcels, err = getParcelsForAuction(stub, responseItem.Auction.ID)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		allBids, err := GetBids(stub)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}
		// Create a slice to hold the bid objects
		for _, bid := range allBids {
			if bid.AuctionID == responseItem.Auction.ID {
				responseItem.Bids = append(responseItem.Bids, bid)
			}
		}

		response = append(response, responseItem)
	}

	// Convert the response slice to JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(responseJSON)
}

// ! Mudar para de ficheiro
func getParcelsForAuction(stub shim.ChaincodeStubInterface, auctionID string) ([]int, error) {
	// Create iterator for all auctionHasParcel entities with the given auctionID
	iterator, err := stub.GetStateByPartialCompositeKey(string(EntityAuctionHasParcel), []string{auctionID})
	if err != nil {
		return nil, err
	}
	defer iterator.Close()

	// Create a slice to hold the auctionHasParcel objects
	var parcels []int

	// Loop through all auctionHasParcel entities and append them to the slice
	for iterator.HasNext() {
		responseData, err := iterator.Next()
		if err != nil {
			return nil, err
		}

		var auctionHasParcel models.AuctionHasParcel
		err = json.Unmarshal(responseData.Value, &auctionHasParcel)
		if err != nil {
			return nil, err
		}

		parcels = append(parcels, auctionHasParcel.ParcelID)
	}

	return parcels, nil
}

// ! Mudar para de ficheiro
func getWinningBidForAuction(stub shim.ChaincodeStubInterface, auctionID string) (models.Bid, error) {
	// Create iterator for all auctionHasParcel entities with the given auctionID
	iterator, err := stub.GetStateByPartialCompositeKey(string(EntityBid), []string{})
	if err != nil {
		return models.Bid{}, err
	}
	defer iterator.Close()

	// Create a slice to hold the auctionHasParcel objects
	var bid models.Bid

	// Loop through all auctionHasParcel entities and append them to the slice
	for iterator.HasNext() {
		responseData, err := iterator.Next()
		if err != nil {
			return models.Bid{}, err
		}

		var resBid models.Bid
		err = json.Unmarshal(responseData.Value, &resBid)
		if err != nil {
			return models.Bid{}, err
		}
		if resBid.Winner && resBid.AuctionID == auctionID {
			bid = resBid
		}
	}

	return bid, nil
}

func (s *AuctionSmartContract) ReadAuctionsByState(stub shim.ChaincodeStubInterface, state string) pb.Response {
	fmt.Println("ReadAuctions Invoke")

	// Create iterator for all auction entities
	iterator, err := stub.GetStateByPartialCompositeKey(string(EntityAuction), []string{})
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	defer iterator.Close()

	// Create a slice to hold the auction and parcel data
	var response []struct {
		Auction models.Auction `json:"auction"`
		Parcels []int          `json:"parcels"`
		Bids    []models.Bid   `json:"bids"`
	}

	// Loop through all auctions and append them to the response slice
	for iterator.HasNext() {
		responseItem := struct {
			Auction models.Auction `json:"auction"`
			Parcels []int          `json:"parcels"`
			Bids    []models.Bid   `json:"bids"`
		}{}

		responseData, err := iterator.Next()
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		err = json.Unmarshal(responseData.Value, &responseItem.Auction)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		if responseItem.Auction.State == models.AuctionState(state) {

			responseItem.Parcels, err = getParcelsForAuction(stub, responseItem.Auction.ID)
			if err != nil {
				return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
			}

			allBids, err := GetBids(stub)
			if err != nil {
				return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
			}
			// Create a slice to hold the bid objects
			for _, bid := range allBids {
				if bid.AuctionID == responseItem.Auction.ID {
					responseItem.Bids = append(responseItem.Bids, bid)
				}
			}

			response = append(response, responseItem)
		}
	}

	// Convert the response slice to JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(responseJSON)
}

func (s *AuctionSmartContract) DeleteAllAuctions(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("DeleteAllAuctions Invoke")

	// Create iterator for all parcel entities
	iterator, err := stub.GetStateByPartialCompositeKey(string(EntityAuction), []string{})
	if err != nil {
		return shim.Error(err.Error())
	}
	defer iterator.Close()

	// Iterate through all parcels and delete them
	for iterator.HasNext() {
		response, err := iterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		// Delete the parcel by its composite key
		err = stub.DelState(response.Key)
		if err != nil {
			return shim.Error(err.Error())
		}
	}

	return shim.Success(nil)
}

func getBidsForAuction(stub shim.ChaincodeStubInterface, auctionID string) ([]models.Bid, error) {
	// Create iterator for all bid entities with the given auctionID
	iterator, err := stub.GetStateByPartialCompositeKey(string(EntityBid), []string{})
	if err != nil {
		return nil, err
	}
	defer iterator.Close()

	// Create a slice to hold the bids
	var bids []models.Bid

	// Loop through all bid entities and append them to the slice
	for iterator.HasNext() {
		responseData, err := iterator.Next()
		if err != nil {
			return nil, err
		}

		var bid models.Bid
		err = json.Unmarshal(responseData.Value, &bid)
		if err != nil {
			return nil, err
		}
		if bid.AuctionID == auctionID {
			bids = append(bids, bid)
		}
	}

	return bids, nil
}

func (s *AuctionSmartContract) ListOfExpiredAuctions(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("CloseExpiredAuctions Invoke")
	// return shim.Success(createErrorResponse(http.StatusInternalServerError, time.Now().Add(1*time.Hour).Format("2006-01-02T15:04:05Z")))
	// Start a new transaction
	// transactionError := false

	// // Start a new transaction
	// transactionId, err := s.StartTransaction(stub)
	// if err != nil {
	// 	return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	// }
	// // Roolback in case of error Or Commit in case of success
	// defer s.CloseTransaction(stub, transactionId, transactionError)

	// Create iterator for all auction entities
	iterator, err := stub.GetStateByPartialCompositeKey(string(EntityAuction), []string{})
	if err != nil {
		// transactionError = true
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	defer iterator.Close()

	// Get the current time
	currentTime := getCurrentTime()

	var response []string

	// Loop through all auctions and close the expired ones
	for iterator.HasNext() {
		responseData, err := iterator.Next()
		if err != nil {
			// transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		var auction models.Auction
		err = json.Unmarshal(responseData.Value, &auction)
		if err != nil {
			// transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		// Check if the auction's EndDate is before the current time
		if auction.EndDate.Before(currentTime) && auction.State == models.AuctionState(models.AuctionOpen) {
			response = append(response, auction.ID)
		}
	}

	res, err := json.Marshal(response)
	if err != nil {
		// transactionError = true
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	return shim.Success(res)
}

func (s *AuctionSmartContract) CloseExpiredAuctions(stub shim.ChaincodeStubInterface, auctionId string) pb.Response {
	fmt.Println("CloseExpiredAuctions Invoke")
	responseItem := struct {
		AuctionID string `json:"auction"`
		Deliverer int    `json:"deliverer_id"`
		Parcels   []struct {
			ID    int           `json:"id"`
			State models.Status `json:"state"`
		} `json:"parcels"`
	}{}

	responseItem.AuctionID = auctionId

	auctionKey, err := s.CreateCompositeKey(stub, EntityAuction, []string{auctionId})
	if err != nil {
		// transactionError = true
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	auctionJSON, err := s.ReadEntity(stub, auctionKey)
	if err != nil {
		// transactionError = true
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	var auction models.Auction
	err = json.Unmarshal(auctionJSON, &auction)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	// Check if there are any bids for the auction
	bids, err := getBidsForAuction(stub, auction.ID)
	if err != nil {
		// transactionError = true
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	fmt.Println("BIDS: ", bids)

	// Update the auction's state based on the presence of bids
	if len(bids) == 0 {
		auction.State = models.AuctionState(models.AuctionClosedNoBids)
	} else {
		auction.State = models.AuctionState(models.AuctionClosedBids)
		var winnerBid models.Bid
		for _, bid := range bids {
			if bid.Status == models.BitStatusLowerBid {
				winnerBid = bid
				break
			}
		}

		description := fmt.Sprint("Auction ", auction.ID, " payment.")
		err = s.TransferBitcirclesBetweenWallets(stub, winnerBid.CourierID, 0, winnerBid.BitcircleAmount, false, description)
		if err != nil {
			// transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		winnerBid.Winner = true

		winnerBidKey, err := s.CreateCompositeKey(stub, EntityBid, []string{fmt.Sprint(winnerBid.ID), fmt.Sprint(auction.ID)})
		if err != nil {
			// transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		dataWinnerBid, err := json.Marshal(winnerBid)
		if err != nil {
			// transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		_, err = s.UpsertEntityRecord(stub, winnerBidKey, dataWinnerBid)
		if err != nil {
			// transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}
		// Set Deliverer
		responseItem.Deliverer = winnerBid.CourierID
	}

	// Convert the updated auction to JSON
	dataAuction, err := json.Marshal(auction)
	if err != nil {
		// transactionError = true
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	// Update the auction record in the ledger within the transaction
	_, err = s.UpsertEntityRecord(stub, auctionKey, dataAuction)
	if err != nil {
		// transactionError = true
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	responseItem.AuctionID = auction.ID

	// Get the parcels associated with the auction
	parcels, err := getParcelsForAuction(stub, auction.ID)
	if err != nil {
		// transactionError = true
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	// Update the state of each parcel to "Closed"
	for _, parcelID := range parcels {
		newParcel := struct {
			ID    int           `json:"id"`
			State models.Status `json:"state"`
		}{}
		parcelKey, err := s.CreateCompositeKey(stub, EntityParcel, []string{fmt.Sprint(parcelID)})
		if err != nil {
			// transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		entity, err := s.ReadEntity(stub, parcelKey)
		if err != nil {
			// transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		var parcel models.Parcel
		err = json.Unmarshal(entity, &parcel)
		if err != nil {
			// transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		newParcel.ID = parcel.ID
		// Update the parcel's state to "Closed"
		if len(bids) == 0 {
			parcel.State = models.State(models.ParcelStatePending)
			newParcel.State = models.ParcelStatePending
		} else {
			parcel.State = models.State(models.ParcelStateDelivery)
			newParcel.State = models.ParcelStateDelivery
		}

		dataParcel, err := json.Marshal(parcel)
		if err != nil {
			// transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		// Update the parcel record in the ledger within the transaction
		_, err = s.UpsertEntityRecord(stub, parcelKey, dataParcel)
		if err != nil {
			// transactionError = true
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}
		responseItem.Parcels = append(responseItem.Parcels, newParcel)
	}

	res, err := json.Marshal(responseItem)
	if err != nil {
		// transactionError = true
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	return shim.Success(res)
}
