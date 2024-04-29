package eth

import (
	"context"
	"log"
	"os"
	"strings"

	"gethmate/utils"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ERC20Token struct {
	ContractAddress common.Address `json:"contract_address"`
	Name            string         `json:"name"`
	Symbol          string         `json:"symbol"`
	Decimals        int            `json:"decimals"`
	Initalized      bool
}

func NewERC20Token(contractAddress common.Address) *ERC20Token {
	return &ERC20Token{
		ContractAddress: contractAddress,
		Initalized:      false,
	}
}

func (t *ERC20Token) Initialize(client *ethclient.Client) {
	jsonBytes, err := os.ReadFile("TokenERC20.json")
	if err != nil {
		log.Fatalf("Failed to read TokenERC20.json: %v", err)
	}
	parsedABI, _ := abi.JSON(strings.NewReader(string(jsonBytes)))

	callMsg := ethereum.CallMsg{
		To:   &t.ContractAddress,
		Data: utils.GetFunctionSelector("name()"),
	}
	ctx := context.Background()

	result, err := client.CallContract(ctx, callMsg, nil)
	if err != nil || len(result) == 0 {
		log.Printf("failed name initialise (%s)", t.ContractAddress)
		return
	} else {
		data, err := parsedABI.Unpack("name", result)
		if err != nil {
			log.Printf("failed name initialise (%s)", t.ContractAddress)
			return
		} else {
			t.Name = data[0].(string)
		}
	}

	callMsg.Data = utils.GetFunctionSelector("symbol()")
	result, err = client.CallContract(ctx, callMsg, nil)
	if err != nil || len(result) == 0 {
		log.Printf("failed symbol initialise (%s)", t.ContractAddress)
		return
	} else {
		data, err := parsedABI.Unpack("symbol", result)
		if err != nil {
			log.Printf("failed symbol initialise (%s)", t.ContractAddress)
			return
		} else {
			t.Symbol = data[0].(string)
		}
	}

	callMsg.Data = utils.GetFunctionSelector("decimals()")
	result, err = client.CallContract(ctx, callMsg, nil)
	if err != nil || len(result) == 0 {
		log.Printf("failed decimals initialize (%s)", t.ContractAddress)
		return
	} else {
		t.Decimals = int(uint8(result[len(result)-1]))
	}
	t.Initalized = true
}

func (t *ERC20Token) Equals(token *ERC20Token) bool {
	return strings.EqualFold(t.ContractAddress.String(), token.ContractAddress.String())
}
