package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"

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
	id        string
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

func (cnode CNode) consume(msg Message) error {
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

func (mesh Mesh) start() error {
	for _, pnode := range mesh.PNodeIdx {
		err := pnode.producer.start()
		if err != nil {
			return err
		}
	}
	for _, cnode := range mesh.Consumers {
		err := cnode.consumer.start()
		if err != nil {
			return err
		}
	}
	go func() {
		log.Info("mesh web console at 80")
		if err := mesh.server.ListenAndServe(); err != nil {
			if err.Error() != "http: Server closed" {
				log.Error(err)
			}
		}
	}()
	return nil
}

func (mesh Mesh) stop() {
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
	PNodeIdx  map[string]*PNode
	Consumers []CNode
	server    *http.Server
	StartedAt int64
}

func newCNode(m Model) (*CNode, error) {
	node := &CNode{id: m.ID, Producers: []PNode{}, Filter: m.Filter}
	log.Debugf("creating consumer: %s", m.ID)
	log.Debugf("\ttype: %s", m.Type)
	log.Debugf("\tfilter: %s", m.Filter)
	switch m.Type {
	case "std":
		c, err := newStdConsumer(m, node)
		if err != nil {
			return nil, err
		}
		node.consumer = c
	case "ticker":
		c, err := newTickerConsumer(m, node)
		if err != nil {
			return nil, err
		}
		node.consumer = c
	case "http":
		c, err := newHttpConsumer(m, node)
		if err != nil {
			return nil, err
		}
		node.consumer = c
	case "kafka":
		c, err := newKafkaConsumer(m, node)
		if err != nil {
			return nil, err
		}
		node.consumer = c
	case "amqp":
		c, err := newAmqpConsumer(m, node)
		if err != nil {
			return nil, err
		}
		node.consumer = c
	case "mqtt":
		c, err := newMqttConsumer(m, node)
		if err != nil {
			return nil, err
		}
		node.consumer = c
	case "ws":
		c, err := newWsConsumer(m, node)
		if err != nil {
			return nil, err
		}
		node.consumer = c
	default:
		return nil, fmt.Errorf("unsupported consumer type: %s", m.Type)
	}
	return node, nil
}

func newPNode(m Model) (*PNode, error) {
	node := &PNode{id: m.ID, Filter: m.Filter}
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
	return node, nil
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
		if c.id == id {
			return &c, true
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

func newMesh(file string) (*Mesh, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrap(err, "mesh load")
	}
	model := MeshModel{}
	err = yaml.Unmarshal([]byte(b), &model)
	if err != nil {
		return nil, errors.Wrap(err, "mesh unmarshal")
	}
	log.Debugf("model: %v", model)
	r := mux.NewRouter()
	server := &http.Server{Addr: ":" + "80", Handler: r}
	mesh := &Mesh{
		ID:        NewUUID(),
		StartedAt: Now(),
		PNodeIdx:  make(map[string]*PNode),
		server:    server}
	r.PathPrefix("/metrics").Methods("GET").Handler(promhttp.Handler())
	r.PathPrefix("/status").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := map[string]interface{}{}
		status["ID"] = mesh.ID
		status["uptime"] = Now() - mesh.StartedAt
		status["diag"] = fmt.Sprintf("%v", mesh)
		b, err := json.Marshal(status)
		if err != nil {
			log.Error(err)
		}
		w.Write(b)
	})
	r.PathPrefix("/").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dashboard(w, r)
	})
	for _, n := range model.Mesh {
		cnode, ok := findCNode(n.In, mesh)
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
		}
		mesh.Consumers = append(mesh.Consumers, *cnode)
		for _, id := range n.Out {
			pnode, ok := findPNode(id, mesh)
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
				mesh.PNodeIdx[id] = p
			}
			cnode.Producers = append(cnode.Producers, *pnode)
		}
	}
	return mesh, nil
}

var metrics = newMetrics("") //global

func main() {
	log.SetLevel(log.DebugLevel)
	if len(os.Args) < 2 {
		fmt.Println("Error: provide a yaml file with mesh definition")
		os.Exit(1)
	}
	file := os.Args[1]
	log.Infof("using mesh file: %s", file)

	mesh, err := newMesh(file)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("%v", mesh)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		for range ch {
			mesh.stop()
			os.Exit(1)
		}
	}()

	//panic("")
	log.Infof("%d consumer(s), %d producer(s) registered", len(mesh.Consumers), len(mesh.PNodeIdx))
	mesh.start()

	select {}
}
