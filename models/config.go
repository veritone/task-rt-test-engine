package models

import "time"

type Config struct {
	PrometheusPort   int           `json:"prometheusPort"`
	LogFormat        string        `json:"logFormat"`
	LogLevel         string        `json:"logLevel"`
	EngineID         string        `json:"engineId"`
	EngineInstanceID string        `json:"engineInstanceId"`
	Kafka            KafkaConfig   `json:"kafka"`
	MaxConcurrency   int           `json:"maxConcurrency"`
	VeritoneBaseUri  string        `json:"veritoneBaseUri"`
	TTLinSec         time.Duration `json:"ttl"`
}

type KafkaConfig struct {
	ConsumerTopic   string `json:"consumerTopic"`
	Brokers         string `json:"brokers"`
	ConsumerGroupId string `json:"consumerGroupId"`
	ProducerTopic   string `json:"producerTopic"`
}
