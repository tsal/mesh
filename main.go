package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/jessevdk/go-flags"

	"github.com/common-nighthawk/go-figure"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type Message struct {
	Data    []byte
	Headers map[string][]byte
}

func (msg Message) Size() int {
	size := 0
	for _, v := range msg.Headers {
		size += len(v)
	}
	return size + len(msg.Data)
}

type Service interface {
	start() error
	stop()
}

type Consumer interface {
	Service
}

type Producer interface {
	Service
	produce(msg Message) error
}

type CNode struct {
	ID        string
	Filter    string
	Processor string
	consumer  Consumer
	Producers []PNode
}

type PNode struct {
	id        string
	Filter    string
	Processor string
	producer  Producer
}

func (cnode *CNode) consume(msg Message) error {
	log.Warn("cnode: ", cnode)
	log.Warn("producers: ", cnode.Producers)
	for _, pnode := range cnode.Producers {
		go func(p Producer) {
			size := float64(msg.Size())
			metrics.MsgSize.Observe(size)
			metrics.BytesIn.Observe(size)
			err := p.produce(msg)
			if err != nil {
				log.Error(err)
				metrics.ErrCnt.Inc()
			}
			metrics.MsgCnt.Inc()
			metrics.BytesOut.Observe(size)
		}(pnode.producer)
	}
	return nil
}

func (mesh *Mesh) start() error {
	for _, pnode := range mesh.PNodeIdx {
		err := pnode.producer.start()
		if err != nil {
			return err
		}
	}
	for _, cnode := range mesh.Consumers {
		log.Warnf("!!!!!!!!!!! %v", cnode)
		err := cnode.consumer.start()
		if err != nil {
			return err
		}
	}
	go func() {
		log.Infof("mesh web console at %s \n /dashboard, /metrics, /status", mesh.server.Addr)
		if err := mesh.server.ListenAndServe(); err != nil {
			if err.Error() != "http: Server closed" {
				log.Error(err)
			}
		}
	}()
	return nil
}

func (mesh *Mesh) stop() {
	log.Infof("stopping mesh: %s", mesh.ID)
	for _, cnode := range mesh.Consumers {
		cnode.consumer.stop()
	}
	for _, pnode := range mesh.PNodeIdx {
		pnode.producer.stop()
	}
	mesh.server.Shutdown(context.Background())
}

type Model struct {
	ID        string `yaml:"id"`
	Type      string
	Details   map[string]interface{}
	Filter    string
	Processor string
	Timeout   int
	Rate      int
	Retry     int
}

type MeshModel struct {
	Version   string
	Consumers []Model
	Producers []Model
	Mesh      []Node
}

type Node struct {
	In  string
	Out []string
}

type Mesh struct {
	ID        string
	PNodeIdx  map[string]PNode
	Consumers []*CNode
	server    *http.Server
	StartedAt int64
}

func newCNode(m Model) (*CNode, error) {
	node := CNode{ID: m.ID, Producers: []PNode{}, Filter: m.Filter}
	log.Debugf("creating consumer: %s", m.ID)
	log.Debugf("\ttype: %s", m.Type)
	log.Debugf("\tfilter: %s", m.Filter)
	switch m.Type {
	case "std":
		c, err := newStdConsumer(m, &node)
		if err != nil {
			return nil, err
		}
		node.consumer = c
	case "ticker":
		c, err := newTickerConsumer(m, &node)
		if err != nil {
			return nil, err
		}
		node.consumer = c
	case "http":
		c, err := newHttpConsumer(m, &node)
		if err != nil {
			return nil, err
		}
		node.consumer = c
	case "kafka":
		c, err := newKafkaConsumer(m, &node)
		if err != nil {
			return nil, err
		}
		node.consumer = c
	case "amqp":
		c, err := newAmqpConsumer(m, &node)
		if err != nil {
			return nil, err
		}
		node.consumer = c
	case "mqtt":
		c, err := newMqttConsumer(m, &node)
		if err != nil {
			return nil, err
		}
		node.consumer = c
	case "ws":
		c, err := newWsConsumer(m, &node)
		if err != nil {
			return nil, err
		}
		node.consumer = c
	default:
		return nil, fmt.Errorf("unsupported consumer type: %s", m.Type)
	}
	return &node, nil
}

func newPNode(m Model) (*PNode, error) {
	node := PNode{id: m.ID, Filter: m.Filter}
	log.Debugf("creating producer: %s", m.ID)
	log.Debugf("\ttype: %s", m.Type)
	log.Debugf("\tfilter: %s", m.Filter)
	switch m.Type {
	case "std":
		p, err := newStdProducer(m)
		if err != nil {
			return nil, err
		}
		node.producer = p
	case "http":
		p, err := newHttpProducer(m)
		if err != nil {
			return nil, err
		}
		node.producer = p
	case "kafka":
		p, err := newKafkaProducer(m)
		if err != nil {
			return nil, err
		}
		node.producer = p
	case "amqp":
		p, err := newAmqpProducer(m)
		if err != nil {
			return nil, err
		}
		node.producer = p
	case "mqtt":
		p, err := newMqttProducer(m)
		if err != nil {
			return nil, err
		}
		node.producer = p
	case "ws":
		p, err := newWsProducer(m)
		if err != nil {
			return nil, err
		}
		node.producer = p
	default:
		return nil, fmt.Errorf("unsupported producer type: %s", m.Type)
	}
	return &node, nil
}

func findModel(id string, model MeshModel) (Model, error) {
	for _, m := range model.Consumers {
		if m.ID == id {
			return m, nil
		}
	}
	for _, m := range model.Producers {
		if m.ID == id {
			return m, nil
		}
	}
	return Model{}, fmt.Errorf("consumer %s not found", id)
}

func findCNode(id string, mesh *Mesh) (*CNode, bool) {
	for _, c := range mesh.Consumers {
		if c.ID == id {
			return c, true
		}
	}
	return nil, false
}

func findPNode(id string, mesh *Mesh) (*PNode, bool) {
	for _, c := range mesh.Consumers {
		for _, p := range c.Producers {
			if p.id == id {
				return &p, true
			}
		}
	}
	return nil, false
}

func newMesh(file string, port int) (*Mesh, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrap(err, "mesh load")
	}
	model := MeshModel{}
	err = yaml.Unmarshal([]byte(data), &model)
	if err != nil {
		return nil, errors.Wrap(err, "mesh unmarshal")
	}
	log.Debugf("model: %v", model)
	r := mux.NewRouter()
	server := http.Server{Addr: fmt.Sprintf(":%d", port), Handler: r}
	mesh := Mesh{
		ID:        NewUUID(),
		StartedAt: Now(),
		PNodeIdx:  make(map[string]PNode),
		server:    &server}
	r.PathPrefix("/metrics").Methods("GET").Handler(promhttp.Handler())
	r.PathPrefix("/status").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := map[string]interface{}{}
		status["ID"] = mesh.ID
		status["uptime"] = Now() - mesh.StartedAt
		status["consumers"] = len(mesh.Consumers)
		status["producers"] = len(mesh.PNodeIdx)
		b, err := json.Marshal(status)
		if err != nil {
			log.Error(err)
		}
		w.Write(b)
	})
	r.PathPrefix("/").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dashboard(&mesh, w, r)
	})
	//TODO extract
	for _, n := range model.Mesh {
		cnode, ok := findCNode(n.In, &mesh)
		if !ok {
			m, err := findModel(n.In, model)
			if err != nil {
				return nil, fmt.Errorf("consumer %s not found", n.In)
			}
			c, err := newCNode(m)
			if err != nil {
				return nil, fmt.Errorf("couldn't create consumer: %v", m)
			}
			cnode = c
			mesh.Consumers = append(mesh.Consumers, c)
		}
		for _, id := range n.Out {
			pnode, ok := findPNode(id, &mesh)
			if !ok {
				m, err := findModel(id, model)
				if err != nil {
					return nil, fmt.Errorf("producer %s not found", id)
				}
				p, err := newPNode(m)
				if err != nil {
					return nil, fmt.Errorf("couldn't create producer: %v, err: %v", m, err)
				}
				pnode = p
				mesh.PNodeIdx[id] = *p
			}
			cnode.Producers = append(cnode.Producers, *pnode)

			println("-1")
			println(cnode)
		}
	}
	return &mesh, nil
}

var metrics = newMetrics("") //global

func main() {
	var opts struct {
		File     string `short:"f" long:"file" description:"File with mesh definition" required:"true"`
		Port     int    `short:"p" long:"port" description:"Port for web interface" default:"8080"`
		LogLevel string `short:"l" long:"loglevel" description:"Logging level" choice:"info" choice:"debug" choice:"cat" choice:"warn" default:"info"`
	}

	_, err := flags.ParseArgs(&opts, os.Args)

	if err != nil {
		log.Error(err)
		fmt.Println("usage: mesh [flags]")
		fmt.Println("flags:")
		fmt.Println("-f --file")
		fmt.Println("-p --port")
		os.Exit(1)
	}

	switch strings.ToLower(opts.LogLevel) {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	}

	fig := figure.NewFigure("Mesh", "", true)
	fig.Print()
	fmt.Println()
	log.Infof("using mesh file: %s", opts.File)

	mesh, err := newMesh(opts.File, opts.Port)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("mesh: %v", mesh)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		for range ch {
			mesh.stop()
			os.Exit(1)
		}
	}()

	log.Infof("%d consumer(s), %d producer(s) registered", len(mesh.Consumers), len(mesh.PNodeIdx))
	mesh.start()

	select {}
}
