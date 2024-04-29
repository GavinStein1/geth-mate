package eth

import (
	"context"
	"encoding/hex"
	"log"
	"math/big"

	"gethmate/utils"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func GetUniswapPools(client *ethclient.Client) []UniswapPool {
	factoryAddress := common.HexToAddress("0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f") // Hardcoded uniswap v2 factory address
	// allPairsLength := getAllPairsLength(factoryAddress, client)
	allPairsLength := 1000
	numRoutines := 1
	ch := make(chan int, numRoutines)
	pools := make([][]UniswapPool, numRoutines)
	tokens := make(map[string]*ERC20Token)

	for i := 0; i < numRoutines; i++ {
		start := i * int(allPairsLength) / numRoutines
		end := (i + 1) * int(allPairsLength) / numRoutines

		pools[i] = make([]UniswapPool, 0)
		go GetPoolsSubRoutine(client, factoryAddress, start, end, &pools[i], tokens, ch)
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

func GetPoolsSubRoutine(client *ethclient.Client, factoryAddress common.Address, start, end int, pools *[]UniswapPool, tokens map[string]*ERC20Token, ch chan int) {
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

func CreateUniswapPair(factoryAddress common.Address, i int, client *ethclient.Client, tokens map[string]*ERC20Token) UniswapPool {
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
