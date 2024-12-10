package handlers

import (
	"aaaas/pipeline-api/pkg/api/helpers"
	"aaaas/pipeline-api/pkg/api/model"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/pcs-aa-aas/commons/pkg/api/server"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	knative "knative.dev/serving/pkg/apis/serving/v1"

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
		for _, nodeId := range sequence {

			node, err := GetNodeByID(payload, nodeId)
			if err != nil {
				sequenceIsValid = false
				fmt.Println("Node error: ", err)
				break
			}

			_, err = validateKsvc(c, "default", node)

			//TODO handle the invalid node data error?

			if err != nil {
				sequenceIsValid = false
				fmt.Println("Ksvc not found: ", err)
				break
			}
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

func validateKsvc(c *server.APICtx, namespace string, node *model.Node) (*knative.Service, error) {
	//TODO handle this kubeconfig part
	kubeconfigPath := "/home/administrator/Documents/pipeline-api/conf/supervisorconf"
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Fatalf("Error reading kubeconfig: %v", err)
	}

	err = knative.AddToScheme(scheme.Scheme)

	// Create the controller-runtime client
	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}
	return GetKsvcFromNode(k8sClient, c.Context, namespace, node)
}

func GetKsvcFromNode(client client.Client, ctx context.Context, namespace string, node *model.Node) (*knative.Service, error) {
	ksvc := &knative.Service{}

	typeNamespacedName := types.NamespacedName{
		Name:      node.Data.FaasID,
		Namespace: namespace,
	}

	err := client.Get(ctx, typeNamespacedName, ksvc)
	if err != nil {
		return &knative.Service{}, err
	}
	return ksvc, err
}
