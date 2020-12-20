package api

import (
	"context"
	"log"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// Config defines all possible values the studios service expects
type Config struct {
	HTTPPort uint `mapstructure:"HTTP_PORT"`
}

func setDefaultConfigs() {
	viper.SetDefault("HTTP_PORT", 8080)
}

func GetConfig(ctx context.Context) (cfg Config, err error) {
	viper.SetConfigName("universal-studios")
	viper.AutomaticEnv()
	setDefaultConfigs()

	all := viper.AllSettings()
	if err = mapstructure.WeakDecode(all, &cfg); err != nil {
		return
	}

	log.Printf("Initiating app with configs: %+v\n", cfg)
	return
}
