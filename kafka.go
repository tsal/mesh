package main

import (
	"context"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
)

// ---- Consumer

type KafkaConsumer struct {
	Component
	Node    *CNode
	reader  *kafka.Reader
	port    string
	uri     string
	groupID string
}

func newKafkaConsumer(model Model, node *CNode) (Consumer, error) {
	brokers, err := getString(model, "brokers")
	if err != nil {
		return nil, err
	}
	topic, err := getString(model, "topic")
	if err != nil {
		return nil, err
	}
	groupID, err := getString(model, "groupID", NewUUID())
	if err != nil {
		return nil, err
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: strings.Split(brokers, ","),
		GroupID: groupID,
		Topic:   topic})
	consumer := &KafkaConsumer{
		Component: newComponent(model),
		reader:    r,
		Node:      node}
	return consumer, nil
}

func (consumer KafkaConsumer) start() error {
	go func() {
		log.Debug("kafka-consumer-start")
		for {
			consumer.Component.doConsume(consumer.Node,
				func() (Message, error) {
					msg, err := consumer.reader.ReadMessage(context.Background())
					if err != nil {
						return Message{}, err
					}
					headers := map[string][]byte{}
					headers[kafkaKey] = msg.Key
					for _, h := range msg.Headers {
						headers[h.Key] = h.Value
					}
					return Message{Data: msg.Value, Headers: headers}, nil
				},
				func(err error) {
				})
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
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      strings.Split(brokers, ","),
		Topic:        topic,
		WriteTimeout: time.Duration(defaultIfZero(model.Timeout, 10000)),
		Balancer:     &kafka.LeastBytes{},
	})

	return KafkaProducer{
		Component: newComponent(model),
		writer:    w}, nil
}

const (
	kafkaKey = "kafka_key"
)

func newKafkaMessage(msg Message) kafka.Message {
	m := kafka.Message{Value: msg.Data}
	if key := msg.Headers[kafkaKey]; key != nil {
		m.Key = key
	}
	for k, v := range msg.Headers {
		m.Headers = append(m.Headers, kafka.Header{Key: k, Value: v})
	}
	return m
}

func (producer KafkaProducer) produce(inMsg Message) error {
	return producer.Component.doProduce(inMsg,
		func(msg Message) (interface{}, error) {
			return newKafkaMessage(msg), nil
		},
		func(kafkaMsg interface{}) error {
			return producer.writer.WriteMessages(context.Background(), kafkaMsg.(kafka.Message))
		})
}

func (producer KafkaProducer) start() error {
	log.Debug("kafka-producer-start")
	return nil
}

func (producer KafkaProducer) stop() {
	log.Debug("kafka-producer-stop")
}
