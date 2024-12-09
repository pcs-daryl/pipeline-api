package config

import (
	cfg "github.com/pcs-aa-aas/commons/pkg/api/config"
)

type ServerConfigImpl struct {
	ApiUri            string `ini:"api_uri"`
	SwaggerUrl        string `ini:"swagger_url"`
	SwaggerHandlerUrl string `ini:"swagger_handler_url"`
	SwaggerPath       string `ini:"swagger_path"`
	TokenTTL          string `ini:"token_ttl"`
	KubeConfigUrl     string `ini:"kubeconfig_url"`
}

func (sc *ServerConfigImpl) GetApiUri() string {
	return sc.ApiUri
}

func (sc *ServerConfigImpl) GetSwaggerUrl() string {
	return sc.SwaggerUrl
}

func (sc *ServerConfigImpl) GetSwaggerHandlerUrl() string {
	return sc.SwaggerHandlerUrl
}

func (sc *ServerConfigImpl) GetSwaggerPath() string {
	return sc.SwaggerPath
}

func (sc *ServerConfigImpl) GetServerConfigImpl() cfg.ServerConfig {
	return sc
}

func (sc *ServerConfigImpl) GetKubeconfigUrl() string {
	return sc.KubeConfigUrl
}

func NewServerConfigImpl() *ServerConfigImpl {
	return &ServerConfigImpl{}
}
