// Package config centralizes startup wiring: reading config, connecting
// to the DB/Kafka/S3, and building the Echo instance. cmd/server/main.go
// should do nothing but call these and wire the layers together.
package config

import "github.com/spf13/viper"

// NewViper reads config.json if present (see repo root) and lets every
// key be overridden by an env var of the same name — env wins, which is
// what you want for container deploys where config.json is a
// local-dev-only convenience.
func NewViper() *viper.Viper {
	v := viper.New()
	v.SetConfigFile("config.json")
	v.AutomaticEnv()
	_ = v.ReadInConfig() // absence is fine; env vars still work
	return v
}
