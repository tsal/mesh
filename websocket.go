package main

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"pack.ag/amqp"
)

// ---- Consumer

type WsConsumer struct {
	Component
	Node     *CNode
	client   *amqp.Client
	receiver *amqp.Receiver
	queue    string
	topic    string
}

func newWsConsumer(model Model, node *CNode) (Consumer, error) {
	var upgrader = websocket.Upgrader{}
	_ = upgrader
	url := model.Details["url"].(string)
	queue := model.Details["queue"].(string)
	client, err := amqp.Dial(url)
	//amqp.ConnSASLPlain("access-key-name", "access-key"),
	if err != nil {
		return nil, err
	}
	consumer := &AmqpConsumer{
		Component: newComponent(model),
		client:    client,
		queue:     queue,
		Node:      node}
	return consumer, nil
}

func (consumer WsConsumer) start() error {
	go func() {
		log.Debug("amqp-consumer-start")
		session, err := consumer.client.NewSession()
		if err != nil {
			log.Error("creating AMQP session:", err)
		}

		receiver, err := session.NewReceiver(
			amqp.LinkSourceAddress("/queue-name"),
			amqp.LinkCredit(10),
		)
		if err != nil {
			log.Fatal("creating receiver link:", err)
		}

		for {
			// Receive next message
			msg, err := receiver.Receive(context.Background())
			if err != nil {
				log.Fatal("reading message from AMQP:", err)
			}

			// Accept message
			msg.Accept()

			err = consumer.Node.consume(Message{Data: msg.GetData()})
			if err != nil {
				log.Error(err)
			}
		}

	}()
	return nil
}

func (consumer WsConsumer) stop() {
	log.Debug("amqp-consumer-stop")
	consumer.receiver.Close(context.Background())
	consumer.client.Close()
}

// --- Producer

type WsProducer struct {
	Component
	client  *amqp.Client
	sender  *amqp.Sender
	url     string
	queue   string
	topic   string
	timeout int
}

func newWsProducer(model Model) (Producer, error) {
	log.Debug(model.Details)
	url, err := getString(model, "url")
	if err != nil {
		return nil, err
	}
	queue, err := getString(model, "queue")
	if err != nil {
		return nil, err
	}
	timeout, err := getInt(model, "timeout", 1000)
	if err != nil {
		return nil, err
	}
	_ = timeout
	client, err := amqp.Dial(url)
	//amqp.ConnSASLPlain("access-key-name", "access-key"),
	if err != nil {
		return nil, err
	}
	return WsProducer{
		Component: newComponent(model),
		client:    client,
		url:       url,
		queue:     queue}, nil
}

func (producer WsProducer) produce(msg Message) error {
	log.Debug("amqp-produce", producer)
	session, err := producer.client.NewSession()
	if err != nil {
		return fmt.Errorf("creating AMQP session: %v", err)
	}
	defer session.Close(context.Background())
	sender, err := session.NewSender(
		amqp.LinkTargetAddress(producer.queue),
	)
	if err != nil {
		return fmt.Errorf("creating sender link: %v", err)
	}
	defer sender.Close(context.Background())
	//ctx, cancel := context.WithTimeout(ctx, 5*time.Second)

	// Send message
	err = sender.Send(context.Background(), amqp.NewMessage(msg.Data))
	if err != nil {
		return fmt.Errorf("sending message: %v", err)
	}

	return nil
}

func (producer WsProducer) start() error {
	log.Debug("ws-producer-start")
	return nil
}

func (producer WsProducer) stop() {
	log.Debug("ws-producer-stop")
}
