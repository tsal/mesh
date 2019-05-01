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

func doCons(cons HttpConsumer) {
	cons.Component.throttle()
}
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
		consumer.Component.throttle()
		data, err := ioutil.ReadAll(r.Body) // 1
		headers := map[string][]byte{}
		for n, h := range r.Header {
			headers[n] = []byte(h[0])
		}
		if err != nil {
			log.Error(err)
			consumer.Component.Metrics.ErrCnt.Inc()
			http.Error(w, err.Error(), 500) // 2
		}
		consumer.Component.Metrics.BytesIn.Observe(float64(len(data)))
		go func() {
			err = consumer.Node.consume(Message{Data: data, Headers: headers})
			if err != nil {
				log.Error(err)
				consumer.Component.Metrics.ErrCnt.Inc()
			}
		}()
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

func (producer HttpProducer) produce(msg Message) error {
	producer.Component.throttle()
	accepted, err := producer.Component.accept(msg)
	if err != nil {
		return err
	}
	if accepted {
		size := float64(len(msg.Data))
		msg, err := producer.Component.process(msg)
		if err != nil {
			return err
		}
		req, err := http.NewRequest(strings.ToUpper(producer.method), producer.url, bytes.NewBuffer(msg.Data))
		for n, v := range msg.Headers {
			req.Header.Set(n, string(v))
		}
		producer.Component.Metrics.BytesIn.Observe(size)
		producer.Component.Metrics.MsgSize.Observe(size)
		if err != nil {
			producer.Component.Metrics.ErrCnt.Inc()
			return err
		}
		log.Debugf("http-produce: %s", producer.Component.ID)
		resp, err := producer.client.Do(req)
		if err != nil {
			producer.Component.Metrics.ErrCnt.Inc()
			return err
		}
		producer.Component.Metrics.BytesOut.Observe(size)
		defer resp.Body.Close()
		return nil
	}
	return nil
}

func (producer HttpProducer) start() error {
	log.Debug("http-producer-start")
	return nil
}

func (producer HttpProducer) stop() {
	log.Debug("http-producer-stop")
}
