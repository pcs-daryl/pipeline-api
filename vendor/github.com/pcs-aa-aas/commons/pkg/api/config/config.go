package config

import (
	"os"

	"github.com/pcs-aa-aas/commons/pkg/log"
	"github.com/pkg/errors"
	"gopkg.in/ini.v1"
)

type ServerConfig interface {
	GetApiUri() string
	GetSwaggerUrl() string
	GetSwaggerHandlerUrl() string
	GetSwaggerPath() string
	GetKubeconfigUrl() string
	GetServerConfigImpl() ServerConfig
}

func ReadConfigFile(path string) *ini.File {
	configPath := os.Getenv("CONFIG_PATH")
	if len(configPath) == 0 {
		configPath = path
	}
	cfg, err := ini.Load(configPath)
	if err != nil {
		log.Error(errors.Wrapf(err, "loading %s", configPath))
		os.Exit(1)
	}
	return cfg
}

func MapSections(configFile *ini.File, sections []string, serverConfig ServerConfig) ServerConfig {
	for _, section := range sections {
		err := configFile.Section(section).MapTo(serverConfig)
		if err != nil {
			log.Error(errors.Wrapf(err, "Loading validation config fail."))
			os.Exit(1)
		}
	}
	return serverConfig
}
