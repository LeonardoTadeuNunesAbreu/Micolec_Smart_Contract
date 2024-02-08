package micolec

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"micolec/chaincode/models"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

// SimpleChaincode example simple Chaincode implementation
type AuctionSmartContract struct{}

type Entity string

const (
	EntityAuction              Entity = "AUCTION"
	EntityParcel               Entity = "PARCEL"
	EntityAuctionHasParcel     Entity = "AUCTION_HAS_PARCEL"
	EntityBid                  Entity = "BID"
	EntityWallet               Entity = "WALLET"
	EntityBitcircleTransaction Entity = "BITCIRCLETRANSACTION"
)

const PlatformWalletId = 0

var Bitcircle_Transaction_ID = 0

type ErrorResponse struct {
	ErrorCode    int    `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

func createErrorResponse(errorCode int, errorMsg string) []byte {
	response := ErrorResponse{
		ErrorCode:    errorCode,
		ErrorMessage: errorMsg,
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		// Return a generic error response if JSON marshaling fails
		defaultResponse := ErrorResponse{
			ErrorCode:    http.StatusInternalServerError,
			ErrorMessage: "Internal Server Error",
		}
		jsonResponse, _ = json.Marshal(defaultResponse)
	}

	return jsonResponse
}

func getCurrentTime() time.Time {
	return time.Now().Add(1 * time.Hour)
}

// ** -----------------------------------------------------
// ** ENTITY RECORDS METHODS
// ** -> BEGIN
// ** -----------------------------------------------------

// Create a compositeKey
func (s *AuctionSmartContract) CreateCompositeKey(stub shim.ChaincodeStubInterface, entity Entity, ids []string) (string, error) {
	compositeKey, err := stub.CreateCompositeKey(string(entity), ids)
	return compositeKey, err
}

// Create Record -> Receive compositeKey + Dados do registo!
func (s *AuctionSmartContract) UpsertEntityRecord(stub shim.ChaincodeStubInterface, key string, data []byte) (bool, error) {
	err := stub.PutState(key, data)
	if err != nil {
		return false, fmt.Errorf("failed to put data for key %s: %w", key, err)
	}
	return true, nil
}

// Check if record Exists
func (s *AuctionSmartContract) EntityRecordExists(stub shim.ChaincodeStubInterface, key string) (bool, error) {
	recordJSON, err := stub.GetState(key)
	if err != nil {
		return false, fmt.Errorf("Failed to read from world state: %v", err)
	}
	return recordJSON != nil, nil
}

// Read record from ledger
func (s *AuctionSmartContract) ReadEntity(stub shim.ChaincodeStubInterface, key string) ([]byte, error) {
	recordJSON, err := stub.GetState(key)
	if err != nil {
		return nil, fmt.Errorf("Failed to read from world state: %v", err)
	}
	if recordJSON == nil {
		return nil, fmt.Errorf("The asset %s does not exist", key)
	}
	return recordJSON, nil
}

func (s *AuctionSmartContract) CreateEntityIterator(stub shim.ChaincodeStubInterface, entity Entity, ids []string) (shim.StateQueryIteratorInterface, error) {
	entityIterator, err := stub.GetStateByPartialCompositeKey(string(entity), ids)
	if err != nil {
		return nil, err
	}
	return entityIterator, nil
}

func (s *AuctionSmartContract) StartTransaction(stub shim.ChaincodeStubInterface) (string, error) {
	transactionId := stub.GetTxID()
	err := stub.SetEvent("tx.start", []byte(transactionId))
	if err != nil {
		return "", err
	}

	return transactionId, nil
}

// func (s *AuctionSmartContract) CloseTransaction(stub shim.ChaincodeStubInterface, transactionId string, err bool) {
// 	if err == true {
// 		// Roll back the transaction
// 		_ = stub.SetEvent("tx.rollback", []byte(transactionId))
// 		_ = stub.DelState(transactionId)
// 		return
// 	}
// 	// Commit the transaction
// 	_ = stub.SetEvent("tx.commit", []byte(transactionId))
// }

func (s *AuctionSmartContract) CloseTransaction(stub shim.ChaincodeStubInterface, transactionId string, err bool) error {
	if err == true {
		// Roll back the transaction
		err := stub.SetEvent("tx.rollback", []byte(transactionId))
		if err != nil {
			return err
		}

		err = stub.DelState(transactionId)
		if err != nil {
			return err
		}
		return nil
	}

	// Commit the transaction
	comitErr := stub.SetEvent("tx.commit", []byte(transactionId))
	if comitErr != nil {
		return comitErr
	}

	return nil
}

// ** -----------------------------------------------------
// ** ENTITY RECORDS METHODS
// ** -> END
// ** -----------------------------------------------------

func (t *AuctionSmartContract) Init(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("Init returning with success")
	return shim.Success(nil)
}

func (t *AuctionSmartContract) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("Invoke")
	if os.Getenv("DEVMODE_ENABLED") != "" {
		fmt.Println("invoking in devmode")
	}
	function, args := stub.GetFunctionAndParameters()
	fmt.Println("ARGS:", args)

	switch function {
	case "ParcelDeliveryParcelAdded":
		if len(args) < 1 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting a JSON object as an argument"))
		}
		var parcel models.Parcel
		err := json.Unmarshal([]byte(args[0]), &parcel)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Failed to parse JSON object: "+err.Error()))
		}
		return t.ParcelDeliveryParcelAdded(stub, parcel)
	case "ReadParcels":
		return t.ReadParcels(stub)
	case "ReadParcelsByState":
		if len(args) < 1 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting a JSON object as an argument"))
		}
		state := args[0]
		return t.ReadParcelsByState(stub, state)
	case "DeleteAllParcels":
		return t.DeleteAllParcels(stub)
	case "ParcelDeliveryAuctionStart":
		if len(args) < 2 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting a JSON object as an argument"))
		}

		var parcels []models.AuctionHasParcel
		err := json.Unmarshal([]byte(args[0]), &parcels)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, err.Error()))
		}

		var auction models.Auction
		err = json.Unmarshal([]byte(args[1]), &auction)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, err.Error()))
		}

		return t.ParcelDeliveryAuctionStart(stub, parcels, auction)
	case "GetAuctionByID":
		id := args[0]
		return t.GetAuctionByID(stub, id)
	case "GetAuctionByParcelID":
		if len(args) < 1 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting 1 arguments: \"ParcelID\""))
		}
		parcelID, err := strconv.Atoi(args[0])
		if err != nil {
			// Handle error if the conversion fails
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Conversion failed:"+err.Error()))
		}
		return t.GetAuctionByParcelID(stub, parcelID)
	case "ReadAuctions":
		return t.ReadAuctions(stub)
	case "ReadAuctionsByState":
		if len(args) < 1 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting a JSON object as an argument"))
		}
		state := args[0]
		return t.ReadAuctionsByState(stub, state)
	case "DeleteAllAuctions":
		return t.DeleteAllAuctions(stub)
	case "ParcelDeliveryBidingRequest":
		if len(args) < 5 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting 5 arguments: \"Id\", \"Auction Id\", \"Money Amount\", \"Bitcircles\", \"CourierId\" and \"Date\""))
		}
		bidID := args[0]
		auctionID := args[1]

		moneyAmount, err := strconv.ParseFloat(args[2], 32)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, err.Error()))
		}
		bitcircle, err := strconv.Atoi(args[3])
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, err.Error()))
		}
		courierID, err := strconv.Atoi(args[4])
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, err.Error()))
		}

		date, err := time.Parse(time.RFC3339, args[5])
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, err.Error()))
		}

		return t.ParcelDeliveryBidingRequest(stub, bidID, auctionID, float32(moneyAmount), bitcircle, courierID, date)
	case "ReadBids":
		return t.ReadBids(stub)
	case "GetBidsForAuction":
		if len(args) < 1 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting \"auctionId\"  as an argument"))
		}
		auctionID := args[0]
		return t.GetBidsForAuction(stub, auctionID)
	case "GetParticipantBids":
		if len(args) < 1 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting \"UserId\" as an argument"))
		}
		userID, err := strconv.Atoi(args[0])
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, fmt.Sprint("Error getting Participant id: ", args[1])))
		}
		return t.GetParticipantBids(stub, userID)
	case "DeleteAllBids":
		return t.DeleteAllBids(stub)
	case "CloseExpiredAuctions":
		if len(args) < 1 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting \"id\" as an argument"))
		}
		id := args[0]
		return t.CloseExpiredAuctions(stub, id)
	case "ListOfExpiredAuctions":
		return t.ListOfExpiredAuctions(stub)
	case "CreateParticipantWallet":
		if len(args) < 1 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting a JSON object as an argument"))
		}
		var wallet models.Wallet
		err := json.Unmarshal([]byte(args[0]), &wallet)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, err.Error()))
		}
		return t.CreateParticipantWallet(stub, wallet)
	case "GetParticipantWalletById":
		if len(args) < 1 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting \"UserId\" as an argument"))
		}
		userID, err := strconv.Atoi(args[0])
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, fmt.Sprint("Error getting Participant id: ", args[0])))
		}
		return t.GetParticipantWalletById(stub, userID)
	case "GetParticipantBitCircleTransactions":
		if len(args) < 1 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting \"UserId\" as an argument"))
		}
		userID, err := strconv.Atoi(args[0])
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, fmt.Sprint("Error getting Participant id: ", args[0])))
		}
		return t.GetParticipantBitCircleTransactions(stub, userID)
	case "TransferBitcircles":
		if len(args) < 5 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting 6 arguments as an argument: \"senderParticipantId\", \"receiverParticipantId\", \"bitcircleAmmount\", \"isReward\", \"description\" "))
		}
		senderParticipantId, err := strconv.Atoi(args[0])
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, fmt.Sprint("Error getting SenderParticipantId: ", args[0])))
		}
		receiverParticipantId, err := strconv.Atoi(args[1])
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, fmt.Sprint("Error getting ReceiverParticipantId ", args[1])))
		}
		bitcircleAmount, err := strconv.Atoi(args[2])
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, fmt.Sprint("Error getting bitcircleAmount: ", args[2])))
		}

		isReward, err := strconv.ParseBool(args[3])
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, fmt.Sprint("Error getting isReward: ", args[3])))
		}

		description := args[4]

		return t.TransferBitcircles(stub, senderParticipantId, receiverParticipantId, bitcircleAmount, isReward, description)
	case "AdminPlatformDashboard":
		return t.AdminPlatformDashboard(stub)
	case "LogisticOperatorDashboard":
		if len(args) < 1 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting \"UserId\" as an argument"))
		}
		userID, err := strconv.Atoi(args[0])
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, fmt.Sprint("Error getting Participant id: ", args[0])))
		}
		return t.LogisticOperatorDashboard(stub, userID)
	case "CourierDashboard":
		return t.CourierDashboard(stub)
	case "SeedParcel":
		if len(args) < 1 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting a JSON object as an argument"))
		}
		var parcels []models.Parcel
		err := json.Unmarshal([]byte(args[0]), &parcels)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Failed to parse JSON object: "+err.Error()))
		}
		return t.SeedParcel(stub, parcels)
	case "SeedAuction":
		if len(args) < 2 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting a JSON object as an argument"))
		}
		var auctions []models.Auction
		err := json.Unmarshal([]byte(args[0]), &auctions)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Failed to parse JSON object: "+err.Error()))
		}
		var auctionsHasParcels []models.AuctionHasParcel
		err = json.Unmarshal([]byte(args[1]), &auctionsHasParcels)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Failed to parse JSON object: "+err.Error()))
		}
		return t.SeedAuction(stub, auctions, auctionsHasParcels)
	case "SeedBid":
		if len(args) < 1 {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting a JSON object as an argument"))
		}
		var bids []models.Bid
		err := json.Unmarshal([]byte(args[0]), &bids)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusBadRequest, "Failed to parse JSON object: "+err.Error()))
		}
		return t.SeedWinningBid(stub, bids)
	// case "TransferBitcirclesBetweenWallets":
	// 	if len(args) < 6 {
	// 		return shim.Success(createErrorResponse(http.StatusBadRequest, "Expecting  \"senderParticipantId\", \"receiverParticipantId\", \"bitcircleAmmount\", \"auctionId\", \"isReward\", \"description\" "))
	// 	}
	// 	senderParticipantId, err := strconv.Atoi(args[0])
	// 	if err != nil {
	// 		return shim.Success(createErrorResponse(http.StatusBadRequest, fmt.Sprint("Error getting Participant id: ", args[1])))
	// 	}
	// 	receiverParticipantId, err := strconv.Atoi(args[1])
	// 	if err != nil {
	// 		return shim.Success(createErrorResponse(http.StatusBadRequest, fmt.Sprint("Error getting Participant id: ", args[1])))
	// 	}
	// 	bitcircleAmmount, err := strconv.Atoi(args[2])
	// 	if err != nil {
	// 		return shim.Success(createErrorResponse(http.StatusBadRequest, fmt.Sprint("Error getting Participant id: ", args[1])))
	// 	}
	// 	auctionId := args[3]
	// 	reward := args[4] == "1"

	// 	description := args[5]

	// 	return t.TransferBitcirclesBetweenWallets(stub, senderParticipantId, receiverParticipantId, bitcircleAmmount, auctionId, reward, description)
	default:
		return shim.Success(createErrorResponse(http.StatusBadRequest, "Invalid invoke function name."))
	}
}
