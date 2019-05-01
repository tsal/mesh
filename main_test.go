package main

import (
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestEmpty(t *testing.T) {
	mesh, err := newMesh("examples/empty.yaml")
	if err != nil {
		log.Error(err)
	}
	err = mesh.start()
	if err != nil {
		t.Fatal(err)
	}
	mesh.stop()
}

func TestTickerHttp(t *testing.T) {
	mesh, err := newMesh("examples/ticker-http.yaml")
	if err != nil {
		log.Error(err)
	}
	err = mesh.start()
	if err != nil {
		t.Fatal(err)
	}
	mesh.stop()
}