package main_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"aaaas/pipeline-api/pkg/api/helpers"
	"aaaas/pipeline-api/pkg/api/model"
)

func getActualOutput(pipelinePayload model.PipelinePayload ) (map[string]interface{}){
	nodes := pipelinePayload.Nodes
	edges := pipelinePayload.Edges

	parallels, series := helpers.TraverseGraph(nodes, edges)

	return map[string]interface{}{
		"parallels": parallels,
		"series": series,
	}
}
var _ = Describe("Pipelines", func() {
	It("should test our first example", func() {
		/*
			0 -> 1 -> 2 -> 3

			simple example of a series
		*/
		pipelinePayload := model.PipelinePayload{
			Nodes: []model.Node{
				{ID: "0", Data: model.NodeData{Label: "API Server Source"}},
				{ID: "1", Data: model.NodeData{Label: "FaaS1"}},
				{ID: "2", Data: model.NodeData{Label: "FaaS2"}},
				{ID: "3", Data: model.NodeData{Label: "FaaS3"}},
			},
			Edges: []model.Edge{
				{ID: "0-1", Source: "0", Target: "1"},
				{ID: "1-2", Source: "1", Target: "2"},
				{ID: "2-3", Source: "2", Target: "3"},
			},
		}

		expectedOutput := map[string]interface{}{
			"parallels": map[string][]string{},
			"series": [][]string{
				{"0", "1", "2", "3"},
			},
		}

		actualOutput := getActualOutput(pipelinePayload)

		// Compare the actual output with the expected output
		Expect(actualOutput).To(BeEquivalentTo(expectedOutput))
	})

	It("should test our second example", func() {
		/*
			0 

			if we have just one node, it should return a series of just 0
		*/
		pipelinePayload := model.PipelinePayload{
			Nodes: []model.Node{
				{ID: "0", Data: model.NodeData{Label: "API Server Source"}},
			},
			Edges: []model.Edge{},
		}

		expectedOutput := map[string]interface{}{
			"parallels": map[string][]string{},
			"series": [][]string{
				{"0"},
			},
		}

		actualOutput := getActualOutput(pipelinePayload)

		// Compare the actual output with the expected output
		Expect(actualOutput).To(BeEquivalentTo(expectedOutput))
	})

	It("should test our third example", func() {
		/*
			nothing
			if we pass in nothing, it should return nothing
		*/
		pipelinePayload := model.PipelinePayload{
			Nodes: []model.Node{},
			Edges: []model.Edge{},
		}

		expectedOutput := map[string]interface{}{
			"parallels": map[string][]string{},
			"series": [][]string{},
		}

		actualOutput := getActualOutput(pipelinePayload)

		// Compare the actual output with the expected output
		Expect(actualOutput).To(BeEquivalentTo(expectedOutput))
	})

	It("should test our fourth example", func() {
		/*
				 4
				 ^
				 |
			0 -> 1 -> 5
			|         |
			V         V
			2 -> 3 -> 6
			The function should capture 0 and 1 as parallels because they have 2 child nodes.
			0 has childs [1,2]
			1 has childs [4,5]

			the sequences should be 
			4
			5 -> 6
			2 -> 3 -> 6
		*/
		pipelinePayload := model.PipelinePayload{
			Nodes: []model.Node{
				{ID: "0", Data: model.NodeData{Label: "API Server Source"}},
				{ID: "1", Data: model.NodeData{Label: "FaaS 1"}},
				{ID: "2", Data: model.NodeData{Label: "FaaS 2"}},
				{ID: "3", Data: model.NodeData{Label: "FaaS 3"}},
				{ID: "4", Data: model.NodeData{Label: "FaaS 4"}},
				{ID: "5", Data: model.NodeData{Label: "FaaS 5"}},
				{ID: "6", Data: model.NodeData{Label: "FaaS 6"}},
			},
			Edges: []model.Edge{
				{ID: "0-1", Source: "0", Target: "1"},
				{ID: "0-2", Source: "0", Target: "2"},
				{ID: "2-3", Source: "2", Target: "3"},
				{ID: "1-4", Source: "1", Target: "4"},
				{ID: "1-5", Source: "1", Target: "5"},
				{ID: "3-6", Source: "3", Target: "6"},
				{ID: "5-6", Source: "5", Target: "6"},
			},
		}

		expectedOutput := map[string]interface{}{
			"parallels": map[string][]string{
				"0": {"1", "2"},
				"1": {"4", "5"},
			},
			"series": [][]string{
				{"2", "3", "6"},
				{"4"},
				{"5", "6"},
			},
		}

		actualOutput := getActualOutput(pipelinePayload)

		// Compare the actual output with the expected output
		Expect(actualOutput).To(BeEquivalentTo(expectedOutput))
	})
})
