package eth

import (
	"context"
	"encoding/hex"
	"log"
	"math"
	"math/big"
	"strings"
	"sync"

	"gethmate/utils"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type UniswapPool struct {
	ContractAddress common.Address `json:"contract_address"`
	Token0          *ERC20Token    `json:"token0"`
	Token1          *ERC20Token    `json:"token1"`
	Reserve0        *big.Int       `json:"reserve0"`
	Reserve1        *big.Int       `json:"reserve1"`
	Initialized     bool
}

func NewUniswapPool(contractAddress string) *UniswapPool {
	return &UniswapPool{
		ContractAddress: common.HexToAddress(contractAddress),
		Initialized:     false,
	}
}

func (u *UniswapPool) Initialize(client *ethclient.Client, tokens *sync.Map) {
	// Token0 address
	callMsg := ethereum.CallMsg{
		To:   &u.ContractAddress,
		Data: utils.GetFunctionSelector("token0()"),
	}

	ctx := context.Background()
	result, err := client.CallContract(ctx, callMsg, nil)
	if err != nil || len(result) == 0 {
		log.Printf("Failed to get token0 for %s: %v\n", u.ContractAddress, err)
		return
	}
	t0Addr := common.HexToAddress(hex.EncodeToString(result))
	t0, exists := tokens.Load(strings.ToLower(t0Addr.String()))
	if !exists {
		u.Token0 = NewERC20Token(t0Addr)
		u.Token0.Initialize(client)
		if !u.Token0.Initalized {
			log.Printf("Failed to initialise token0 %s\n", t0Addr.String())
			return
		}
		tokens.Store(strings.ToLower(t0Addr.String()), u.Token0)
	} else {
		u.Token0 = t0.(*ERC20Token)
	}

	// Token1 address
	callMsg.To = &u.ContractAddress
	callMsg.Data = utils.GetFunctionSelector("token1()")
	result, err = client.CallContract(ctx, callMsg, nil)
	if err != nil || len(result) == 0 {
		log.Printf("Failed to get token1 for %s: %v\n", u.ContractAddress, err)
		return
	}
	t1Addr := common.HexToAddress(hex.EncodeToString(result))
	t1, exists := tokens.Load(strings.ToLower(t1Addr.String()))
	if !exists {
		u.Token1 = NewERC20Token(t1Addr)
		u.Token1.Initialize(client)
		if !u.Token1.Initalized {
			log.Printf("Failed to initialise token1 %s\n", t1Addr.String())
			return
		}
		tokens.Store(strings.ToLower(t0Addr.String()), u.Token0)
	} else {
		u.Token1 = t1.(*ERC20Token)
	}

	// Reserves
	u.UpdateReserves(client)
	u.Initialized = true
}

func (u *UniswapPool) UpdateReserves(client *ethclient.Client) {
	callMsg := ethereum.CallMsg{
		To:   &u.ContractAddress,
		Data: utils.GetFunctionSelector("getReserves()"),
	}

	ctx := context.Background()
	result, err := client.CallContract(ctx, callMsg, nil)
	if err != nil || len(result) == 0 {
		log.Fatalf("Failed to get reserves for %s: %v", u.ContractAddress, err)
	}

	u.Reserve0 = new(big.Int).SetBytes(result[0:32])
	u.Reserve1 = new(big.Int).SetBytes(result[32:64])
}

func (u UniswapPool) GetK() big.Int {
	return *new(big.Int).Mul(u.Reserve0, u.Reserve1)
}

func (u UniswapPool) GetToken0Price() big.Float {
	if u.Reserve0.Cmp(big.NewInt(0)) == 0 || u.Reserve1.Cmp(big.NewInt(0)) == 0 {
		return *big.NewFloat(0)
	}
	priceRatio := new(big.Float).Quo(new(big.Float).SetInt(u.Reserve1), new(big.Float).SetInt(u.Reserve0))
	priceRatio.Mul(priceRatio, big.NewFloat(math.Pow10(u.Token0.Decimals-u.Token1.Decimals)))
	return *priceRatio
}

func (u UniswapPool) GetToken1Price() big.Float {
	if u.Reserve0.Cmp(big.NewInt(0)) == 0 || u.Reserve1.Cmp(big.NewInt(0)) == 0 {
		return *big.NewFloat(0)
	}
	priceRatio := new(big.Float).Quo(new(big.Float).SetInt(u.Reserve0), new(big.Float).SetInt(u.Reserve1))
	priceRatio.Mul(priceRatio, big.NewFloat(math.Pow10(u.Token1.Decimals-u.Token0.Decimals)))
	return *priceRatio
}

func (u UniswapPool) GetPrice(tokenIn string) big.Float {
	if strings.EqualFold(tokenIn, u.Token0.ContractAddress.String()) {
		return u.GetToken0Price()
	} else if strings.EqualFold(tokenIn, u.Token1.ContractAddress.String()) {
		return u.GetToken1Price()
	} else {
		log.Println("Token not in pool")
		return big.Float{}
	}
}

func (u UniswapPool) GetToken1Out(token0Amount big.Int) big.Float {
	r0f := new(big.Float).SetInt(u.Reserve0)
	r1f := new(big.Float).SetInt(u.Reserve1)
	token0Amountf := new(big.Float).SetInt(&token0Amount)

	k := new(big.Float).Mul(r0f, r1f)
	denominator := new(big.Float).Add(token0Amountf, r0f)
	k.Quo(k, denominator)

	result := new(big.Float).Sub(r1f, k)
	return *result
}

func (u UniswapPool) GetToken0Out(token1Amount big.Int) big.Float {
	r0f := new(big.Float).SetInt(u.Reserve0)
	r1f := new(big.Float).SetInt(u.Reserve1)
	token1Amountf := new(big.Float).SetInt(&token1Amount)

	k := new(big.Float).Mul(r0f, r1f)
	denominator := new(big.Float).Add(token1Amountf, r1f)
	k.Quo(k, denominator)

	result := new(big.Float).Sub(r0f, k)
	return *result
}

func (u UniswapPool) GetTokenAmountOut(tokenIn ERC20Token, amountIn big.Int) big.Float {
	if strings.EqualFold(tokenIn.ContractAddress.String(), u.Token0.ContractAddress.String()) {
		return u.GetToken1Out(amountIn)
	} else if strings.EqualFold(tokenIn.ContractAddress.String(), u.Token1.ContractAddress.String()) {
		return u.GetToken0Out(amountIn)
	} else {
		log.Fatalf("Token %s is not in pool %s", tokenIn.ContractAddress, u.ContractAddress)
		return big.Float{}
	}
}

func (u UniswapPool) GetReservesFromTokenContract(contractAddress string) big.Int {
	if strings.EqualFold(contractAddress, u.Token0.ContractAddress.String()) {
		return *u.Reserve0
	} else if strings.EqualFold(contractAddress, u.Token1.ContractAddress.String()) {
		return *u.Reserve1
	} else {
		return *big.NewInt(0)
	}
}
