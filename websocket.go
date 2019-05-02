package main

import (
	"fmt"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// ---- Consumer

type WsConsumer struct {
	Component
	Node     *CNode
	Upgrader websocket.Upgrader
	Port     string
	Uri      string
}

func newWsConsumer(model Model, node *CNode) (Consumer, error) {
	port, err := getInt(model, "port", 8080)
	if err != nil {
		return nil, err
	}
	uri, err := getString(model, "uri", "/")
	if err != nil {
		return nil, err
	}
	consumer := &WsConsumer{
		Component: newComponent(model),
		Node:      node,
		Port:      fmt.Sprintf("%d", port),
		Uri:       uri,
		Upgrader:  websocket.Upgrader{}}
	return consumer, nil
}

func (consumer WsConsumer) start() error {
	go func() {
		log.Debug("ws-consumer-start")
		//TODO
	}()
	return nil
}

func (consumer WsConsumer) stop() {
	log.Debug("ws-consumer-stop")
}

// --- Producer

type WsProducer struct {
	Component
	url         string
	Conn        *websocket.Conn
	MessageType string
}

func newWsProducer(model Model) (Producer, error) {
	log.Debug(model.Details)
	url, err := getString(model, "url")
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return WsProducer{
		Component: newComponent(model),
		url:       url}, nil
}

func (producer WsProducer) produce(msgIn Message) error {
	return producer.Component.doProduce(msgIn,
		func(msgOut Message) (interface{}, error) {
			//TODO ?
			return msgOut.Data, nil
		},
		func(data interface{}) error {
			return producer.Conn.WriteMessage(websocket.TextMessage, data.([]byte))
		})
}

func (producer WsProducer) start() error {
	log.Debug("ws-producer-start")
	conn, _, err := websocket.DefaultDialer.Dial(producer.url, nil)
	if err != nil {
		return err
	}
	producer.Conn = conn
	return nil
}

func (producer WsProducer) stop() {
	log.Debug("ws-producer-stop")
	producer.Conn.Close()
}
