package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"gethmate/eth"
	"gethmate/graph"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type BlockNumberResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Result  string `json:"result"`
}

func main() {
	args := make(map[string]bool)
	for _, arg := range os.Args {
		args[arg] = true
	}
	wsClient, err := ethclient.Dial("ws://localhost:8546")
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client (ws): %v", err)
	}
	defer wsClient.Close()

	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client (http): %v", err)
	}
	defer client.Close()

	headers := make(chan *types.Header)
	ctx := context.Background()
	sub, err := wsClient.SubscribeNewHead(ctx, headers)
	if err != nil {
		log.Fatal(err)
	}

	// Get uniswap pools
	fmt.Printf("Starting GethMate.\nTimestamp: %s\n", time.Now())
	fmt.Println("Getting all Uniswap pools. This may take some time...")
	// allPools := eth.GetUniswapPools()
	allPools := eth.GetUniswapPools(client)

	// Create graph
	fmt.Println("Initialising data structures. This may take some time...")
	graph := graph.NewGraph()
	for _, pool := range allPools {
		graph.AddEdge(&pool)
	}

	// Trim away low liquidity pools
	// To do this, we can run algorithm to get liquidity value in Eth and
	// remove nodes and their edges that are below a threshold.
	// (TAKES AGES...)
	if args["trim"] {
		fmt.Println("Trimming data structure.")
		graph.TrimNodes(*new(big.Float).SetInt64(300))
	}
	startAmountIn := new(big.Float).SetFloat64(0.1)

	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
		case header := <-headers:
			blockNumber := header.Number
			fmt.Println("New block:", blockNumber.String())

			// Update edge weights for new block
			graph.UpdateAllEdges(client)

			// Find arbitrage path
			graph.Strategy(startAmountIn)
		}
	}
}
