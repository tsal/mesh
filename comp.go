package main

import (
	log "github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"
)

type Component struct {
	ID        string
	Limiter   ratelimit.Limiter
	Timeout   int
	Filter    string
	Processor string
	Metrics
}

// Throttles by waiting if needed only if Limiter is set
func (c Component) throttle() {
	if c.Limiter != nil {
		c.Limiter.Take()
	}
}

// Filters message, true means accepted
func (c Component) accept(msg Message) (bool, error) {
	if c.Filter == "" {
		return true, nil
	}

	return true, nil
}

// Processes message
func (c Component) process(msg Message) (Message, error) {
	if c.Processor == "" {
		return msg, nil
	}
	return msg, nil
}

//
//
//
//

func (comp Component) doConsume(node *CNode, f1 func() (Message, error), f2 func(error)) {
	comp.throttle()
	msgIn, err := f1()
	if err != nil {
		log.Error(err)
		comp.Metrics.ErrCnt.Inc()
		f2(err)
	}
	accepted, err := comp.accept(msgIn)
	if err != nil {
		log.Error(err)
	}
	if accepted {
		size := float64(msgIn.Size())
		comp.Metrics.BytesIn.Observe(size)
		msgOut, err := comp.process(msgIn)
		if err != nil {
			log.Error(err)
		}
		go func() {
			err = node.consume(msgOut)
			if err != nil {
				log.Error(err)
				comp.Metrics.ErrCnt.Inc()
			}
		}()
	}
}

func (comp Component) doProduce(msgIn Message, f1 func(Message) (interface{}, error), f2 func(rq interface{}) error) error {
	comp.throttle()
	accepted, err := comp.accept(msgIn)
	if err != nil {
		return err
	}
	if accepted {
		size := float64(msgIn.Size())
		comp.Metrics.BytesIn.Observe(size)
		comp.Metrics.MsgSize.Observe(size)
		msgOut, err := comp.process(msgIn)
		if err != nil {
			return err
		}
		rq, err := f1(msgOut)
		if err != nil {
			comp.Metrics.ErrCnt.Inc()
			return err
		}
		log.Debugf("produce: %s", comp.ID)
		err = f2(rq)
		if err != nil {
			comp.Metrics.ErrCnt.Inc()
			return err
		}
		comp.Metrics.BytesOut.Observe(size)
		return nil
	}
	return nil
}
