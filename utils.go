package main

import (
	"fmt"
	"reflect"

	"go.uber.org/ratelimit"

	log "github.com/sirupsen/logrus"
)

func getString(model Model, name string, deflt ...string) (string, error) {
	x := model.Details[name]
	if x == nil {
		if len(deflt) == 0 {
			return "", fmt.Errorf("mandatory parameter not provided: %s", name)
		}
		log.Debugf("param not found: %s, defaulting to: %s", name, deflt[0])
		return deflt[0], nil
	}
	xt := reflect.TypeOf(x).Kind()
	if xt != reflect.String {
		return "", fmt.Errorf("parameter is not string: %s", name)
	}
	return model.Details[name].(string), nil
}

func getInt(model Model, name string, deflt ...int) (int, error) {
	x := model.Details[name]
	if x == nil {
		if len(deflt) == 0 {
			return 0, fmt.Errorf("mandatory parameter not provided: %s", name)
		}
		log.Debug(model.Details)
		log.Debugf("param not found: %s, defaulting to: %d", name, deflt[0])
		return deflt[0], nil
	}
	xt := reflect.TypeOf(x).Kind()
	if xt != reflect.Int {
		return 0, fmt.Errorf("parameter is not int: %s", name)
	}
	return model.Details[name].(int), nil
}

func getBool(model Model, name string, deflt ...bool) (bool, error) {
	x := model.Details[name]
	if x == nil {
		if len(deflt) == 0 {
			return false, fmt.Errorf("mandatory parameter not provided: %s", name)
		}
		log.Debugf("param not found: %s, defaulting to: %v", name, deflt[0])
		return deflt[0], nil
	}
	xt := reflect.TypeOf(x).Kind()
	if xt != reflect.Bool {
		return false, fmt.Errorf("parameter is not bool: %s", name)
	}
	return model.Details[name].(bool), nil
}

func newLimiter(rate int) ratelimit.Limiter {
	log.Warn("rate", rate)
	if rate == 0 {
		return nil
	}
	return ratelimit.New(rate)
}

func defaultIfZero(x int, deflt int) int {
	if x == 0 {
		return deflt
	}
	return x
}

// Throttles by waiting if needed only if Limiter is set
func (c Component) throttle() {
	if c.Limiter != nil {
		c.Limiter.Take()
	}
}

func (c Component) accept(msg Message) (bool,error) {
	if c.Filter == "" {
		return true,nil
	}
	
	return true,nil
}

func (c Component) process(msg Message) (Message, error) {
	if c.Processor == "" {
		return msg, nil
	}
	return msg, nil
}
