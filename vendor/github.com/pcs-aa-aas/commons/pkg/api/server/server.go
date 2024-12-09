package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pcs-aa-aas/commons/pkg/api/config"
	"github.com/pcs-aa-aas/commons/pkg/api/dto"
	"github.com/pcs-aa-aas/commons/pkg/api/middleware"
	"github.com/pcs-aa-aas/commons/pkg/log"
	"github.com/pkg/errors"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	healthcheck "github.com/RaMin0/gin-health-check"
	"github.com/gin-gonic/gin"
)

type APIServer struct {
	SuperClientset kubernetes.Interface
	Router         *gin.Engine
	Server         *http.Server
	Config         config.ServerConfig
}

func NewAPIServer(config config.ServerConfig, supervisorConfigPath string, middlewareConfig *config.MiddlewareConfig) (*APIServer, error) {

	app := gin.New()
	app.Use(gin.Logger())
	app.Use(gin.Recovery())
	app.Use(healthcheck.Default())

	if middlewareConfig.EnableCORS {
		app.Use(middleware.CORSMiddleware())
	}

	if middlewareConfig.EnableLogger {
		app.Use(middleware.LoggerMiddleware())
	}

	if middlewareConfig.EnableKubeconfig {
		app.Use(middleware.KubeconfigMiddleware(config.GetKubeconfigUrl()))
	}

	// For Swagger support
	swaggerUrl := config.GetSwaggerUrl()
	if len(swaggerUrl) > 0 {
		url := ginSwagger.URL(swaggerUrl)
		app.GET(config.GetSwaggerHandlerUrl(), ginSwagger.WrapHandler(swaggerFiles.Handler, url))
		app.StaticFile(swaggerUrl, config.GetSwaggerPath())
		log.Info("registered swagger route")
	}

	var client *kubernetes.Clientset
	if len(supervisorConfigPath) > 0 {
		cfg, err := clientcmd.BuildConfigFromFlags("", supervisorConfigPath)
		if err != nil {
			log.Fatal("unable to build config for supervisor cluster", err)
		}
		client, err = kubernetes.NewForConfig(cfg)
		if err != nil {
			log.Fatal("unable to instantiate clientset for supervisor cluster", err)
		}
	}

	apiServer := &APIServer{
		Router:         app,
		Server:         nil,
		SuperClientset: client,
		Config:         config,
	}

	return apiServer, nil
}

func (a *APIServer) Listen() {
	a.Server = &http.Server{
		Addr:    a.Config.GetApiUri(),
		Handler: a.Router,
	}

	go func() {
		err := a.Server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")
}

func (a *APIServer) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.Server.Shutdown(ctx); err != nil {
		log.Fatal("APIServer forced to shutdown:", err)
	}
}

func toGinHandler(s *APIServer, manifest APIHandlerManifest) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		out := dto.ResponseMsg{}

		code, obj := manifest.HandlerFunc(s, &APICtx{
			Context: ctx,
		})

		out.Status = float64(code)

		if code < 200 || code >= 300 {
			out.ErrorMsg = fmt.Sprintf("%v", obj)
			ctx.JSON(code, out)
			return
		} else {
			out.Result = obj
			ctx.JSON(code, out)
		}
		ctx.Next()
	}
}

func (a *APIServer) SetupRoutes(apiGroups []APIHandlerGroup) {
	for _, apiGroup := range apiGroups {

		grp := a.Router.Group(apiGroup.GroupPath())

		for _, manifest := range apiGroup.HandlerManifests() {

			switch manifest.HTTPMethod {
			case http.MethodGet:
				grp.GET(manifest.Path, toGinHandler(a, manifest))
			case http.MethodPost:
				grp.POST(manifest.Path, toGinHandler(a, manifest))
			case http.MethodPut:
				grp.PUT(manifest.Path, toGinHandler(a, manifest))
			case http.MethodPatch:
				grp.PATCH(manifest.Path, toGinHandler(a, manifest))
			case http.MethodDelete:
				grp.DELETE(manifest.Path, toGinHandler(a, manifest))
			case http.MethodOptions:
				grp.OPTIONS(manifest.Path, toGinHandler(a, manifest))
			}
		}
	}
}

func SetupServer(cfg config.ServerConfig, supervisorConfigPath string, middlewareConfig *config.MiddlewareConfig) *APIServer {
	server, err := NewAPIServer(cfg, supervisorConfigPath, middlewareConfig)
	if err != nil {
		log.Error(errors.Wrap(err, "creating api server"))
		os.Exit(1)
	}

	return server
}

func Run(apiConfigPath string, configSections []string, routes []APIHandlerGroup, serverCfgImpl config.ServerConfig, supKubeconfigPath string) {
	serverConfig := MapConfig(apiConfigPath, configSections, serverCfgImpl)
	middlewareConfig := config.NewMiddlewareConfig()
	SetUpServerAndRun(serverConfig, supKubeconfigPath, middlewareConfig, routes)
}

func RunWithMiddlewareConfigs(apiConfigPath string, configSections []string, routes []APIHandlerGroup, serverCfgImpl config.ServerConfig, supKubeconfigPath string, middlewareConfig *config.MiddlewareConfig) {
	serverConfig := MapConfig(apiConfigPath, configSections, serverCfgImpl)
	SetUpServerAndRun(serverConfig, supKubeconfigPath, middlewareConfig, routes)
}

func MapConfig(apiConfigPath string, configSections []string, serverCfgImpl config.ServerConfig) config.ServerConfig {
	cfg := config.ReadConfigFile(apiConfigPath)
	serverConfig := config.MapSections(cfg, configSections, serverCfgImpl)
	log.Debugf("APIServer Conf: %+v", serverConfig)
	log.Debugf("Host URL: %+v", serverConfig.GetApiUri())
	return serverConfig
}

func SetUpServerAndRun(serverConfig config.ServerConfig, supKubeconfigPath string, middlewareConfig *config.MiddlewareConfig, routes []APIHandlerGroup) {
	server := SetupServer(serverConfig, supKubeconfigPath, middlewareConfig)
	server.SetupRoutes(routes)

	defer server.Close()

	server.Listen()
}

// ServeRequest is used for writing tests.
func (a *APIServer) ServeRequest(httpMtd, url string, bodyObj, respObj interface{}) (code int, err error) {
	msg, _ := json.Marshal(bodyObj)
	req, _ := http.NewRequest(httpMtd, url, bytes.NewBuffer(msg))
	req.Header.Add("Content-Type", "application/json")

	w := httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	rawResp, err := io.ReadAll(w.Result().Body)
	if err != nil {
		return 0, err
	}

	resp := dto.ResponseMsg{}
	if err := json.Unmarshal(rawResp, &resp); err != nil {
		fmt.Printf("%s", rawResp)
		return 0, errors.WithMessagef(err, "%s", rawResp)
	}

	code = int(resp.Status)
	if resp.Status < 200 || resp.Status >= 300 {
		return code, errors.New(resp.ErrorMsg)
	}

	respBod, err := json.Marshal(resp.Result)
	if err != nil {
		return code, errors.Wrap(err, "converting resp data back to bytes")
	}
	if err := json.Unmarshal(respBod, respObj); err != nil {
		return code, errors.Wrap(err, "converting resp data bytes into respObj")
	}

	return code, nil
}
