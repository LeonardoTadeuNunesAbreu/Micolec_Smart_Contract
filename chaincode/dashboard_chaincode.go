package micolec

import (
	"encoding/json"
	"fmt"
	"micolec/chaincode/models"
	"net/http"
	"sort"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

func (s *AuctionSmartContract) AdminPlatformDashboard(stub shim.ChaincodeStubInterface) pb.Response {
	auctions, err := s.GetLastAuctions(stub)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusBadRequest, err.Error()))
	}

	bids, err := s.GetLastBids(stub)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusBadRequest, err.Error()))
	}

	var response struct {
		Auctions []models.Auction `json:"auctions"`
		Bids     []models.Bid     `json:"bids"`
	}

	response.Auctions = auctions
	response.Bids = bids

	responseJson, err := json.Marshal(response)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(responseJson)
}

func GetClosedAuctionsByMonthYear(auctions []models.Auction) []map[string]interface{} {
	closedAuctions := make(map[string]int)
	closedAuctionsBids := make(map[string]int)

	for _, auction := range auctions {
		monthYear := fmt.Sprintf("%02d/%d", auction.EndDate.Month(), auction.EndDate.Year())
		if auction.State == models.AuctionState(models.AuctionClosedBids) {
			closedAuctionsBids[monthYear]++
		}
		closedAuctions[monthYear]++
	}

	result := []map[string]interface{}{}
	for monthYear, totalAuctions := range closedAuctions {
		entry := map[string]interface{}{
			"month_year":               monthYear,
			"total_auctions":           totalAuctions,
			"total_auctions_with_bids": closedAuctionsBids[monthYear],
		}
		result = append(result, entry)
	}

	return result
}

func (s *AuctionSmartContract) LogisticOperatorDashboard(stub shim.ChaincodeStubInterface, userId int) pb.Response {
	auctions, err := GetAuctions(stub)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusBadRequest, err.Error()))
	}

	// Quantidade Leilões Abertos
	openAuctions := 0
	// My Last Auctions
	var myAuctions []models.Auction
	// Closed Auctions
	var myClosedAuctionAmount = 0
	for _, auction := range auctions {
		if auction.State == models.AuctionState(models.AuctionOpen) {
			openAuctions += 1
		}

		if auction.ParticipantId == userId {
			myAuctions = append(myAuctions, auction)
			if auction.State == models.AuctionState(models.AuctionClosedBids) {
				myClosedAuctionAmount += 1
			}
		}
	}

	lastAuctions := make([]models.Auction, 0)

	// Sort bids by date in descending order
	sort.SliceStable(myAuctions, func(i, j int) bool {
		return myAuctions[i].StartDate.After(myAuctions[j].StartDate)
	})

	// Get the last bids (assuming you want the top N bids)
	N := 5 // Number of last bids to retrieve
	if len(myAuctions) < N {
		N = len(myAuctions)
	}

	lastAuctions = myAuctions[:N]

	var response struct {
		OpenAuctionsAmount int                      `json:"open_auctions_amount"`
		MyLastAuctions     []models.Auction         `json:"my_last_auctions"`
		AuctionsPlotData   []map[string]interface{} `json:"auctions_plot_data"`
	}
	response.OpenAuctionsAmount = openAuctions
	response.MyLastAuctions = lastAuctions
	response.AuctionsPlotData = GetClosedAuctionsByMonthYear(myAuctions)

	responseJson, err := json.Marshal(response)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(responseJson)
}

func (s *AuctionSmartContract) CourierDashboard(stub shim.ChaincodeStubInterface) pb.Response {
	auctions, err := GetAuctions(stub)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusBadRequest, err.Error()))
	}

	// Quantidade Leilões Abertos
	openAuctions := 0
	for _, auction := range auctions {
		if auction.State == models.AuctionState(models.AuctionOpen) {
			openAuctions += 1
		}
	}
	var response struct {
		OpenAuctionsAmount int `json:"open_auctions_amount"`
	}
	response.OpenAuctionsAmount = openAuctions

	responseJson, err := json.Marshal(response)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(responseJson)
}
