package micolec

import (
	"encoding/json"
	"errors"
	"fmt"
	"micolec/chaincode/models"
	"net/http"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

func GetWallets(stub shim.ChaincodeStubInterface) ([]models.Wallet, error) {
	walletIterator, err := stub.GetStateByPartialCompositeKey(string(EntityWallet), []string{})
	if err != nil {
		return nil, err
	}
	defer walletIterator.Close()

	var wallets []models.Wallet
	for walletIterator.HasNext() {
		response, err := walletIterator.Next()
		if err != nil {
			return nil, err
		}
		var wallet models.Wallet
		err = json.Unmarshal(response.Value, &wallet)
		if err != nil {
			return nil, err
		}
		wallets = append(wallets, wallet)
	}

	return wallets, nil
}

func (s *AuctionSmartContract) GetParticipantWallet(stub shim.ChaincodeStubInterface, participantId int) (models.Wallet, error) {
	var wallet models.Wallet

	walletKey, err := s.CreateCompositeKey(stub, EntityWallet, []string{fmt.Sprint(participantId)})
	if err != nil {
		return wallet, err
	}
	walletJson, err := s.ReadEntity(stub, walletKey)
	if err != nil {
		return wallet, err
	}

	err = json.Unmarshal(walletJson, &wallet)
	if err != nil {
		return wallet, err
	}

	return wallet, nil
}

func (s *AuctionSmartContract) CreateParticipantWallet(stub shim.ChaincodeStubInterface, wallet models.Wallet) pb.Response {
	fmt.Println("CreateParticipantWallet Invoke")
	walletKey, err := s.CreateCompositeKey(stub, EntityWallet, []string{fmt.Sprint(wallet.ParticipantId)})
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	walletExists, err := s.EntityRecordExists(stub, walletKey)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	if walletExists {
		return shim.Success(createErrorResponse(http.StatusBadRequest, fmt.Sprintf("This Participant (%d) already has a wallet", wallet.ParticipantId)))
	}

	walletJson, err := json.Marshal(wallet)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	_, err = s.UpsertEntityRecord(stub, walletKey, walletJson)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(walletJson)
}

func (s *AuctionSmartContract) GetParticipantWalletById(stub shim.ChaincodeStubInterface, participantId int) pb.Response {
	fmt.Println("GetParticipantWalletById Invoke")
	walletKey, err := s.CreateCompositeKey(stub, EntityWallet, []string{fmt.Sprint(participantId)})
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	walletJson, err := s.ReadEntity(stub, walletKey)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	if walletJson == nil {
		return shim.Success(createErrorResponse(http.StatusNotFound, "Participant wallet not found"))
	}

	return shim.Success(walletJson)
}

// func (s *AuctionSmartContract) UpdateBitcircles(stub shim.ChaincodeStubInterface, participantId int, bitcircleAmount int) error {
// 	fmt.Println("UpdateBitcircles")
// 	walletKey, err := s.CreateCompositeKey(stub, EntityWallet, []string{fmt.Sprint(participantId)})
// 	if err != nil {
// 		return err
// 	}

// 	walletJson, err := s.ReadEntity(stub, walletKey)
// 	if err != nil {
// 		return err
// 	}
// 	if walletJson == nil {
// 		return errors.New("Wallet not found")
// 	}

// 	var wallet models.Wallet
// 	err = json.Unmarshal(walletJson, &wallet)
// 	if err != nil {
// 		return err
// 	}

// 	if bitcircleAmount < 0 {
// 		if wallet.Balance < (-bitcircleAmount) {
// 			return errors.New("Insufficient balance on your wallet")
// 		}
// 		wallet.Balance = wallet.Balance - bitcircleAmount
// 		wallet.UsableBalance = wallet.UsableBalance - bitcircleAmount
// 	} else {
// 		wallet.Balance = wallet.Balance + bitcircleAmount
// 		wallet.UsableBalance = wallet.UsableBalance + bitcircleAmount
// 	}

// 	dataWallet, err := json.Marshal(wallet)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = s.UpsertEntityRecord(stub, walletKey, dataWallet)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

func (s *AuctionSmartContract) RefundBitcirclesForAuction(stub shim.ChaincodeStubInterface, participantId int, bitcircleAmmount int, moneyAmount float32, auctionId string, isReward bool) error {
	fmt.Println("Refound Bitcircles")
	walletKey, err := s.CreateCompositeKey(stub, EntityWallet, []string{fmt.Sprint(participantId)})
	if err != nil {
		return err
	}
	walletJson, err := s.ReadEntity(stub, walletKey)
	if err != nil {
		return err
	}
	if walletJson == nil {
		return fmt.Errorf("Wallet not found")
	}

	var wallet models.Wallet
	err = json.Unmarshal(walletJson, &wallet)
	if err != nil {
		return err
	}

	wallet.UsableBalance = wallet.UsableBalance + bitcircleAmmount

	dataWallet, err := json.Marshal(wallet)
	if err != nil {
		return err
	}

	_, err = s.UpsertEntityRecord(stub, walletKey, dataWallet)
	if err != nil {
		return err
	}

	return nil
}

func (s *AuctionSmartContract) ReserveBitcirclesForBid(stub shim.ChaincodeStubInterface, participantId int, bitcircleAmmount int, moneyAmount float32, auctionId string, isReward bool) error {
	fmt.Println("PayBitcircles")
	walletKey, err := s.CreateCompositeKey(stub, EntityWallet, []string{fmt.Sprint(participantId)})
	if err != nil {
		return err
	}
	walletJson, err := s.ReadEntity(stub, walletKey)
	if err != nil {
		return err
	}
	if walletJson == nil {
		return fmt.Errorf("Wallet not found")
	}

	var wallet models.Wallet
	err = json.Unmarshal(walletJson, &wallet)
	if err != nil {
		return err
	}

	if wallet.UsableBalance < bitcircleAmmount {
		return errors.New("Insufficient balance on your wallet")
	}

	wallet.UsableBalance = wallet.UsableBalance - bitcircleAmmount

	dataWallet, err := json.Marshal(wallet)
	if err != nil {
		return err
	}

	_, err = s.UpsertEntityRecord(stub, walletKey, dataWallet)
	if err != nil {
		return err
	}

	return nil
}

func (s *AuctionSmartContract) VerifyWalletAmount(stub shim.ChaincodeStubInterface, participantId int, bitcircleAmount int) error {
	fmt.Println("Verify if wallet have the ammount")
	walletKey, err := s.CreateCompositeKey(stub, EntityWallet, []string{fmt.Sprint(participantId)})
	if err != nil {
		return err
	}
	walletJson, err := s.ReadEntity(stub, walletKey)
	if err != nil {
		return err
	}
	if walletJson == nil {
		return fmt.Errorf("Wallet not found")
	}

	var wallet models.Wallet
	err = json.Unmarshal(walletJson, &wallet)
	if err != nil {
		return err
	}

	if wallet.UsableBalance < bitcircleAmount {
		return errors.New("Insufficient balance on your wallet")
	}

	return nil
}

func (s *AuctionSmartContract) GetWallet(stub shim.ChaincodeStubInterface, participantIdd int) (models.Wallet, string, error) {
	var wallet models.Wallet

	walletKey, err := s.CreateCompositeKey(stub, EntityWallet, []string{fmt.Sprint(participantIdd)})
	if err != nil {
		return wallet, walletKey, err
	}
	walletJson, err := s.ReadEntity(stub, walletKey)
	if err != nil {
		return wallet, walletKey, err
	}
	if walletJson == nil {
		return wallet, walletKey, fmt.Errorf("Wallet not found")
	}

	err = json.Unmarshal(walletJson, &wallet)
	if err != nil {
		return wallet, walletKey, err
	}

	return wallet, walletKey, nil
}

func GetTransactionId(stub shim.ChaincodeStubInterface) (int, error) {
	transactionsIterator, err := stub.GetStateByPartialCompositeKey(string(EntityBitcircleTransaction), []string{})
	if err != nil {
		return 0, err
	}
	defer transactionsIterator.Close()

	var transactions []models.BitcircleTransaction
	for transactionsIterator.HasNext() {
		response, err := transactionsIterator.Next()
		if err != nil {
			return 0, err
		}
		var transaction models.BitcircleTransaction
		err = json.Unmarshal(response.Value, &transaction)
		if err != nil {
			return 0, err
		}
		transactions = append(transactions, transaction)
	}

	return len(transactions) + 1, nil
}

func (s *AuctionSmartContract) TransferBitcircles(stub shim.ChaincodeStubInterface, senderParticipantId int, receiverParticipantId int, bitcircleAmmount int, isReward bool, description string) pb.Response {
	err := s.VerifyWalletAmount(stub, senderParticipantId, bitcircleAmmount)
	if err != nil {
		return shim.Success(createErrorResponse(500, err.Error()))
	}

	err = s.TransferBitcirclesBetweenWallets(stub, senderParticipantId, receiverParticipantId, bitcircleAmmount, isReward, description)
	if err != nil {
		return shim.Success(createErrorResponse(500, err.Error()))
	}

	var res struct {
		SenderParticipantId   int    `json:"sender_participant_id"`
		ReceiverParticipantId int    `json:"receiver_participant_id"`
		BitcircleAmount       int    `json:"bitcircle_amount"`
		IsReward              bool   `json:"isReward"`
		Description           string `json:"description"`
	}
	res.SenderParticipantId = senderParticipantId
	res.ReceiverParticipantId = receiverParticipantId
	res.BitcircleAmount = bitcircleAmmount
	res.Description = description
	res.IsReward = isReward

	//Convert the slice of bids to JSON
	resJSON, err := json.Marshal(res)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(resJSON)
}

func (s *AuctionSmartContract) TransferBitcirclesBetweenWallets(stub shim.ChaincodeStubInterface, senderParticipantId int, receiverParticipantId int, bitcircleAmmount int, isReward bool, description string) error {
	fmt.Println("PayBitcircles")

	transactionError := false

	// Start a new transaction
	transactionId, err := s.StartTransaction(stub)
	if err != nil {
		return err
	}
	// Roolback in case of error Or Commit in case of success
	defer s.CloseTransaction(stub, transactionId, transactionError)

	senderWallet, senderWalletKey, err := s.GetWallet(stub, senderParticipantId)
	if err != nil {
		return err
	}

	receiverWallet, receiverWalletKey, err := s.GetWallet(stub, receiverParticipantId)
	if err != nil {
		return err
	}

	currentTime := time.Now()
	currentDate := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC)

	senderWallet.Balance = senderWallet.Balance - bitcircleAmmount
	if isReward {
		senderWallet.UsableBalance = senderWallet.UsableBalance - bitcircleAmmount
	}
	senderWallet.LastMovement = currentDate

	receiverWallet.Balance = receiverWallet.Balance + bitcircleAmmount
	receiverWallet.UsableBalance = receiverWallet.UsableBalance + bitcircleAmmount
	receiverWallet.LastMovement = currentDate

	var bitcircletransaction models.BitcircleTransaction
	// transactionId, err := GetTransactionId(stub)
	// if err != nil {
	// 	return err
	// }

	Bitcircle_Transaction_ID = Bitcircle_Transaction_ID + 1

	bitcircletransaction.ID = Bitcircle_Transaction_ID
	bitcircletransaction.BitcircleAmount = bitcircleAmmount
	bitcircletransaction.SenderParticipantId = senderParticipantId
	bitcircletransaction.ReceiverParticipantId = receiverParticipantId
	bitcircletransaction.Date = currentDate
	bitcircletransaction.Description = description

	bitcircleTransactionKey, err := s.CreateCompositeKey(stub, EntityBitcircleTransaction, []string{fmt.Sprint(bitcircletransaction.ID)})
	if err != nil {
		return err
	}

	dataBitcircleTransaction, err := json.Marshal(bitcircletransaction)
	if err != nil {
		return err
	}
	_, err = s.UpsertEntityRecord(stub, bitcircleTransactionKey, dataBitcircleTransaction)
	if err != nil {
		return err
	}

	dataSenderWallet, err := json.Marshal(senderWallet)
	if err != nil {
		return err
	}

	_, err = s.UpsertEntityRecord(stub, senderWalletKey, dataSenderWallet)
	if err != nil {
		return err
	}

	dataReceiverWallet, err := json.Marshal(receiverWallet)
	if err != nil {
		return err
	}

	_, err = s.UpsertEntityRecord(stub, receiverWalletKey, dataReceiverWallet)
	if err != nil {
		return err
	}

	return nil
}

func GetAllBitCircleTransactions(stub shim.ChaincodeStubInterface) ([]models.BitcircleTransaction, error) {
	bitcircleTransactionIterator, err := stub.GetStateByPartialCompositeKey(string(EntityBitcircleTransaction), []string{})
	if err != nil {
		return nil, err
	}
	defer bitcircleTransactionIterator.Close()

	var bitcircletransactions []models.BitcircleTransaction
	for bitcircleTransactionIterator.HasNext() {
		response, err := bitcircleTransactionIterator.Next()
		if err != nil {
			return nil, err
		}
		var bitcircletransaction models.BitcircleTransaction
		err = json.Unmarshal(response.Value, &bitcircletransaction)
		if err != nil {
			return nil, err
		}
		bitcircletransactions = append(bitcircletransactions, bitcircletransaction)
	}

	return bitcircletransactions, nil
}

func (s *AuctionSmartContract) GetParticipantBitCircleTransactions(stub shim.ChaincodeStubInterface, userID int) pb.Response {
	allBitcircleTransactions, err := GetAllBitCircleTransactions(stub)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	var bitcircletransactions []models.BitcircleTransaction
	for _, bitcircleTransaction := range allBitcircleTransactions {
		if bitcircleTransaction.SenderParticipantId == userID || bitcircleTransaction.ReceiverParticipantId == userID {
			bitcircletransactions = append(bitcircletransactions, bitcircleTransaction)
		}
	}

	//Convert the slice of bids to JSON
	bitcircleTransactionsJSON, err := json.Marshal(bitcircletransactions)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(bitcircleTransactionsJSON)
}
