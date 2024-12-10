package handlers

import (
	"aaaas/pipeline-api/pkg/api/helpers"
	"aaaas/pipeline-api/pkg/api/model"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/pcs-aa-aas/commons/pkg/api/server"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	flows "knative.dev/eventing/pkg/apis/flows/v1"
	messaging "knative.dev/eventing/pkg/apis/messaging/v1"
	duck "knative.dev/pkg/apis/duck/v1"
	serving "knative.dev/serving/pkg/apis/serving/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HandlerGroup struct{}

func (h HandlerGroup) GroupPath() string {
	return "v1"
}

func (h HandlerGroup) HandlerManifests() []server.APIHandlerManifest {
	return []server.APIHandlerManifest{
		{
			Path:        "pipeline",
			HTTPMethod:  http.MethodPost,
			HandlerFunc: h.addPipeline,
		},
	}
}

func (k *HandlerGroup) addPipeline(s *server.APIServer, c *server.APICtx) (code int, obj interface{}) {
	var payload model.PipelinePayload

	//TODO don't hardcode this
	k8sClient := getK8sClient()

	if err := c.ShouldBindJSON(&payload); err != nil {
		return http.StatusBadRequest, err
	}

	// identify the parallels and sequences
	parallels, sequences := helpers.TraverseGraph(
		payload.Nodes, payload.Edges)

	// handle sequences
	// for each service in the sequence, check if it exists
	sequenceIsValid := true
	for i, sequence := range sequences {
		fmt.Println("Checking sequence ", i)

		validNodes := []string{}
		for _, nodeId := range sequence {

			// fetch the node
			node, err := GetNodeByID(payload, nodeId)
			if err != nil {
				sequenceIsValid = false
				fmt.Println("Node error: ", err)
				break
			}

			// ensure it is a valid svc in the cluster
			_, err = validateKsvc(c, k8sClient, "default", node)

			if err != nil {
				sequenceIsValid = false
				fmt.Println("Ksvc not found: ", err)
				break
			}

			// add the nodes to validNodes
			validNodes = append(validNodes, node.Data.FaasID)
		}

		// with the valid nodes, construct our sequence
		sequence := TranslateSequence(validNodes, "knative")
		err := ApplySequence(c, k8sClient, sequence)

		if err != nil {
			fmt.Println("Sequence error: ", err)
			break
		}
	}

	if !sequenceIsValid {
		return http.StatusBadRequest, "Knative services are invalid"
	}

	return http.StatusOK, map[string]interface{}{
		"message":   "success",
		"parallels": parallels,
		"sequences": sequences,
	}
}

func GetNodeByID(payload model.PipelinePayload, id string) (*model.Node, error) {
	for _, node := range payload.Nodes {
		if node.ID == id {
			return &node, nil // Return a pointer to the node
		}
	}
	return nil, fmt.Errorf("Invalid nodes found: node id -> " + id)
}

func validateKsvc(c *server.APICtx, k8sClient client.Client, namespace string, node *model.Node) (*serving.Service, error) {
	//TODO handle k8s client
	return GetKsvcFromNode(k8sClient, c.Context, namespace, node)
}

func GetKsvcFromNode(client client.Client, ctx context.Context, namespace string, node *model.Node) (*serving.Service, error) {
	ksvc := &serving.Service{}

	typeNamespacedName := types.NamespacedName{
		Name:      node.Data.FaasID,
		Namespace: namespace,
	}

	err := client.Get(ctx, typeNamespacedName, ksvc)
	if err != nil {
		return &serving.Service{}, err
	}
	return ksvc, err
}

func getK8sClient() client.Client {
	//TODO handle this kubeconfig part
	kubeconfigPath := "/home/administrator/Documents/pipeline-api/conf/supervisorconf"
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Fatalf("Error reading kubeconfig: %v", err)
	}

	err = serving.AddToScheme(scheme.Scheme)

	// Create the controller-runtime client
	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}
	return k8sClient
}

func ApplySequence(ctx context.Context, k8sClient client.Client, sequence flows.Sequence) error {
	return k8sClient.Create(ctx, &sequence)
}

func TranslateSequence(faasIdList []string, namespace string) flows.Sequence {
	steps := []flows.SequenceStep{}

	for _, faasId := range faasIdList {
		steps = append(steps,
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

	return flows.Sequence{
		ObjectMeta: v1.ObjectMeta{
			Name:      "randomly-generate-here",
			Namespace: namespace,
		},
		Spec: flows.SequenceSpec{
			ChannelTemplate: &messaging.ChannelTemplateSpec{
				TypeMeta: v1.TypeMeta{
					APIVersion: "messaging.knative.dev/v1",
					Kind:       "InMemoryChannel",
				},
			},
			Steps: steps,
		},
	}
}
