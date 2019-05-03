package main

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const (
	filter1  = "function filter() { if (headers['foo']=='foo') {return true} else { return false } }"
	process1 = "function process() { headers['foo']='foo'; }"
	process2 = "function foo() { headers['foo']='foo'; }"
)

func TestFilter(t *testing.T) {
	msg := Message{Data: []byte("abc"), Headers: map[string][]byte{"foo": []byte("foo")}}
	accepted, err := evalFilter("filter", filter1, msg)
	if err != nil {
		log.Fatal(err)
	}
	log.Debug("accepted ", accepted)
	require := require.New(t)
	require.True(accepted)
}

func TestProcess(t *testing.T) {
	msgIn := Message{Data: []byte("abc"), Headers: map[string][]byte{}}
	msgOut, err := evalProcess("process1", process1, msgIn)
	if err != nil {
		log.Fatal(err)
	}
	log.Debug("msgOut ", msgOut)
	require := require.New(t)
	require.Equal(map[string][]byte{"foo": []byte("foo")}, msgOut.Headers)

	msgOut, err = evalProcess("process2", process2, msgIn)
	require.Error(err)
}

func BenchmarkFilter(b *testing.B) {
	for n := 0; n < b.N; n++ {
		msgIn := Message{Data: []byte("abc"), Headers: map[string][]byte{}}
		evalProcess("process", process1, msgIn)
	}
}

