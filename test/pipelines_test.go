package main_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"aaaas/pipeline-api/pkg/api/handlers"
	"aaaas/pipeline-api/pkg/api/helpers"
	"aaaas/pipeline-api/pkg/api/model"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	flows "knative.dev/eventing/pkg/apis/flows/v1"
	messaging "knative.dev/eventing/pkg/apis/messaging/v1"
	duck "knative.dev/pkg/apis/duck/v1"
	serving "knative.dev/serving/pkg/apis/serving/v1"
)

var testFaasList = []string{"func-0", "func-1", "func-2", "func-3", "func-4", "func-5", "func-6", "func-7"}

var _ = Describe("Pipelines", func() {
	ctx := context.Background()

	BeforeEach(func() {
		By("Creating some test ksvc")
		for _, faasId := range testFaasList {
			Expect(createKsvc(ctx, faasId)).To(Succeed())
		}
	})

	AfterEach(func() {
		By("Cleaning up the env")
		Expect(deleteAllKsvc(ctx)).To(Succeed())
		Expect(deleteAllSequences(ctx)).To(Succeed())
		Expect(deleteAllParallels(ctx)).To(Succeed())
	})

	Context("when verifying the startup environment", func() {
		It("should check that initial ksvc is created successfully (test function)", func() {
			for _, faasId := range testFaasList {
				ksvc, err := getKsvc(ctx, faasId)
				Expect(err).NotTo(HaveOccurred())
				Expect(ksvc.ObjectMeta.Name).To(BeEquivalentTo(faasId))
			}
		})

		It("should check that initial ksvc is created successfully (custom function)", func() {
			node := model.Node{ID: "0", Data: model.NodeData{Label: "func-1", FaasID: "func-1"}}
			ksvc, err := handlers.GetKsvcFromNode(k8sClient, ctx, namespace, &node)
			Expect(err).NotTo(HaveOccurred())
			Expect(ksvc.ObjectMeta.Name).To(BeEquivalentTo("func-1"))
		})
	})

	Context("When handling nodes and edges", func() {
		It("should test our simple sequence", func() {
			/*
				0 -> 1 -> 2 -> 3

				simple example of a sequence
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
				"sequences": [][]string{
					{"0", "1", "2", "3"},
				},
			}

			actualOutput := getActualOutput(pipelinePayload)

			// Compare the actual output with the expected output
			Expect(actualOutput).To(BeEquivalentTo(expectedOutput))
		})

		It("should test a single node", func() {
			/*
				0

				if we have just one node, it should return a sequence of just 0
			*/
			pipelinePayload := model.PipelinePayload{
				Nodes: []model.Node{
					{ID: "0", Data: model.NodeData{Label: "API Server Source"}},
				},
				Edges: []model.Edge{},
			}

			expectedOutput := map[string]interface{}{
				"parallels": map[string][]string{},
				"sequences": [][]string{
					{"0"},
				},
			}

			actualOutput := getActualOutput(pipelinePayload)

			// Compare the actual output with the expected output
			Expect(actualOutput).To(BeEquivalentTo(expectedOutput))
		})

		It("should test a null example", func() {
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
				"sequences": [][]string{},
			}

			actualOutput := getActualOutput(pipelinePayload)

			// Compare the actual output with the expected output
			Expect(actualOutput).To(BeEquivalentTo(expectedOutput))
		})

		It("should test a more complicated tree", func() {
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
				"sequences": [][]string{
					{"2", "3", "6"},
					{"4"},
					{"5", "6"},
				},
			}

			actualOutput := getActualOutput(pipelinePayload)

			// Compare the actual output with the expected output
			Expect(actualOutput).To(BeEquivalentTo(expectedOutput))
		})

		It("should test a BST", func() {
			/*
				     0
					/ \
				   1   2
				  / \  /\
				 3  4  5 6

				In this example, 0, 1 and 2 should return as parallels
				0: [1,2]
				1: [3,4]
				2: [5,6]

				3, 4, 5, 6 should be a single node sequence
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
					{ID: "1-3", Source: "1", Target: "3"},
					{ID: "1-4", Source: "1", Target: "4"},
					{ID: "2-5", Source: "2", Target: "5"},
					{ID: "2-6", Source: "2", Target: "6"},
				},
			}

			expectedOutput := map[string]interface{}{
				"parallels": map[string][]string{
					"0": {"1", "2"},
					"1": {"3", "4"},
					"2": {"5", "6"},
				},
				"sequences": [][]string{
					{"3"},
					{"4"},
					{"5"},
					{"6"},
				},
			}

			actualOutput := getActualOutput(pipelinePayload)

			// Compare the actual output with the expected output
			Expect(actualOutput).To(BeEquivalentTo(expectedOutput))
		})
	})

	Context("When validating the nodes and edges", func() {
		It("should check the sequences against the node list", func() {
			// if we somehow check a node that doesn't exist against the payload, throw an error
			pipelinePayload := model.PipelinePayload{
				Nodes: []model.Node{
					{ID: "0", Data: model.NodeData{Label: "FaaS 0"}},
					{ID: "1", Data: model.NodeData{Label: "FaaS 1"}},
					{ID: "2", Data: model.NodeData{Label: "FaaS 2"}},
				},
				Edges: []model.Edge{
					{ID: "0-1", Source: "0", Target: "1"},
					{ID: "0-2", Source: "0", Target: "2"},
					{ID: "1-2", Source: "1", Target: "2"},
				},
			}
			// invalid node 4
			_, err := handlers.GetNodeByID(pipelinePayload.Nodes, "4")
			Expect(err).To(HaveOccurred())
		})

		It("should check the node list against the FaaS ids (working)", func() {
			// front end should have done the validation so we do not pass in an invalid faas id
			// but we do sanity checks to ensure the faas id is valid
			pipelinePayload := model.PipelinePayload{
				Nodes: []model.Node{
					{ID: "0", Data: model.NodeData{Label: "FaaS 0", FaasID: "func-1"}},
					{ID: "1", Data: model.NodeData{Label: "FaaS 1", FaasID: "func-2"}},
					{ID: "2", Data: model.NodeData{Label: "FaaS 2", FaasID: "func-3"}},
				},
				Edges: []model.Edge{
					{ID: "0-1", Source: "0", Target: "1"},
					{ID: "1-2", Source: "1", Target: "2"},
				},
			}

			_, sequences := helpers.TraverseGraph(
				pipelinePayload.Nodes, pipelinePayload.Edges)

			validNodes, err := handlers.GetValidNodes(ctx, k8sClient, namespace, sequences[0], pipelinePayload.Nodes)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(validNodes)).To(BeEquivalentTo(3))
		})

		It("should check the node list against the FaaS ids (fail)", func() {
			// if faas does not exist we throw an error
			pipelinePayload := model.PipelinePayload{
				Nodes: []model.Node{
					{ID: "0", Data: model.NodeData{Label: "FaaS 0", FaasID: "func-1"}},
					{ID: "1", Data: model.NodeData{Label: "FaaS 1", FaasID: "func-2"}},
					{ID: "2", Data: model.NodeData{Label: "FaaS 2", FaasID: "func-999"}}, //invalid node
				},
				Edges: []model.Edge{
					{ID: "0-1", Source: "0", Target: "1"},
					{ID: "1-2", Source: "1", Target: "2"},
				},
			}

			_, sequences := helpers.TraverseGraph(
				pipelinePayload.Nodes, pipelinePayload.Edges)

			validNodes, err := handlers.GetValidNodes(ctx, k8sClient, namespace, sequences[0], pipelinePayload.Nodes)
			Expect(err).To(HaveOccurred())
			Expect(len(validNodes)).To(BeEquivalentTo(0))
		})
	})

	Context("When managing knative sequences", func() {
		It("should correctly return errors for functions that do not exist", func() {
			node := model.Node{ID: "0", Data: model.NodeData{Label: "abcdef", FaasID: "abcdef"}}
			_, err := handlers.GetKsvcFromNode(k8sClient, ctx, namespace, &node)
			Expect(err).To(HaveOccurred())
		})

		It("should correctly build the knative sequence given a list of valid nodes", func() {
			validFaasIds := []string{"abc", "def", "ghi"}
			sequenceName := "test-sequence"
			sequence := handlers.TranslateSequence(validFaasIds, namespace, sequenceName)

			expectedSteps := []flows.SequenceStep{}

			for _, faasId := range validFaasIds {
				expectedSteps = append(expectedSteps,
					flows.SequenceStep{
						Destination: duck.Destination{
							Ref: &duck.KReference{
								APIVersion: "serving.knative.dev/v1",
								Kind:       "Service",
								Name:       faasId,
							},
						},
					})
			}

			expectedSequence := flows.Sequence{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sequenceName,
					Namespace: namespace,
				},
				Spec: flows.SequenceSpec{
					ChannelTemplate: &messaging.ChannelTemplateSpec{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "messaging.knative.dev/v1",
							Kind:       "InMemoryChannel",
						},
					},
					Steps: expectedSteps,
				},
			}
			Expect(sequence).To(BeEquivalentTo(expectedSequence))
		})

		It("should create the sequence in the cluster", func() {
			validFaasIds := []string{"abc", "def", "ghi"}
			sequenceName := "test-sequence"
			sequence := handlers.TranslateSequence(validFaasIds, namespace, sequenceName)

			err := handlers.ApplySequence(ctx, k8sClient, sequence)
			Expect(err).NotTo(HaveOccurred())

			createdSequence, err := getSequence(ctx, sequenceName)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(createdSequence.Spec.Steps)).To(BeEquivalentTo(3))
		})
	})

	Context("when managing knative parallels", func() {
		It("should handle multiple parallels", func() {
			/*
				0 -> 1
				|
				V
				2
				The function should capture 0  as parallels because they have 2 child nodes.
				0 has childs [1,2]

				the sequences should be
				1,
				2

				we label these nodes beforehand since we are only testing this function
			*/
			pipelinePayload := model.PipelinePayload{
				Nodes: []model.Node{
					{ID: "0", Data: model.NodeData{Label: "func-0", FaasID: "func-0"}},
					{ID: "1", Data: model.NodeData{Label: "func-1", FaasID: "func-1"}, SequenceId: "def"},
					{ID: "2", Data: model.NodeData{Label: "func-5", FaasID: "func-5"}, SequenceId: "abc"},
				},
				Edges: []model.Edge{
					{ID: "0-1", Source: "0", Target: "1"},
					{ID: "0-2", Source: "0", Target: "2"},
				},
			}

			parallels, _ := helpers.TraverseGraph(pipelinePayload.Nodes, pipelinePayload.Edges)
			_, exists := parallels["0"]
			Expect(exists).To(BeTrue())

			branches := parallels["0"]
			parallel := handlers.TranslateParallel(branches, namespace, "random-parallel", pipelinePayload.Nodes)

			Expect(len(parallel.Spec.Branches)).To(BeEquivalentTo(2))
			Expect(parallel.Spec.Branches[0].Subscriber.Ref.Name).To(BeEquivalentTo("def"))
			Expect(parallel.Spec.Branches[1].Subscriber.Ref.Name).To(BeEquivalentTo("abc"))
		})
	})

	Context("When testing the full flow", func() {
		It("should handle simple sequences 1", func() {
			/*
				0 -> 1 -> 2 -> 3

				simple example of a sequence
			*/
			pipelinePayload := model.PipelinePayload{
				Nodes: []model.Node{
					{ID: "0", Data: model.NodeData{Label: "func-1", FaasID: "func-1"}},
					{ID: "1", Data: model.NodeData{Label: "func-2", FaasID: "func-2"}},
					{ID: "2", Data: model.NodeData{Label: "func-3", FaasID: "func-3"}},
					{ID: "3", Data: model.NodeData{Label: "func-4", FaasID: "func-4"}},
				},
				Edges: []model.Edge{
					{ID: "0-1", Source: "0", Target: "1"},
					{ID: "1-2", Source: "1", Target: "2"},
					{ID: "2-3", Source: "2", Target: "3"},
				},
			}

			err := handlers.ProcessPayload(k8sClient, ctx, pipelinePayload, namespace)
			Expect(err).NotTo(HaveOccurred())

			sequenceList, err := getSequenceList(ctx)
			Expect(err).NotTo(HaveOccurred())
			
			//expect one sequence to be deployed
			Expect(sequenceList.Items).To(HaveLen(1))

			steps := sequenceList.Items[0].Spec.Steps
			Expect(len(steps)).To(BeEquivalentTo(4))
			Expect(steps[0].Destination.Ref.Name).To(BeEquivalentTo("func-1"))
			Expect(steps[1].Destination.Ref.Name).To(BeEquivalentTo("func-2"))
			Expect(steps[2].Destination.Ref.Name).To(BeEquivalentTo("func-3"))
			Expect(steps[3].Destination.Ref.Name).To(BeEquivalentTo("func-4"))

			//ensure no parallels created
			parallelList, err := getParallelList(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(parallelList.Items).To(HaveLen(0))
		})

		It("should handle simple sequences 2", func() {
			/*
				4 -> 4 -> 4 -> 4

				simple example of a sequence with repeated functions
			*/
			pipelinePayload := model.PipelinePayload{
				Nodes: []model.Node{
					{ID: "0", Data: model.NodeData{Label: "func-4", FaasID: "func-4"}},
					{ID: "1", Data: model.NodeData{Label: "func-4", FaasID: "func-4"}},
					{ID: "2", Data: model.NodeData{Label: "func-4", FaasID: "func-4"}},
					{ID: "3", Data: model.NodeData{Label: "func-4", FaasID: "func-4"}},
				},
				Edges: []model.Edge{
					{ID: "0-1", Source: "0", Target: "1"},
					{ID: "1-2", Source: "1", Target: "2"},
					{ID: "2-3", Source: "2", Target: "3"},
				},
			}

			err := handlers.ProcessPayload(k8sClient, ctx, pipelinePayload, namespace)
			Expect(err).NotTo(HaveOccurred())

			sequenceList, err := getSequenceList(ctx)
			Expect(err).NotTo(HaveOccurred())

			//expect one sequence to be deployed
			Expect(len(sequenceList.Items)).To(BeEquivalentTo(1))

			steps := sequenceList.Items[0].Spec.Steps
			Expect(len(steps)).To(BeEquivalentTo(4))
			Expect(steps[0].Destination.Ref.Name).To(BeEquivalentTo("func-4"))
			Expect(steps[1].Destination.Ref.Name).To(BeEquivalentTo("func-4"))
			Expect(steps[2].Destination.Ref.Name).To(BeEquivalentTo("func-4"))
			Expect(steps[3].Destination.Ref.Name).To(BeEquivalentTo("func-4"))

			//ensure no parallels created
			parallelList, err := getParallelList(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(parallelList.Items).To(HaveLen(0))
		})

		It("should test a more complicated tree", func() {
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
					{ID: "0", Data: model.NodeData{Label: "func-0", FaasID: "func-0"}},
					{ID: "1", Data: model.NodeData{Label: "func-1", FaasID: "func-1"}},
					{ID: "2", Data: model.NodeData{Label: "func-2", FaasID: "func-2"}},
					{ID: "3", Data: model.NodeData{Label: "func-3", FaasID: "func-3"}},
					{ID: "4", Data: model.NodeData{Label: "func-4", FaasID: "func-4"}},
					{ID: "5", Data: model.NodeData{Label: "func-5", FaasID: "func-5"}},
					{ID: "6", Data: model.NodeData{Label: "func-6", FaasID: "func-6"}},
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

			err := handlers.ProcessPayload(k8sClient, ctx, pipelinePayload, namespace)
			Expect(err).NotTo(HaveOccurred())

			sequenceList, err := getSequenceList(ctx)
			Expect(err).NotTo(HaveOccurred())

			//expect 3 sequence to be deployed
			Expect(sequenceList.Items).To(HaveLen(3))

			// check that the sequence starts are labelled correctly
			Expect(pipelinePayload.Nodes[2].SequenceId).NotTo(BeEquivalentTo(""))
			Expect(pipelinePayload.Nodes[4].SequenceId).NotTo(BeEquivalentTo(""))
			Expect(pipelinePayload.Nodes[5].SequenceId).NotTo(BeEquivalentTo(""))

			Expect(pipelinePayload.Nodes[1].SequenceId).To(BeEquivalentTo(""))
			Expect(pipelinePayload.Nodes[3].SequenceId).To(BeEquivalentTo(""))
			Expect(pipelinePayload.Nodes[6].SequenceId).To(BeEquivalentTo(""))

			// check that there are 2 parallels
			parallelList, err := getParallelList(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(parallelList.Items).To(HaveLen(2))
		})

		It("should test a BST", func() {
			/*
				     0
					/ \
				   1   2
				  / \  /\
				 3  4  5 6

				In this example, 0, 1 and 2 should return as parallels
				0: [1,2]
				1: [3,4]
				2: [5,6]

				3, 4, 5, 6 should be a single node sequence
			*/
			pipelinePayload := model.PipelinePayload{
				Nodes: []model.Node{
					{ID: "0", Data: model.NodeData{Label: "func-0", FaasID: "func-0"}},
					{ID: "1", Data: model.NodeData{Label: "func-1", FaasID: "func-1"}},
					{ID: "2", Data: model.NodeData{Label: "func-2", FaasID: "func-2"}},
					{ID: "3", Data: model.NodeData{Label: "func-3", FaasID: "func-3"}},
					{ID: "4", Data: model.NodeData{Label: "func-4", FaasID: "func-4"}},
					{ID: "5", Data: model.NodeData{Label: "func-5", FaasID: "func-5"}},
					{ID: "6", Data: model.NodeData{Label: "func-6", FaasID: "func-6"}},
				},
				Edges: []model.Edge{
					{ID: "0-1", Source: "0", Target: "1"},
					{ID: "0-2", Source: "0", Target: "2"},
					{ID: "1-3", Source: "1", Target: "3"},
					{ID: "1-4", Source: "1", Target: "4"},
					{ID: "2-5", Source: "2", Target: "5"},
					{ID: "2-6", Source: "2", Target: "6"},
				},
			}

			err := handlers.ProcessPayload(k8sClient, ctx, pipelinePayload, namespace)
			Expect(err).NotTo(HaveOccurred())

			sequenceList, err := getSequenceList(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(sequenceList.Items)).To(BeEquivalentTo(4))
			Expect(len(sequenceList.Items[0].Spec.Steps)).To(BeEquivalentTo(1))
			Expect(len(sequenceList.Items[1].Spec.Steps)).To(BeEquivalentTo(1))
			Expect(len(sequenceList.Items[2].Spec.Steps)).To(BeEquivalentTo(1))
			Expect(len(sequenceList.Items[3].Spec.Steps)).To(BeEquivalentTo(1))

			// check that there are 3 parallels
			parallelList, err := getParallelList(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(parallelList.Items).To(HaveLen(3))
		})
	})
})

func createKsvc(ctx context.Context, funcName string) error {
	ksvc := &serving.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      funcName,
			Namespace: namespace,
		},
		Spec: serving.ServiceSpec{
			ConfigurationSpec: serving.ConfigurationSpec{
				Template: serving.RevisionTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{},
					Spec: serving.RevisionSpec{
						PodSpec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
	}
	return k8sClient.Create(ctx, ksvc)
}

func deleteKsvc(ctx context.Context, funcName string) error {
	ksvc := &serving.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      funcName,
			Namespace: namespace,
		},
		Spec: serving.ServiceSpec{
			ConfigurationSpec: serving.ConfigurationSpec{
				Template: serving.RevisionTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{},
					Spec: serving.RevisionSpec{
						PodSpec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			},
		},
	}
	return k8sClient.Delete(ctx, ksvc)
}

func getKsvc(ctx context.Context, funcName string) (*serving.Service, error) {
	ksvc := &serving.Service{}
	typeNamespacedName := types.NamespacedName{
		Name:      funcName,
		Namespace: namespace,
	}
	err := k8sClient.Get(ctx, typeNamespacedName, ksvc)
	return ksvc, err
}

func getSequence(ctx context.Context, sequenceName string) (*flows.Sequence, error) {
	ksequence := &flows.Sequence{}
	typeNamespacedName := types.NamespacedName{
		Name:      sequenceName,
		Namespace: namespace,
	}
	err := k8sClient.Get(ctx, typeNamespacedName, ksequence)
	return ksequence, err
}

func getSequenceList(ctx context.Context) (*flows.SequenceList, error) {
	ksequenceList := &flows.SequenceList{}
	err := k8sClient.List(ctx, ksequenceList)
	return ksequenceList, err
}

func getParallelList(ctx context.Context) (*flows.ParallelList, error) {
	kparallelList := &flows.ParallelList{}
	err := k8sClient.List(ctx, kparallelList)
	return kparallelList, err
}

func deleteAllKsvc(ctx context.Context) error {
	for _, faasId := range testFaasList{
		if err := deleteKsvc(ctx, faasId); err != nil {
			return fmt.Errorf("failed to delete faas %s in namespace %s: %w",faasId, namespace, err)
		}
	}
	return nil
}

func deleteAllSequences(ctx context.Context) error {
	// Define a list to hold all Knative Sequences
	sequenceList := &flows.SequenceList{}
	// List all sequences in the given namespace
	if err := k8sClient.List(ctx, sequenceList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list sequences in namespace %s: %w", namespace, err)
	}

	// Iterate through each sequence and delete it
	for _, sequence := range sequenceList.Items {
		if err := k8sClient.Delete(ctx, &sequence); err != nil {
			return fmt.Errorf("failed to delete sequence %s in namespace %s: %w", sequence.Name, namespace, err)
		}
	}
	return nil
}

func deleteAllParallels(ctx context.Context) error {
	// Define a list to hold all Knative parallels
	parallelList := &flows.ParallelList{}
	// List all parallels in the given namespace
	if err := k8sClient.List(ctx, parallelList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list parallels in namespace %s: %w", namespace, err)
	}

	// Iterate through each sequence and delete it
	for _, parallel := range parallelList.Items {
		if err := k8sClient.Delete(ctx, &parallel); err != nil {
			return fmt.Errorf("failed to delete parallel %s in namespace %s: %w", parallel.Name, namespace, err)
		}
	}
	return nil
}

func getActualOutput(pipelinePayload model.PipelinePayload) map[string]interface{} {
	nodes := pipelinePayload.Nodes
	edges := pipelinePayload.Edges

	parallels, sequences := helpers.TraverseGraph(nodes, edges)

	return map[string]interface{}{
		"parallels": parallels,
		"sequences": sequences,
	}
}