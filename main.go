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

// MapEdgesBySource groups edges by their source
func MapEdgesBySource(edges []model.Edge) map[string][]map[string]string {
	edgeMap := make(map[string][]map[string]string)

	for _, edge := range edges {
		// Create a new map for id and target
		edgeInfo := map[string]string{
			"id":     edge.ID,
			"target": edge.Target,
		}

		// Append the edge information to the slice for the corresponding source
		edgeMap[edge.Source] = append(edgeMap[edge.Source], edgeInfo)
	}

	return edgeMap
}

// FindNodesWithMultipleOutgoingEdgesAndNeighbors identifies nodes with multiple outgoing edges and their neighbors
func FindNodesWithMultipleOutgoingEdgesAndNeighbors(nodes []model.Node, edges []model.Edge) map[string][]string {
	// Map to store outgoing neighbors for each node
	outgoingNeighbors := make(map[string][]string)

	// Populate the outgoingNeighbors map
	for _, edge := range edges {
		outgoingNeighbors[edge.Source] = append(outgoingNeighbors[edge.Source], edge.Target)
	}

	// Filter nodes with more than one outgoing edge and collect neighbors
	result := make(map[string][]string)
	for _, node := range nodes {
		neighbors := outgoingNeighbors[node.ID]
		if len(neighbors) > 1 {
			result[node.ID] = neighbors
		}
	}

	return result
}

// FindNodesWithMultipleOutgoingEdgesAndNeighbors identifies nodes with multiple outgoing edges and their neighbors,
// and identifies single path nodes (nodes part of chains), excluding already visited nodes.
func FindNodesWithMultipleOutgoingEdgesAndNeighborsAndSinglePathNodes(
	nodes []model.Node, edges []model.Edge) (map[string][]string, map[string][]string) {

	// Map to store outgoing neighbors for each node
	outgoingNeighbors := make(map[string][]string)
	// Map to track nodes with multiple outgoing edges
	multipleOutgoingEdgesNodes := make(map[string][]string)
	// Map to store single path nodes
	singlePathNodes := make(map[string][]string)

	// Populate the outgoingNeighbors map
	for _, edge := range edges {
		outgoingNeighbors[edge.Source] = append(outgoingNeighbors[edge.Source], edge.Target)
	}

	// Identify nodes with multiple outgoing edges and collect their neighbors
	for _, node := range nodes {
		neighbors := outgoingNeighbors[node.ID]
		if len(neighbors) > 1 {
			multipleOutgoingEdgesNodes[node.ID] = neighbors
		}
	}

	// Identify single path nodes (excluding already visited nodes)
	visited := make(map[string]bool)
	for _, node := range nodes {
		// Skip nodes already identified as part of multiple outgoing edges
		if _, found := multipleOutgoingEdgesNodes[node.ID]; found {
			continue
		}

		// Track single path nodes (nodes forming a chain)
		if len(outgoingNeighbors[node.ID]) == 1 && !visited[node.ID] {
			chain := []string{}
			currentNode := node.ID
			// Traverse the chain of nodes with exactly one outgoing edge
			for {
				if nextNode, exists := outgoingNeighbors[currentNode]; exists && len(nextNode) == 1 {
					nextNodeID := nextNode[0]
					// Skip nodes already in a previous chain (already visited)
					if visited[nextNodeID] {
						break
					}
					chain = append(chain, nextNodeID)
					visited[currentNode] = true
					currentNode = nextNodeID
				} else {
					// If no outgoing edge (or multiple edges), we stop the chain
					break
				}
			}
			// If we found a chain, include all the nodes in the result
			if len(chain) > 0 {
				singlePathNodes[node.ID] = chain
				// Mark all nodes in the chain as visited to prevent future traversal
				for _, n := range chain {
					visited[n] = true
				}
			}
		}

		// If a node has no outgoing edges and is not part of a multiple-edge node, it's a single node path
		if len(outgoingNeighbors[node.ID]) == 0 && !visited[node.ID] {
			singlePathNodes[node.ID] = []string{} // Add an empty array for nodes with no outgoing edges
		}
	}

	return multipleOutgoingEdgesNodes, singlePathNodes
}
