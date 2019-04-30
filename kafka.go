package main

import (
	"context"
	"strings"

	kafka "github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
)

// ---- Consumer

type KafkaConsumer struct {
	Component
	Node   *CNode
	reader *kafka.Reader
	port   string
	uri    string
}

func newKafkaConsumer(model Model, node *CNode) (Consumer, error) {
	brokers := model.Details["brokers"].(string)
	topic := model.Details["topic"].(string)
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   strings.Split(brokers, ","),
		Topic:     topic,
		Partition: 0,
		MinBytes:  10e3, // 10KB
		MaxBytes:  10e6, // 10MB
	})
	consumer := &KafkaConsumer{
		Component: Component{
			ID:      model.ID,
			Metrics: newMetrics(model.ID)},
		reader: r,
		Node:   node}
	return consumer, nil
}

func (consumer KafkaConsumer) start() error {
	go func() {
		log.Debug("kafka-consumer-start")
		for {
			m, err := consumer.reader.ReadMessage(context.Background())
			if err != nil {
				break
			}
			err = consumer.Node.consume(Message{Data: m.Value}) //TODO: key!
			if err != nil {
				log.Error(err)
			}
		}
	}()
	return nil
}

func (consumer KafkaConsumer) stop() {
	log.Debug("kafka-consumer-stop")
	consumer.reader.Close()
}

// --- Producer

type KafkaProducer struct {
	Component
	writer  *kafka.Writer
	timeout int
}

func newKafkaProducer(model Model) (Producer, error) {
	log.Debug(model.Details)
	brokers, err := getString(model, "brokers")
	if err != nil {
		return nil, err
	}
	topic, err := getString(model, "topic")
	if err != nil {
		return nil, err
	}
	timeout, err := getInt(model, "timeout", 1000)
	if err != nil {
		return nil, err
	}
	_ = timeout

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  strings.Split(brokers, ","),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})

	return KafkaProducer{
		Component: Component{ID: model.ID,
			Metrics: newMetrics(model.ID)},
		writer: w}, nil
}

const (
	KEY = "kafka_key"
)

func newKafkaMessage(msg Message) kafka.Message {
	if key := msg.Headers[KEY]; key != nil {
		return kafka.Message{
			Key:   key,
			Value: msg.Data,
		}
	}
	return kafka.Message{
		Value: msg.Data,
	}
}

func (producer KafkaProducer) produce(msg Message) error {
	log.Debug("kafka-produce")
	return producer.writer.WriteMessages(context.Background(), newKafkaMessage(msg))
}

func (producer KafkaProducer) start() error {
	log.Debug("kafka-producer-start")
	return nil
}

func (producer KafkaProducer) stop() {
	log.Debug("kafka-producer-stop")
}
