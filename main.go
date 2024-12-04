package main

import (
	"aaaas/pipeline-api/pkg/api/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// Create a Gin router instance
	router := gin.Default()

	// Define the /pipeline POST route
	router.POST("/pipeline", func(c *gin.Context) {
		var payload model.PipelinePayload

		// Attempt to bind the JSON payload
		if err := c.ShouldBindJSON(&payload); err != nil {
			// Log and return a binding error
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Identify nodes with multiple outgoing edges and their neighbors, and single path nodes
		multipleOutgoingEdgesNodes, singlePathNodes := FindNodesWithMultipleOutgoingEdgesAndNeighborsAndSinglePathNodes(
			payload.Nodes, payload.Edges)

		// Respond with the results
		c.JSON(http.StatusOK, gin.H{
			"message":                    "success",
			"multipleOutgoingEdgesNodes": multipleOutgoingEdgesNodes,
			"singlePathNodes":            singlePathNodes,
		})
	})

	// Start the server on port 8080
	router.Run(":8080")
}

// FindNodesWithMultipleOutgoingEdgesAndNeighbors identifies nodes with multiple outgoing edges and their neighbors,
// and identifies single path nodes (nodes part of chains), excluding already visited nodes.
func FindNodesWithMultipleOutgoingEdgesAndNeighborsAndSinglePathNodes(
	nodes []model.Node, edges []model.Edge) (map[string][]string, [][]string) {

	// Map to store outgoing neighbors for each node
	outgoingNeighbors := make(map[string][]string)
	// Map to store incoming neighbors for each node
	incomingNeighbors := make(map[string][]string)
	// Map to track nodes with multiple outgoing edges
	multipleOutgoingEdgesNodes := make(map[string][]string)
	// Slice to store single path chains
	var singlePathChains [][]string

	// Populate the outgoingNeighbors and incomingNeighbors maps
	for _, edge := range edges {
		outgoingNeighbors[edge.Source] = append(outgoingNeighbors[edge.Source], edge.Target)
		incomingNeighbors[edge.Target] = append(incomingNeighbors[edge.Target], edge.Source)
	}

	// Identify nodes with multiple outgoing edges and collect their neighbors
	for _, node := range nodes {
		neighbors := outgoingNeighbors[node.ID]
		if len(neighbors) > 1 {
			multipleOutgoingEdgesNodes[node.ID] = neighbors
		}
	}

	// Function to traverse a single chain
	traverseChain := func(startNode string) []string {
		chain := []string{}
		currentNode := startNode
		visited := make(map[string]bool)

		for {
			// Add the current node to the chain
			chain = append(chain, currentNode)
			visited[currentNode] = true

			// Get the next node
			nextNodes, exists := outgoingNeighbors[currentNode]
			if !exists || len(nextNodes) != 1 {
				break // Stop if no outgoing edge or multiple outgoing edges
			}

			nextNode := nextNodes[0]
			if visited[nextNode] {
				break // Prevent infinite loops
			}

			currentNode = nextNode
		}
		return chain
	}

	// Find all chains starting from nodes with zero or one outgoing edge
	visitedGlobal := make(map[string]bool)
	for _, node := range nodes {
		if visitedGlobal[node.ID] {
			continue
		}

		// Only start chains from nodes with zero or one outgoing edge
		if len(outgoingNeighbors[node.ID]) <= 1 {
			chain := traverseChain(node.ID)

			// Mark all nodes in the chain as globally visited
			for _, n := range chain {
				visitedGlobal[n] = true
			}

			// Add the chain to the result
			singlePathChains = append(singlePathChains, chain)
		}
	}

	return multipleOutgoingEdgesNodes, singlePathChains
}