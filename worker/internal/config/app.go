package config

import "github.com/spf13/viper"

type AppConfig struct {
	Env string
}

func LoadAppConfig(v *viper.Viper) AppConfig {
	v.SetDefault("ENV", "development")
	return AppConfig{Env: v.GetString("ENV")}
}
