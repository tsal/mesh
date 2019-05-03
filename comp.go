package main

import (
	"io/ioutil"
	"strings"
	"time"

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

func newComponent(model Model) Component {
	return Component{
		ID:        model.ID,
		Limiter:   newLimiter(model.Rate),
		Timeout:   model.Timeout,
		Filter:    model.Filter,
		Processor: model.Processor,
		Metrics:   newMetrics(model.ID)}
}

// if Limiter is set -> throttle by waiting if needed
func (comp *Component) throttle() {
	if comp.Limiter != nil {
		comp.Limiter.Take()
	}
}

// handle file or inlined JS
func getJsCode(value string) (string, error) {
	if strings.HasSuffix(strings.ToLower(value), ".js") {
		b, err := ioutil.ReadFile(value)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return value, nil
}

// Filters message, true means accepted, false rejected
func (comp Component) accept(msg Message) (bool, error) {
	if comp.Filter == "" {
		return true, nil
	}
	code, err := getJsCode(comp.Filter)
	if err != nil {
		return false, err
	}
	log.Debugf("code: %v", code)
	t0 := time.Now()
	accepted, err := evalFilter("filter_"+comp.ID, code, msg)
	metrics.JsTime.Observe(float64(time.Since(t0)) / float64(time.Millisecond))
	//TODO diag header
	return accepted, err
}

// Processes message
func (comp Component) process(msg Message) (Message, error) {
	if comp.Processor == "" {
		return msg, nil
	}
	code, err := getJsCode(comp.Processor)
	if err != nil {
		return Message{}, err
	}
	log.Debugf("code: %v", code)
	t0 := time.Now()
	msgOut, err := evalProcess("process_"+comp.ID, code, msg)
	metrics.JsTime.Observe(float64(time.Since(t0)) / float64(time.Millisecond))
	//TODO diag header
	return msgOut, err
}

//
//
//
//

func (comp *Component) doConsume(node *CNode, f1 func() (Message, error), f2 func(error)) {
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
		comp.Metrics.ErrCnt.Inc()
	}
	if accepted {
		size := float64(msgIn.Size())
		comp.Metrics.BytesIn.Observe(size)
		msgOut, err := comp.process(msgIn)
		if err != nil {
			log.Error(err)
			comp.Metrics.ErrCnt.Inc()
			return
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

func (comp *Component) doProduce(msgIn Message, f1 func(Message) (interface{}, error), f2 func(interface{}) error) error {
	comp.throttle()
	accepted, err := comp.accept(msgIn)
	if err != nil {
		log.Error(err)
		comp.Metrics.ErrCnt.Inc()
		return err
	}
	if accepted {
		size := float64(msgIn.Size())
		comp.Metrics.BytesIn.Observe(size)
		comp.Metrics.MsgSize.Observe(size)
		msgOut, err := comp.process(msgIn)
		if err != nil {
			log.Error(err)
			comp.Metrics.ErrCnt.Inc()
			return err
		}
		rq, err := f1(msgOut)
		if err != nil {
			log.Error(err)
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
