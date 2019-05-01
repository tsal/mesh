package main

import (
	"time"

	log "github.com/sirupsen/logrus"
)

type TickerConsumer struct {
	Component
	Node     *CNode
	ticker   *time.Ticker
	interval int
}

func newTickerConsumer(model Model, node *CNode) (Consumer, error) {
	interval, err := getInt(model, "interval", 1500)
	if err != nil {
		return nil, err
	}
	consumer := TickerConsumer{Node: node}
	consumer.ticker = time.NewTicker(time.Duration(interval) * time.Millisecond)
	return consumer, nil
}

func (consumer TickerConsumer) start() error {
	go func() {
		log.Debug("ticker-consumer-start")
		for t := range consumer.ticker.C {
			err := consumer.Node.consume(Message{Data: []byte(t.String())})
			if err != nil {
				log.Error(err)
			}
		}
	}()
	return nil
}

func (consumer TickerConsumer) stop() {
	log.Debug("ticker-consumer-stop")
	consumer.ticker.Stop()
}
