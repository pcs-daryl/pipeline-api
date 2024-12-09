package server

import (
	"github.com/gin-gonic/gin"
)

// An SourceHandlerGroup is a group of handlers that deal with the same resources and share the same group path.
// An example group path will be /api/v1/auth
type APIHandlerGroup interface {
	GroupPath() string
	HandlerManifests() []APIHandlerManifest
}

type APIHandlerManifest struct {
	Path        string
	HTTPMethod  string
	HandlerFunc APIHandlerFunc
}

type APICtx struct {
	*gin.Context
}

type APIHandlerFunc func(s *APIServer, c *APICtx) (code int, obj interface{})
