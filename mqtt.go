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
	client   *mqtt.Client
	url      string
	clientID string
	topic    string
}

func newMqttClient(url, clientID string) (*mqtt.Client, error) {
	opts := mqtt.NewClientOptions().AddBroker(url).SetClientID(clientID)
	opts.SetKeepAlive(2 * time.Second)
	//opts.SetDefaultPublishHandler(f)
	opts.SetPingTimeout(1 * time.Second)
	client := mqtt.NewClient(opts)
	return &client, nil
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
	client, err := newMqttClient(url, clientID)
	if err != nil {
		return nil, err
	}
	consumer := &MqttConsumer{
		Component: newComponent(model),
		client:    client,
		Node:      node}
	return consumer, nil
}

func (consumer MqttConsumer) start() error {
	go func() {
		log.Debug("mqtt-consumer-start")

	}()
	return nil
}

func (consumer MqttConsumer) stop() {
	log.Debug("mqtt-consumer-stop")

}

// --- Producer

type MqttProducer struct {
	Component
	client  *mqtt.Client
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
	if err != nil {
		return nil, err
	}
	client, err := newMqttClient(url, clientID)
	if err != nil {
		return nil, err
	}
	return MqttProducer{
		Component: newComponent(model),
		client:    client}, nil
}

func (producer MqttProducer) produce(msg Message) error {
	log.Debug("mqtt-produce")
	c := *producer.client
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("%v", token.Error())
	}
	defer c.Disconnect(250)
	token := c.Publish(producer.topic, 0, false, msg.Data)
	token.Wait() //FIXME timeout
	return nil
}

func (producer MqttProducer) start() error {
	log.Debug("mqtt-producer-start")
	return nil
}

func (producer MqttProducer) stop() {
	log.Debug("mqtt-producer-stop")
}
