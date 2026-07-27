package config

import (
	"strings"

	"github.com/spf13/viper"

	"github.com/multimodal-recsys/catalog-writer-api/internal/messaging"
)

func NewKafkaProducer(v *viper.Viper) *messaging.KafkaProducer {
	brokers := strings.Split(v.GetString("KAFKA_BROKERS"), ",")
	topic := v.GetString("KAFKA_TOPIC")
	return messaging.NewKafkaProducer(brokers, topic)
}
