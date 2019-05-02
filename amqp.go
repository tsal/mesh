package main

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"pack.ag/amqp"
)

// ---- Consumer

type AmqpConsumer struct {
	Component
	Node     *CNode
	client   *amqp.Client
	receiver *amqp.Receiver
	queue    string
	topic    string
}

func newAmqpConsumer(model Model, node *CNode) (Consumer, error) {
	url, err := getString(model, "url")
	if err != nil {
		return nil, err
	}
	queue, err := getString(model, "queue")
	if err != nil {
		return nil, err
	}
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

func (consumer AmqpConsumer) start() error {
	go func() {
		log.Debug("amqp-consumer-start")
		session, err := consumer.client.NewSession()
		if err != nil {
			log.Error("creating amqp session:", err)
		}
		receiver, err := session.NewReceiver(
			amqp.LinkSourceAddress("/queue-name"),
			amqp.LinkCredit(10),
		)
		if err != nil {
			log.Fatal("creating amqp receiver:", err)
		}
		for {
			consumer.Component.doConsume(consumer.Node,
				func() (Message, error) {
					// Receive next message
					msg, err := receiver.Receive(context.Background())
					if err != nil {
						return Message{}, err
					}
					// Accept message
					msg.Accept()
					return Message{Data: msg.GetData()}, nil
				},
				func(err error) {
				})
		}
	}()
	return nil
}

func (consumer AmqpConsumer) stop() {
	log.Debug("amqp-consumer-stop")
	consumer.receiver.Close(context.Background())
	consumer.client.Close()
}

// --- Producer

type AmqpProducer struct {
	Component
	client  *amqp.Client
	sender  *amqp.Sender
	url     string
	queue   string
	topic   string
	timeout int
}

func newAmqpProducer(model Model) (Producer, error) {
	log.Debug(model.Details)
	url, err := getString(model, "url")
	if err != nil {
		return nil, err
	}
	queue, err := getString(model, "queue")
	if err != nil {
		return nil, err
	}
	return AmqpProducer{
		Component: newComponent(model),
		url:       url,
		queue:     queue}, nil
}

func (producer AmqpProducer) produce(msg Message) error {
	return producer.Component.doProduce(msg,
		func(msg Message) (interface{}, error) {
			return amqp.NewMessage(msg.Data), nil
		},
		func(amqMsg interface{}) error {
			session, err := producer.client.NewSession()
			if err != nil {
				return fmt.Errorf("creating amqp session: %v", err)
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
			err = sender.Send(context.Background(), amqMsg.(*amqp.Message))
			if err != nil {
				return fmt.Errorf("sending message: %v", err)
			}
			return nil
		})
}

func (producer AmqpProducer) start() error {
	log.Debug("amqp-producer-start")
	client, err := amqp.Dial(producer.url)
	//amqp.ConnSASLPlain("access-key-name", "access-key"),
	if err != nil {
		return err
	}
	producer.client = client
	return nil
}

func (producer AmqpProducer) stop() {
	log.Debug("amqp-producer-stop")
	producer.client.Close()
}
