
package main

import (
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"strings"
	"encoding/pem"
	"crypto/x509"
	"encoding/json"
)

var logger = shim.NewLogger("RelationshipChaincode")

const orderIndex = "Order"

type Order struct {
	Buyer string `json:"buyer"`
	Seller string `json:"seller"`
	Asset string `json:"asset"`
	Qty string `json:"qty"`
	Price string `json:"price"`
	Reference string `json:"reference"`
	Status string `json:"status"`
}

type RelationshipChaincode struct {
}

func (t *RelationshipChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (t *RelationshipChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	if function == "order" {
		return t.order(stub, args)
	} else if function == "query" {
		return t.query(stub, args)
	}

	return pb.Response{Status:400, Message:"invalid invoke function name"}
}

func (t *RelationshipChaincode) order(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	logger.Debug("order", args)

	var err error

	if len(args) != 5 {
		return pb.Response{Status:400, Message:"incorrect number of arguments"}
	}

	buyer := getCreatorOrganization(stub)

	seller := args[0]
	asset := args[1]
	qty := args[2]
	price := args[3]
	reference := args[4]

	keyParts := []string{buyer, seller, asset, qty, price, reference}
	key, err := stub.CreateCompositeKey(orderIndex, keyParts)

	logger.Debug("CreateCompositeKey key", key)

	if err != nil {
		return shim.Error(err.Error())
	}

	data, err := stub.GetState(key)

	logger.Debug("GetState data", data)

	if err != nil {
		return shim.Error(err.Error())
	}

	var status = []byte("initiated")

	if data != nil {
		status = []byte("matched")
	}

	logger.Debug("status", status)

	stub.PutState(key, []byte(status))

	return shim.Success(status)
}

func (t *RelationshipChaincode) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	it, err := stub.GetStateByPartialCompositeKey(orderIndex, []string{})
	if err != nil {
		return shim.Error(err.Error())
	}
	defer it.Close()

	orders := []Order{}
	for it.HasNext() {
		next, err := it.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		logger.Debug("next", next)

		_, compositeKeyParts, err := stub.SplitCompositeKey(next.Key)
		if err != nil {
			return shim.Error(err.Error())
		}

		order := Order{Buyer:compositeKeyParts[0],
			Seller:compositeKeyParts[1],
			Asset:compositeKeyParts[2],
			Qty:compositeKeyParts[3],
			Price:compositeKeyParts[4],
			Reference:compositeKeyParts[5]}

		order.Status = string(next.Value)

		logger.Debug("order", order)

		orders = append(orders, order)
	}

	ret, err := json.Marshal(orders)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(ret)
}

func getCreatorOrganization(stub shim.ChaincodeStubInterface) string {
	certificate, _ := stub.GetCreator()

	data := certificate[strings.Index(string(certificate), "-----") : strings.LastIndex(string(certificate), "-----")+5]
	block, _ := pem.Decode([]byte(data))
	cert, _ := x509.ParseCertificate(block.Bytes)
	organization := cert.Issuer.Organization[0]

	logger.Debug("getOrganization: " + organization)

	return organization
}

func main() {
	err := shim.Start(new(RelationshipChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}
