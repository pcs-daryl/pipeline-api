package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"aaaas/pipeline-api/pkg/api/helpers"
	"aaaas/pipeline-api/pkg/api/model"
)

var _ = Describe("Pipelines", func() {
	// You can load test data from JSON files here
	var inputJSON []byte

	// This will be run before each test case
	BeforeEach(func() {
		// Load the input and expected output JSON from files (test cases)
		var err error
		inputJSON, err = os.ReadFile(filepath.Join("test", "testdata", "input.json"))
		Expect(err).NotTo(HaveOccurred())
	})

	// The actual test for the function
	It("should correctly identify nodes with multiple outgoing edges and single path chains", func() {
		expectedOutput := map[string]interface{}{
			"multipleOutgoingEdgesNodes": map[string][]string{
				"1": {"2", "3"},
				"2": {"5", "6"},
			},
			"singlePathNodes": [][]string{
				{"3", "4", "7"},
				{"5"},
				{"6", "7"},
			},
		}
		// Unmarshal input JSON into the corresponding Go structs
		var pipelinePayload model.PipelinePayload
		err := json.Unmarshal(inputJSON, &pipelinePayload)
		Expect(err).NotTo(HaveOccurred())

		// Call the function to test
		multipleOutgoingEdgesNodes, singlePathNodes := helpers.FindNodesWithMultipleOutgoingEdgesAndNeighborsAndSinglePathNodes(pipelinePayload.Nodes, pipelinePayload.Edges)

		// Marshal the actual output back into JSON for comparison
		actualOutput := map[string]interface{}{
			"multipleOutgoingEdgesNodes": multipleOutgoingEdgesNodes,
			"singlePathNodes":            singlePathNodes,
		}

		Expect(err).NotTo(HaveOccurred())

		// Compare the actual output with the expected output
		Expect(actualOutput).To(BeEquivalentTo(expectedOutput))
	})
})
