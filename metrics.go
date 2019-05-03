package main

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func newSummary(name string) prometheus.Summary {
	s := prometheus.NewSummary(prometheus.SummaryOpts{
		Name: name,
	})
	prometheus.MustRegister(s)
	return s
}

func newCounter(name string) prometheus.Counter {
	c := prometheus.NewCounter(prometheus.CounterOpts{
		Name: name,
	})
	prometheus.MustRegister(c)
	return c
}

type Metrics struct {
	ID       string
	BytesIn  prometheus.Summary
	BytesOut prometheus.Summary
	MsgSize  prometheus.Summary
	ErrCnt   prometheus.Counter
	MsgCnt   prometheus.Counter
	JsTime   prometheus.Summary
}

func newMetrics(infix string) Metrics {
	prefix := "mesh_" + infix
	if !strings.HasSuffix(prefix, "_") {
		prefix = prefix + "_"
	}
	return Metrics{
		BytesIn:  newSummary(prefix + "bytes_in"),
		BytesOut: newSummary(prefix + "bytes_out"),
		MsgSize:  newSummary(prefix + "msg_size"),
		ErrCnt:   newCounter(prefix + "err_count"),
		MsgCnt:   newCounter(prefix + "msg_count"),
		JsTime:   newSummary(prefix + "js_time")}

}
