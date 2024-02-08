package micolec

import (
	"encoding/json"
	"errors"
	"fmt"
	"micolec/chaincode/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

func validateParcelDeliveryParcelAdded(parcel *models.Parcel) error {
	var errorMessages []string

	if !(parcel.ID > 0) {
		errorMessages =
			append(errorMessages, "ID Higher Equal 1")
	}

	if parcel.State != models.State(models.ParcelStatePending) {
		errorMessages =
			append(errorMessages, "Invalid Parcel State, when creating a parcel default state most be 'Pending'")
	}

	//TODO Validar datas de entrada se são maiores que a data atual.

	if parcel.RequiredDeliveryDate.Before(parcel.AddedToPlatform) {
		errorMessages =
			append(errorMessages, "Date Required to Delivery most be After the Date to add on the Platform")
	}

	if !(len(strings.TrimSpace(parcel.PickupPostalArea)) >= 3) {
		errorMessages =
			append(errorMessages, "PickupPostalArea Min. Length 3")
	}

	if !(len(strings.TrimSpace(parcel.DeliveryPostalArea)) >= 3) {
		errorMessages =
			append(errorMessages, "DeliveryPostalArea Min. Length 3")
	}

	if !(parcel.BitcircleReward >= 0) {
		errorMessages =
			append(errorMessages, "BitcircleReward Higher Equal 0. Current value: "+strconv.Itoa(parcel.BitcircleReward))
	}

	// TODO -> Validar com o Valentim e sugerir alteração para Int ou float
	// if !(parcel.Weight > 0) {
	// 	errorMessages =
	// 		append(errorMessages, "Weight Higher than 0")
	// }

	if !(parcel.Volumes > 0) {
		errorMessages =
			append(errorMessages, "Volume Higher than 0")
	}

	if len(errorMessages) > 0 {
		return errors.New(strings.Join(errorMessages, "\n"))
	}

	if !(parcel.LogisticOperatorId > 0) {
		errorMessages =
			append(errorMessages, "Invalid Logistic Operator Id")
	}

	// // TODO -> Validar com o Valentim, um id numerico deve ser armazenado como int
	// if !(parcel.EndCustomerId != "") {
	// 	errorMessages =
	// 		append(errorMessages, "Invalid End Customer Id")
	// }

	if len(errorMessages) > 0 {
		return errors.New(strings.Join(errorMessages, "\n"))
	}

	return nil
}

func (s *AuctionSmartContract) ReadParcels(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("ReadParcels Invoke")
	// Create iterator for all parcel entities
	iterator, err := stub.GetStateByPartialCompositeKey(string(EntityParcel), []string{})
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	defer iterator.Close()

	// Create a slice to hold the parcel objects
	var parcels []models.Parcel

	// Loop through all parcels and append them to the slice
	for iterator.HasNext() {
		response, err := iterator.Next()
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		var parcel models.Parcel
		err = json.Unmarshal(response.Value, &parcel)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		parcels = append(parcels, parcel)
	}

	// Convert the slice of parcels to JSON
	parcelJSON, err := json.Marshal(parcels)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(parcelJSON)
}

func (s *AuctionSmartContract) ReadParcelsByState(stub shim.ChaincodeStubInterface, state string) pb.Response {
	fmt.Println("ReadParcelsByState Invoke")
	// Create iterator for all parcel entities
	iterator, err := stub.GetStateByPartialCompositeKey(string(EntityParcel), []string{})
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	defer iterator.Close()

	// Create a slice to hold the parcel objects
	var parcels []models.Parcel

	// Loop through all parcels and append them to the slice
	for iterator.HasNext() {
		response, err := iterator.Next()
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		var parcel models.Parcel
		err = json.Unmarshal(response.Value, &parcel)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}
		if parcel.State == models.State(state) {
			parcels = append(parcels, parcel)
		}
	}

	// Convert the slice of parcels to JSON
	parcelJSON, err := json.Marshal(parcels)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(parcelJSON)
}

/* Command to create a Parcel
CORE_PEER_ADDRESS=127.0.0.1:7051 peer chaincode invoke -o 127.0.0.1:7050 -C ch1 -n mycc -c '{"Args":["ParcelDeliveryParcelAdded","{\"id\":2,\"state\":\"Pending\",\"added_to_platform\":\"2023-05-11T00:00:00Z\",\"required_delivery_date\":\"2023-05-15T12:00:00Z\",\"pickup_postal_area\":\"Area1\",\"delivery_postal_area\":\"Area2\",\"notified_ce_option\":false,\"bitcircle_reward\":100,\"weight\":10,\"length\":20,\"width\":15,\"height\":30,\"volume\":900}"]}'
*/

func (s *AuctionSmartContract) ParcelDeliveryParcelAdded(stub shim.ChaincodeStubInterface, parcel models.Parcel) pb.Response {
	fmt.Println("ParcelDeliveryParcelAdded Invoke")
	// Create and store parcel entity
	parcelKey, err := s.CreateCompositeKey(stub, EntityParcel, []string{fmt.Sprint(parcel.ID)})
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusBadRequest, err.Error()))
	}

	// Check if parcel already exists
	if parcelRecordExists, err := s.EntityRecordExists(stub, parcelKey); err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	} else if parcelRecordExists {
		return shim.Success(createErrorResponse(http.StatusConflict, "Record Already Exists"))
	}

	// Validate parcel
	if err := validateParcelDeliveryParcelAdded(&parcel); err != nil {
		fmt.Println("Error Validating Parcel")
		return shim.Success(createErrorResponse(http.StatusBadRequest, err.Error()))
	}

	// Insert Parcel on the blockchain
	jsonData, err := json.Marshal(parcel)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	_, err = s.UpsertEntityRecord(stub, parcelKey, jsonData)
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}

	return shim.Success(jsonData)
}

func (s *AuctionSmartContract) DeleteAllParcels(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("DeleteAllParcels Invoke")
	// Create iterator for all parcel entities
	iterator, err := stub.GetStateByPartialCompositeKey(string(EntityParcel), []string{})
	if err != nil {
		return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
	}
	defer iterator.Close()

	// Iterate through all parcels and delete them
	for iterator.HasNext() {
		response, err := iterator.Next()
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}

		// Delete the parcel by its composite key
		err = stub.DelState(response.Key)
		if err != nil {
			return shim.Success(createErrorResponse(http.StatusInternalServerError, err.Error()))
		}
	}

	return shim.Success(nil)
}

// ** -----------------------------------------------------
// ** PARCEL
// ** -> END
// ** -----------------------------------------------------
