package handlers

import (
	"aaaas/pipeline-api/pkg/api/helpers"
	"aaaas/pipeline-api/pkg/api/model"
	"fmt"
	"net/http"

	"github.com/pcs-aa-aas/commons/pkg/api/server"
	"k8s.io/apimachinery/pkg/api/errors"
	knative "knative.dev/serving/pkg/apis/serving/v1"
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

			//TODO get the ksvc and do something with it
			_, err := getKnativeService(s, c, "default", payload, nodeId)

			//TODO handle the invalid node data error?

			if errors.IsNotFound(err) {
				sequenceIsValid = false
				fmt.Println("Ksvc not found in default namespace: ", nodeId)
			} else {
				fmt.Println("Found ksvc in default namespace")
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

func getNodeByID(payload model.PipelinePayload, id string) *model.Node {
	for _, node := range payload.Nodes {
		if node.ID == id {
			return &node // Return a pointer to the node
		}
	}
	return nil // Return nil if no node with the given ID is found
}

func getKnativeService(s *server.APIServer, c *server.APICtx, namespace string, payload model.PipelinePayload, nodeId string) (knative.Service, error) {
	node := getNodeByID(payload, nodeId)
	if node == nil {
		return knative.Service{}, fmt.Errorf("Invalid node data")
	}
	clientset := s.SuperClientset
	ksvc := knative.Service{}
	err := clientset.AdmissionregistrationV1().RESTClient().
		Get().
		AbsPath("/apis/serving.knative.dev/v1").
		Namespace(namespace).
		Resource("services").
		Name(node.Data.FaasID).
		Do(c.Context).
		Into(&ksvc)
	return ksvc, err
}
