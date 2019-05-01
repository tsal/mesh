package main

import (
	"sync"

	"github.com/dop251/goja"
	log "github.com/sirupsen/logrus"
)

var vmPool = sync.Pool{
	New: func() interface{} {
		return goja.New()
	},
}

var jsMetrics = struct {
}{}

func eval(id string, code string, msg Message) (goja.Value, error) {
	vm := vmPool.Get().(*goja.Runtime)
	defer vmPool.Put(vm)
	vm.Set("body", string(msg.Data))
	vm.Set("headers", msg.Headers)
	program, err := goja.Compile(id, code, false)
	if err != nil {
		return nil, err
	}
	_, err = vm.RunProgram(program)
	if err != nil {
		return nil, err
	}
	m := vm.Get("headers")
	log.Warnf("%v", m.Export())
	return nil, nil
}
