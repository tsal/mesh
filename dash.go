package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/cbroglie/mustache"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	defaultUser = "admin"
	defaultPass = "admin"
)

type node struct {
	Name string
	Desc string
}

type edge struct {
	Start string
	End   string
}

func getDesc(model Model) string {
	//fmt.Sprintf("%v", cm.Details)
	b, _ := yaml.Marshal(model.Details)
	return strings.ReplaceAll(string(b), "\n", "\\n")
}

func dashboard(mesh *Mesh, model MeshModel, w http.ResponseWriter, r *http.Request) {
	user, pass, _ := r.BasicAuth()
	if !(user == GetenvOr("MESH_USER", defaultUser) && pass == GetenvOr("MESH_PASSWORD", defaultPass)) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Mesh"`)
		w.WriteHeader(401)
		w.Write([]byte("401 Unauthorized\n"))
		return
	}
	uptime := time.Duration(Now() - mesh.StartedAt)

	var nodes []node
	var edges []edge

	for _, c := range mesh.Consumers {
		cm, _ := findModel(c.ID, model)
		nodes = append(nodes, node{c.ID, getDesc(cm)})
		for _, p := range c.Producers {
			pm, _ := findModel(p.ID, model)
			nodes = append(nodes, node{p.ID, getDesc(pm)})
			edges = append(edges, edge{c.ID, p.ID})
		}
	}

	report, err := getMetrics(mesh)
	if err != nil {
		log.Error(err)
	}

	data, err := mustache.Render(html, map[string]interface{}{
		"mesh":   mesh,
		"uptime": uptime,
		"nodes":  nodes,
		"edges":  edges,
		"report": report})

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

  <p>
  <a href="/status">Status</a> | 
  <a href="/metrics">Metrics</a>
</p>

<div id="mynetwork"></div>

{{report}}

<script type="text/javascript">
  // create an array with nodes
  var nodes = new vis.DataSet([

  {{#nodes}}
		{id: '{{Name}}', label: 'ID:{{Name}} \n {{Desc}}', font: { multi: 'html', size: 10 }},
  {{/nodes}}

  ]);

  // create an array with edges
  var edges = new vis.DataSet([
	
	{{#edges}}
	{from: '{{Start}}', to: '{{End}}', arrows:'to'},
	{{/edges}}
	
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
