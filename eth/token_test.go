package eth

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func TestERC20Token(t *testing.T) {
	fmt.Println("TestERC20Token")
	token := NewERC20Token(common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"))
	if !strings.EqualFold(token.ContractAddress.String(), "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2") {
		t.Errorf("Expected 0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2, got %s", token.ContractAddress.String())
	}
	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		t.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	ch := make(chan int)
	token.Initialize(client)
	<-ch
	if token.Name != "Wrapped Ether" {
		fmt.Println([]byte(token.Name))
		fmt.Println([]byte("Wrapped Ether"))
		t.Errorf("Expected Wrapped Ether, got %s", token.Name)
	}
	if token.Symbol != "WETH" {
		fmt.Println([]byte(token.Symbol))
		fmt.Println([]byte("WETH"))
		t.Errorf("Expected WETH, got %s", token.Symbol)
	}
	if token.Decimals != 18 {
		t.Errorf("Expected 18, got %d", token.Decimals)
	}
}
