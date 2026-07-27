package config

import "github.com/spf13/viper"

func NewViper() *viper.Viper {
	v := viper.New()
	v.SetConfigFile("config.json")
	v.AutomaticEnv()
	_ = v.ReadInConfig()
	return v
}
