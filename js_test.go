package main

import (
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestHeader(t *testing.T) {
	msg := Message{Data: []byte("abc"), Headers: map[string][]byte{}}
	_, err := eval("p1", "headers['x']='eeee'", msg)
	if err != nil {
		log.Fatal(err)
	}
}
