package kafka

import (
	"fmt"
	"sync"

	"project-srv/config"

	"github.com/smap-hcmut/shared-libs/go/kafka"
)

var (
	producerInstance kafka.IProducer
	producerOnce     sync.Once
	producerMu       sync.RWMutex
	producerInitErr  error
)

// ConnectProducer initializes and connects to Kafka producer using singleton pattern.
func ConnectProducer(cfg config.KafkaConfig) (kafka.IProducer, error) {
	producerMu.Lock()
	defer producerMu.Unlock()

	if producerInstance != nil {
		return producerInstance, nil
	}

	if producerInitErr != nil {
		producerOnce = sync.Once{}
		producerInitErr = nil
	}

	var err error
	producerOnce.Do(func() {
		producer, e := kafka.NewProducer(kafka.Config{
			Brokers: cfg.Brokers,
			Topic:   cfg.Topic,
		})
		if e != nil {
			err = fmt.Errorf("failed to create Kafka producer: %w", e)
			producerInitErr = err
			return
		}
		producerInstance = producer
	})

	return producerInstance, err
}

// DisconnectProducer closes the Kafka producer and resets the singleton.
func DisconnectProducer() error {
	producerMu.Lock()
	defer producerMu.Unlock()

	if producerInstance != nil {
		if err := producerInstance.Close(); err != nil {
			return err
		}
		producerInstance = nil
		producerOnce = sync.Once{}
		producerInitErr = nil
	}
	return nil
}
