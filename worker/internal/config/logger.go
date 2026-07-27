package config

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func NewLogger(v *viper.Viper) *logrus.Logger {
	log := logrus.New()
	if v.GetString("ENV") == "production" {
		log.SetFormatter(&logrus.JSONFormatter{})
	}
	return log
}
