package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// ---- Consumer

type StdConsumer struct {
	Node *CNode
}

func newStdConsumer(model Model, node *CNode) (Consumer, error) {
	consumer := &StdConsumer{Node: node}
	return consumer, nil
}

func (consumer StdConsumer) start() error {
	log.Debug("std-consumer-start")
	go func() {
		for {
			var line string
			fmt.Scanln(&line)
			consumer.Node.consume(Message{Data: []byte(line)})
		}
	}()
	return nil
}

func (consumer StdConsumer) stop() {
	log.Debug("std-consumer-stop")
}

// --- Producer

type StdProducer struct {
}

func newStdProducer(model Model) (Producer, error) {
	return StdProducer{}, nil
}

func (producer StdProducer) produce(msg Message) error {
	fmt.Println(string(msg.Data))
	return nil
}

func (producer StdProducer) start() error {
	log.Debug("std-producer-stop")
	return nil
}

func (producer StdProducer) stop() {
	log.Debug("std-producer-stop")
}
