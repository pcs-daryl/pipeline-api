package handlers

import (
	"aaaas/pipeline-api/pkg/api/helpers"
	"aaaas/pipeline-api/pkg/api/model"
	"net/http"

	"github.com/pcs-aa-aas/commons/pkg/api/server"
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

	parallels, sequences := helpers.TraverseGraph(
		payload.Nodes, payload.Edges)

	return http.StatusOK, map[string]interface{}{
		"message":   "success",
		"parallels": parallels,
		"sequences": sequences,
	}
}
