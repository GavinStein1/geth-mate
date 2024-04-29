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
	fmt.Println("Getting all Uniswap pools from factory. This may take some time...")
	allPools := eth.GetUniswapPools(client)

	// Create graph
	fmt.Println("Initialising data structures. This may take some time...")
	graph := graph.NewGraph()
	for _, pool := range allPools {
		graph.AddEdge(&pool)
	}
	fmt.Println(len(graph.Nodes))
	fmt.Println(len(graph.Edges))

	// Trim away low liquidity pools
	// To do this, we can run algorithm to get liquidity value in Eth and
	// remove nodes and their edges that are below a threshold.
	if args["trim"] {
		fmt.Println("Trimming data structure.")
		graph.TrimNodes(*new(big.Float).SetInt64(30))
	}
	fmt.Println(len(graph.Nodes))
	fmt.Println(len(graph.Edges))
	return
	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
		case header := <-headers:
			fmt.Println("New block:", header.Number.String())
			for i, pool := range allPools {
				pool.UpdateReserves(client)
				fmt.Println(i)
			}
		}
	}

}
