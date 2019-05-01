package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// ---- Consumer

type HttpConsumer struct {
	Component
	Node   *CNode
	server *http.Server
	port   string
	uri    string
	method string
}

func newHttpConsumer(model Model, node *CNode) (Consumer, error) {
	port, err := getInt(model, "port", 8080)
	if err != nil {
		return nil, err
	}
	uri, err := getString(model, "uri", "/")
	if err != nil {
		return nil, err
	}
	method, err := getString(model, "method", "get")
	if err != nil {
		return nil, err
	}
	rate, err := getInt(model, "rate", 0)
	if err != nil {
		return nil, err
	}
	r := mux.NewRouter()
	p := fmt.Sprintf("%d", port)
	server := &http.Server{Addr: ":" + p, Handler: r}
	consumer := &HttpConsumer{
		Component: Component{
			ID:      model.ID,
			Limiter: newLimiter(rate),
			Metrics: newMetrics(model.ID)},
		server: server,
		Node:   node,
		port:   p,
		uri:    uri,
		method: method}

	r.PathPrefix(uri).Methods(strings.ToUpper(method)).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		consumer.Component.doConsume(node, func() (Message, error) {
			data, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return Message{}, err
			}
			headers := map[string][]byte{}
			for n, h := range r.Header {
				headers[n] = []byte(h[0])
			}
			return Message{Data: data, Headers: headers}, nil
		}, func(err error) {
			http.Error(w, err.Error(), 500)
		})
	})
	return consumer, nil
}

func (consumer HttpConsumer) start() error {
	go func() {
		log.Debug("http-consumer-start")
		log.Infof("listening: port=%s, uri=%s, method=%s", consumer.port, consumer.uri, consumer.method)
		if err := consumer.server.ListenAndServe(); err != nil {
			if err.Error() != "http: Server closed" {
				log.Error(err)
			}
		}
	}()
	return nil
}

func (consumer HttpConsumer) stop() {
	log.Debug("http-consumer-stop")
	consumer.server.Shutdown(context.Background())
}

// --- Producer

type HttpProducer struct {
	Component
	client        *http.Client
	url           string
	method        string
	headers       map[string]string
	auth          []string
	tlsPublicKey  string
	tlsPrivateKey string
}

func newHttpProducer(model Model) (Producer, error) {
	url, err := getString(model, "url")
	if err != nil {
		return nil, err
	}
	method, err := getString(model, "method", "get")
	if err != nil {
		return nil, err
	}
	skipVerify, err := getBool(model, "tlsSkipVerify", false)
	if err != nil {
		return nil, err
	}
	tlsPublicKey, err := getString(model, "tlsPublicKey", "")
	if err != nil {
		return nil, err
	}
	tlsPrivateKey, err := getString(model, "tlsPrivateKey", "")
	if err != nil {
		return nil, err
	}
	auth, err := getString(model, "auth", "")
	if err != nil {
		return nil, err
	}
	headers, err := getString(model, "headers", "")
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerify},
	}
	return HttpProducer{
		Component: Component{
			ID:      model.ID,
			Limiter: newLimiter(model.Rate),
			Metrics: newMetrics(model.ID)},
		client: &http.Client{
			Timeout:   time.Duration(defaultIfZero(model.Timeout, 10000)) * time.Millisecond,
			Transport: tr},
		url:           url,
		method:        method,
		headers:       getHeaders(headers),
		auth:          getBasicAuth(auth),
		tlsPublicKey:  tlsPublicKey,
		tlsPrivateKey: tlsPrivateKey}, nil
}

func getHeaders(headers string) map[string]string {
	return map[string]string{}
}

func getBasicAuth(auth string) []string {
	return []string{}
}

func (producer HttpProducer) produce(msgIn Message) error {
	return producer.Component.doProduce(msgIn,
		func(msgOut Message) (interface{}, error) {
			req, err := http.NewRequest(strings.ToUpper(producer.method), producer.url, bytes.NewBuffer(msgOut.Data))
			for n, v := range msgOut.Headers {
				req.Header.Set(n, string(v))
			}
			return req, err
		},
		func(rq interface{}) error {
			resp, err := producer.client.Do(rq.(*http.Request))
			if err != nil {
				producer.Component.Metrics.ErrCnt.Inc()
				return err
			}
			defer resp.Body.Close()
			return nil
		})
}

func (producer HttpProducer) start() error {
	log.Debug("http-producer-start")
	return nil
}

func (producer HttpProducer) stop() {
	log.Debug("http-producer-stop")
}
