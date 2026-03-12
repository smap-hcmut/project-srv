package kafka

import (
	"fmt"
	"sync"

	"project-srv/config"

	"github.com/smap-hcmut/shared-libs/go/kafka"
)

var (
	producerInstance kafka.IProducer
	consumerInstance kafka.IConsumer
	producerOnce     sync.Once
	consumerOnce     sync.Once
	producerMu       sync.RWMutex
	consumerMu       sync.RWMutex
	producerInitErr  error
	consumerInitErr  error
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

// ConnectConsumer initializes and connects to Kafka consumer using singleton pattern.
func ConnectConsumer(cfg config.KafkaConfig) (kafka.IConsumer, error) {
	consumerMu.Lock()
	defer consumerMu.Unlock()

	if consumerInstance != nil {
		return consumerInstance, nil
	}

	if consumerInitErr != nil {
		consumerOnce = sync.Once{}
		consumerInitErr = nil
	}

	var err error
	consumerOnce.Do(func() {
		consumer, e := kafka.NewConsumer(kafka.ConsumerConfig{
			Brokers: cfg.Brokers,
			GroupID: cfg.GroupID,
		})
		if e != nil {
			err = fmt.Errorf("failed to create Kafka consumer: %w", e)
			consumerInitErr = err
			return
		}
		consumerInstance = consumer
	})

	return consumerInstance, err
}

// GetProducer returns the singleton Kafka producer instance.
func GetProducer() kafka.IProducer {
	producerMu.RLock()
	defer producerMu.RUnlock()

	if producerInstance == nil {
		panic("Kafka producer not initialized. Call ConnectProducer() first")
	}
	return producerInstance
}

// GetConsumer returns the singleton Kafka consumer instance.
func GetConsumer() kafka.IConsumer {
	consumerMu.RLock()
	defer consumerMu.RUnlock()

	if consumerInstance == nil {
		panic("Kafka consumer not initialized. Call ConnectConsumer() first")
	}
	return consumerInstance
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

// DisconnectConsumer closes the Kafka consumer and resets the singleton.
func DisconnectConsumer() error {
	consumerMu.Lock()
	defer consumerMu.Unlock()

	if consumerInstance != nil {
		if err := consumerInstance.Close(); err != nil {
			return err
		}
		consumerInstance = nil
		consumerOnce = sync.Once{}
		consumerInitErr = nil
	}
	return nil
}
