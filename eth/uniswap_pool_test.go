package eth

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
)

func TestUniswapPool(t *testing.T) {
	fmt.Println("TestUniswapPool")
	poolAddr := "0x0d4a11d5eeaac28ec3f61d100daf4d40471f1852"
	pool := NewUniswapPool(poolAddr)
	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		t.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	tokens := make(map[string]*ERC20Token)
	pool.Initialize(client, tokens)
	if !strings.EqualFold(pool.Token0.ContractAddress.String(), "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2") {
		t.Errorf("Expected 0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2, got %s", pool.Token0.ContractAddress.String())
	}
	if !strings.EqualFold(pool.Token1.ContractAddress.String(), "0xdac17f958d2ee523a2206206994597c13d831ec7") {
		t.Errorf("Expected 0xdac17f958d2ee523a2206206994597c13d831ec7, got %s", pool.Token1.ContractAddress.String())
	}

	fmt.Printf("Need to manually check reserves with real time values\n")
	fmt.Printf("Reserve0: %v\n", pool.Reserve0)
	fmt.Printf("Reserve1: %v\n", pool.Reserve1)
}
