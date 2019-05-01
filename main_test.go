package main

import (
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestMesh(t *testing.T) {
	mesh, err := newMesh("examples/empty.yaml")
	if err != nil {
		log.Error(err)
	}
	mesh.start()
	mesh.stop()
}
