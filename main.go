package main

import (
	"aaaas/pipeline-api/pkg/api/helpers"
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
		parallels, sequences := helpers.TraverseGraph(
			payload.Nodes, payload.Edges)

		// Respond with the results
		c.JSON(http.StatusOK, gin.H{
			"message":                    "success",
			"parallels": parallels,
			"sequences": sequences,
		})
	})

	// Start the server on port 8080
	router.Run(":8080")
}

