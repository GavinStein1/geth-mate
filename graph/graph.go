package graph

import (
	"container/list"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"strings"

	"gethmate/eth"

	"github.com/ethereum/go-ethereum/ethclient"
)

type Graph struct {
	Nodes map[string]*Node
	Edges map[string]*Edge
}

type Node struct {
	Token *eth.ERC20Token
	Edges []*Edge
}

type Edge struct {
	Start *Node
	Dest  *Node
	Pool  *eth.UniswapPool
}

func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[string]*Node),
		Edges: make(map[string]*Edge),
	}
}

func (g *Graph) GetNode(tokenAddress string) *Node {
	node, exists := g.Nodes[strings.ToLower(tokenAddress)]
	if !exists {
		return nil
	} else {
		return node
	}
}

func (g *Graph) GetEdge(poolAddress string) *Edge {
	edge, exists := g.Edges[strings.ToLower(poolAddress)]
	if !exists {
		return nil
	} else {
		return edge
	}
}

func (g *Graph) AddEdge(pool *eth.UniswapPool) {
	_, exists := g.Edges[pool.ContractAddress.String()]
	if exists {
		return
	}
	t0AddressLower := strings.ToLower(pool.Token0.ContractAddress.String())
	t1AddressLower := strings.ToLower(pool.Token1.ContractAddress.String())
	startNode, exists := g.Nodes[t0AddressLower]
	if !exists {
		g.Nodes[t0AddressLower] = &Node{
			Token: pool.Token0,
			Edges: make([]*Edge, 0),
		}
		startNode = g.Nodes[t0AddressLower]
	}
	destNode, exists := g.Nodes[t1AddressLower]
	if !exists {
		g.Nodes[t1AddressLower] = &Node{
			Token: pool.Token1,
			Edges: make([]*Edge, 0),
		}
		destNode = g.Nodes[t1AddressLower]
	}

	edge := &Edge{
		Start: startNode,
		Dest:  destNode,
		Pool:  pool,
	}

	g.Edges[strings.ToLower(pool.ContractAddress.String())] = edge

	startNode.Edges = append(startNode.Edges, edge)
	destNode.Edges = append(destNode.Edges, edge)
}

func (g *Graph) RemoveEdge(edge *Edge) {
	start := edge.Start
	dest := edge.Dest
	delete(g.Edges, strings.ToLower(edge.Pool.ContractAddress.String()))
	for i, e := range start.Edges {
		if strings.EqualFold(e.Pool.ContractAddress.String(), edge.Pool.ContractAddress.String()) {
			start.Edges = append(start.Edges[:i], start.Edges[i+1:]...)
			break
		}
	}
	if len(start.Edges) == 0 {
		delete(g.Nodes, strings.ToLower(start.Token.ContractAddress.String()))
	}
	for i, e := range dest.Edges {
		if strings.EqualFold(e.Pool.ContractAddress.String(), edge.Pool.ContractAddress.String()) {
			dest.Edges = append(dest.Edges[:i], dest.Edges[i+1:]...)
			break
		}
	}
	if len(dest.Edges) == 0 {
		delete(g.Nodes, strings.ToLower(dest.Token.ContractAddress.String()))
	}
}

func (g *Graph) TrimNodes(threshold big.Float) {
	src, exists := g.Nodes["0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"] // Hardcoded WETH contract on Eth mainnet
	if !exists {
		log.Fatalln("WETH not found in graph")
	}

	// Modified Breadth-First Search (BFS) for trimming
	price_in_eth := make(map[*Node]*big.Float)

	// Initialize distances and predecessors
	for _, node := range g.Nodes {
		price_in_eth[node] = big.NewFloat(math.Inf(1))
	}
	price_in_eth[src] = big.NewFloat(1)
	queue := list.New()
	queue.PushBack(src)
	visited := make(map[*Node]bool)
	visited[src] = true
	i := 1
	for queue.Len() > 0 {
		fmt.Printf("Iteration %d\n", i)
		i += 1
		// Get current node
		current := queue.Front().Value.(*Node)
		queue.Remove(queue.Front())
		fmt.Printf("Edges: %d\n", len(current.Edges))
		j := 0
		for _, edge := range current.Edges {
			fmt.Printf("Edge %d\n", j)
			j += 1
			var neighbor *Node
			if strings.EqualFold(current.Token.ContractAddress.String(), edge.Start.Token.ContractAddress.String()) {
				neighbor = edge.Dest
			} else {
				neighbor = edge.Start
			}
			if !visited[neighbor] {
				visited[neighbor] = true
				price := edge.Pool.GetPrice(current.Token.ContractAddress.String())
				reserves := edge.Pool.GetReservesFromTokenContract(neighbor.Token.ContractAddress.String())
				reservesF := new(big.Float).SetInt(&reserves)
				reservesF.Quo(reservesF, big.NewFloat(math.Pow10(neighbor.Token.Decimals)))
				if price.Cmp(big.NewFloat(0)) == 0 {
					// setting the price to negative 1 will render reservesF < 0 < threshold
					price.SetInt64(-1)
				}
				reservesF.Quo(reservesF, &price)
				if reservesF.Cmp(&threshold) == -1 {
					// Remove edge and set price to -1 so that remaining edges are marked for removal
					g.RemoveEdge(edge)
					price_in_eth[neighbor].SetInt64(-1)
				} else {
					price_in_eth[neighbor].Mul(price_in_eth[current], &price)
					queue.PushBack(neighbor)
				}
			}
		}
	}
	// Print addresses of nodes that are still in graph to file
	file, err := os.OpenFile("addresses.txt", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("Failed to open file")
	}
	defer file.Close()
	for _, edge := range g.Edges {
		file.WriteString(edge.Pool.ContractAddress.String() + "\n")
	}
	err = file.Sync()
	if err != nil {
		log.Println("Failed to write to file")
	}
	fmt.Println("Trimmed graph")
}

func (g *Graph) UpdateAllEdges(client *ethclient.Client) {
	numRoutines := 24
	ch := make(chan int, numRoutines)
	keys := make([]string, 0, len(g.Edges))
	for key := range g.Edges {
		keys = append(keys, key)
	}
	for i := 0; i < numRoutines; i++ {
		start := i * int(len(g.Edges)) / numRoutines
		end := (i + 1) * int(len(g.Edges)) / numRoutines
		go g.updateEdges(client, keys, start, end, ch)
	}

	// Wait for all goroutines to finish
	for i := 0; i < numRoutines; i++ {
		<-ch
	}
}

func (g *Graph) updateEdges(client *ethclient.Client, keys []string, start, end int, ch chan int) {

	for i := start; i < end; i++ {
		edge := g.Edges[keys[i]]
		edge.Pool.UpdateReserves(client)
	}
	ch <- 1
}
