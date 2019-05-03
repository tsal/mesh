package main

import (
	"net/http"
	"time"

	"github.com/cbroglie/mustache"
	log "github.com/sirupsen/logrus"
)

const (
	defaultUser = "admin"
	defaultPass = "admin"
)

func dashboard(mesh *Mesh, w http.ResponseWriter, r *http.Request) {
	user, pass, _ := r.BasicAuth()
	if !(user == defaultUser && pass == defaultPass) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Mesh"`)
		w.WriteHeader(401)
		w.Write([]byte("401 Unauthorized\n"))
		return
	}
	uptime := time.Duration(Now() - mesh.StartedAt)

	type node struct {
		name string
	}
	var nodes []node

	for _, c := range mesh.Consumers {
		nodes = append(nodes, node{c.id})
		for _, p := range c.Producers {
			nodes = append(nodes, node{p.id})
		}
	}
	
	log.Warnf("in dash: %v, nodes=%v", mesh, nodes)

	//getMetrics()

	data, err := mustache.Render(html, map[string]interface{}{
		"mesh":   mesh,
		"uptime": uptime,
		"nodes":  nodes})
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
	w.Write([]byte(data))
}

var html = `
<html>
<head>
<title>Mesh Dashboard</title>
<script type="text/javascript" src="https://cdnjs.cloudflare.com/ajax/libs/vis/4.21.0/vis.min.js"></script>
<link href="https://cdnjs.cloudflare.com/ajax/libs/vis/4.21.0/vis.min.css" rel="stylesheet"  type="text/css">

<style type="text/css">
    #mynetwork {
      width: 600px;
      height: 400px;
      border: 1px solid lightgray;
    }
  </style>
 </head>

  <body>

  <h2> Mesh: {{mesh.ID}}</h2>

  {{nodes}}
  		{{name}}
  {{nodes}}

  <p>
  <a href="/status">Status</a> | 
  <a href="/metrics">Metrics</a>
</p>

<div id="mynetwork"></div>

<script type="text/javascript">
  // create an array with nodes
  var nodes = new vis.DataSet([
    {id: 1, label: 'Node 1'},
    {id: 2, label: 'Node 2'},
    {id: 3, label: 'Node 3'},
    {id: 4, label: 'Node 4'},
    {id: 5, label: 'Node 5'}
  ]);

  // create an array with edges
  var edges = new vis.DataSet([
    {from: 1, to: 3},
    {from: 1, to: 2},
    {from: 2, to: 4},
    {from: 2, to: 5},
    {from: 3, to: 3}
  ]);

  // create a network
  var container = document.getElementById('mynetwork');
  var data = {
    nodes: nodes,
    edges: edges
  };
  var options = {};
  var network = new vis.Network(container, data, options);
</script>

</body>
</html>

`
