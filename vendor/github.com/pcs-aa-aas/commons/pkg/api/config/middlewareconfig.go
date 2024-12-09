package config

type MiddlewareConfig struct {
	EnableCORS       bool
	EnableLogger     bool
	EnableKubeconfig bool
}

type Option func(*MiddlewareConfig)

func NewMiddlewareConfig(options ...Option) *MiddlewareConfig {
	// Options Defaults the configs to true
	config := &MiddlewareConfig{
		EnableCORS:       true,
		EnableLogger:     true,
		EnableKubeconfig: true,
	}

	for _, option := range options {
		option(config)
	}

	return config
}

// Options Setters
func DisableCORSMiddleware() Option {
	return func(mc *MiddlewareConfig) {
		mc.EnableCORS = false
	}
}

func DisableLoggerMiddleware() Option {
	return func(mc *MiddlewareConfig) {
		mc.EnableLogger = false
	}
}

func DisableKubeconfigMiddleware() Option {
	return func(mc *MiddlewareConfig) {
		mc.EnableKubeconfig = false
	}
}
