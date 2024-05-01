package eth

import (
	"context"
	"encoding/hex"
	"log"
	"math/big"
	"sync"

	"gethmate/utils"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func GetUniswapPools(client *ethclient.Client) []UniswapPool {
	filename := "addresses.txt"
	addresses, err := utils.ReadAddressesFromFile(filename)
	if err != nil {
		log.Fatalf("Failed to read addresses from file: %v", err)
	}
	numRoutines := 12
	ch := make(chan int, numRoutines)
	pools := make([][]UniswapPool, numRoutines)
	var tokens = &sync.Map{} // TODO: Benchmark whether it is faster to thread and use sync.Map or to run single process with map
	for i := 0; i < numRoutines; i++ {
		start := i * int(len(addresses)) / numRoutines
		end := (i + 1) * int(len(addresses)) / numRoutines

		pools[i] = make([]UniswapPool, 0)
		go GetPoolsSubRoutine(client, &addresses, start, end, &pools[i], tokens, ch)
	}
	// Wait for all goroutines to finish
	for i := 0; i < numRoutines; i++ {
		<-ch
	}

	// Combine all pools
	var allPools []UniswapPool
	for _, p := range pools {
		allPools = append(allPools, p...)
	}

	return allPools
}

func GetUniswapPoolsFromFactory(client *ethclient.Client) []UniswapPool {
	factoryAddress := common.HexToAddress("0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f") // Hardcoded uniswap v2 factory address
	allPairsLength := getAllPairsLength(factoryAddress, client)
	numRoutines := 12
	ch := make(chan int, numRoutines)
	pools := make([][]UniswapPool, numRoutines)
	var tokens = &sync.Map{} // TODO: Benchmark whether it is faster to thread and use sync.Map or to run single process with map

	for i := 0; i < numRoutines; i++ {
		start := i * int(allPairsLength) / numRoutines
		end := (i + 1) * int(allPairsLength) / numRoutines

		pools[i] = make([]UniswapPool, 0)
		go GetPoolsSubRoutineFromFactory(client, factoryAddress, start, end, &pools[i], tokens, ch)
	}

	// Wait for all goroutines to finish
	for i := 0; i < numRoutines; i++ {
		<-ch
	}

	// Combine all pools
	var allPools []UniswapPool
	for _, p := range pools {
		allPools = append(allPools, p...)
	}

	return allPools
}

func GetPoolsSubRoutine(client *ethclient.Client, addresses *[]string, start, end int, pools *[]UniswapPool, tokens *sync.Map, ch chan int) {
	for i := start; i < end; i++ {
		pool := NewUniswapPool((*addresses)[i])
		pool.Initialize(client, tokens)
		if pool.Initialized {
			*pools = append(*pools, *pool)
		}
	}
	ch <- 1
}

func GetPoolsSubRoutineFromFactory(client *ethclient.Client, factoryAddress common.Address, start, end int, pools *[]UniswapPool, tokens *sync.Map, ch chan int) {
	for i := start; i < end; i++ {
		tmpPool := CreateUniswapPair(factoryAddress, i, client, tokens)
		if tmpPool.Initialized {
			*pools = append(*pools, tmpPool)
		}
	}
	ch <- 1
}

func getAllPairsLength(factoryAddress common.Address, client *ethclient.Client) int64 {
	callMsg := ethereum.CallMsg{
		To:   &factoryAddress,
		Data: utils.GetFunctionSelector("allPairsLength()"),
	}

	ctx := context.Background()
	result, err := client.CallContract(ctx, callMsg, nil)
	if err != nil {
		log.Fatalf("1 Utils.go - %v", err)
	}

	allPairsLength := new(big.Int).SetBytes(result)
	return allPairsLength.Int64()
}

func CreateUniswapPair(factoryAddress common.Address, i int, client *ethclient.Client, tokens *sync.Map) UniswapPool {
	callMsg := ethereum.CallMsg{
		To:   &factoryAddress,
		Data: utils.GetFunctionSelector("allPairs(uint256)"),
	}
	callMsg.Data = append(callMsg.Data, common.LeftPadBytes(big.NewInt(int64(i)).Bytes(), 32)...)
	ctx := context.Background()

	result, err := client.CallContract(ctx, callMsg, nil)
	if err != nil {
		log.Fatalf("2 Utils.go - %v", err)
	}

	addr := common.HexToAddress(hex.EncodeToString(result))

	pool := UniswapPool{
		ContractAddress: addr,
	}
	pool.Initialize(client, tokens)
	if !pool.Initialized {
		log.Printf("Failed to initialise pool %s\n", pool.ContractAddress)
	}
	return pool
}
