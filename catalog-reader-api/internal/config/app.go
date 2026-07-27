package config

import "github.com/spf13/viper"

type AppConfig struct {
	Port string
	Env  string // "development" | "production"
}

func LoadAppConfig(v *viper.Viper) AppConfig {
	v.SetDefault("PORT", "8080")
	v.SetDefault("ENV", "development")
	return AppConfig{Port: v.GetString("PORT"), Env: v.GetString("ENV")}
}
