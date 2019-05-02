package main

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

// ---- Consumer

type MqttConsumer struct {
	Component
	Node     *CNode
	client   mqtt.Client
	url      string
	clientID string
	topic    string
}

func newMqttClient(url string, handler mqtt.MessageHandler) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().AddBroker(url)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetDefaultPublishHandler(handler)
	opts.SetPingTimeout(1 * time.Second)
	return mqtt.NewClient(opts), nil
}

func newMqttConsumer(model Model, node *CNode) (Consumer, error) {
	url, err := getString(model, "url")
	if err != nil {
		return nil, err
	}
	clientID, err := getString(model, "clientID")
	if err != nil {
		return nil, err
	}
	_ = clientID
	consumer := &MqttConsumer{
		Component: newComponent(model),
		Node:      node}

	handler := func(client mqtt.Client, msg mqtt.Message) {
		consumer.Component.doConsume(consumer.Node,
			func() (Message, error) {
				return Message{Data: msg.Payload()}, nil
			},
			func(err error) {
			})
	}

	client, err := newMqttClient(url, handler)
	if err != nil {
		return nil, err
	}
	consumer.client = client
	return consumer, nil
}

func (consumer MqttConsumer) start() error {
	go func() {
		log.Debug("mqtt-consumer-start")
		if token := consumer.client.Subscribe(consumer.topic, 0, nil); token.Wait() && token.Error() != nil {
			log.Error(token.Error())
		}
	}()
	return nil
}

func (consumer MqttConsumer) stop() {
	log.Debug("mqtt-consumer-stop")
	if token := consumer.client.Unsubscribe(consumer.topic); token.Wait() && token.Error() != nil {
		log.Error(token.Error())
	}
	consumer.client.Disconnect(250)
}

// --- Producer

type MqttProducer struct {
	Component
	client  mqtt.Client
	topic   string
	timeout int
}

func newMqttProducer(model Model) (Producer, error) {
	log.Debug(model.Details)
	url, err := getString(model, "url")
	if err != nil {
		return nil, err
	}
	clientID, err := getString(model, "clientID")
	_ = clientID
	if err != nil {
		return nil, err
	}
	client, err := newMqttClient(url, nil)
	if err != nil {
		return nil, err
	}
	return MqttProducer{
		Component: newComponent(model),
		client:    client}, nil
}

func (producer MqttProducer) produce(inMsg Message) error {
	return producer.Component.doProduce(inMsg,
		func(msg Message) (interface{}, error) {
			return msg.Data, nil
		},
		func(data interface{}) error {
			if token := producer.client.Connect(); token.Wait() && token.Error() != nil {
				return fmt.Errorf("%v", token.Error())
			}
			defer producer.client.Disconnect(250)
			token := producer.client.Publish(producer.topic, 0, false, data)
			token.WaitTimeout(time.Duration(defaultIfZero(producer.Timeout, 10000)))
			return token.Error()
		})
}

func (producer MqttProducer) start() error {
	log.Debug("mqtt-producer-start")
	return nil
}

func (producer MqttProducer) stop() {
	log.Debug("mqtt-producer-stop")
	producer.client.Disconnect(250)
}
