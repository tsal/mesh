package main

import (
	"sync"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var vmPool = sync.Pool{
	New: func() interface{} {
		return goja.New()
	},
}

func setVars(vm *goja.Runtime, msg Message) {
	vm.Set("body", string(msg.Data))
	hdrs := map[string]string{}
	for k, v := range msg.Headers {
		hdrs[k] = string(v)
	}
	vm.Set("headers", hdrs)
}

var programsCache = map[string]*goja.Program{}
var programsMx = sync.RWMutex{}

func getProgram(id string, code string) (*goja.Program, error) {
	programsMx.RLock()
	if p, ok := programsCache[id]; ok {
		return p, nil
	}
	programsMx.RUnlock()
	p, err := goja.Compile(id, code, false)
	if err != nil {
		return nil, err
	}
	programsMx.Lock()
	programsCache[id] = p
	programsMx.Unlock()
	return p, nil
}

func evalFilter(id string, code string, msg Message) (bool, error) {
	vm := vmPool.Get().(*goja.Runtime)
	defer vmPool.Put(vm)
	setVars(vm, msg)
	code += " ; _result=filter();"
	log.Warn(code)
	program, err := getProgram(id, code)
	if err != nil {
		return false, errors.Wrapf(err, "js compile: |%s|", code)
	}
	_, err = vm.RunProgram(program)
	if err != nil {
		return false, errors.Wrap(err, "js run")
	}
	result := vm.Get("_result")
	log.Warnf("result: %v", result)
	return result.ToBoolean(), nil
}

func evalProcess(id string, code string, msg Message) (Message, error) {
	vm := vmPool.Get().(*goja.Runtime)
	defer vmPool.Put(vm)
	setVars(vm, msg)
	code += " ; process();"
	program, err := getProgram(id, code)
	if err != nil {
		return Message{}, errors.Wrapf(err, "js compile: |%s|", code)
	}
	_, err = vm.RunProgram(program)
	if err != nil {
		return Message{}, errors.Wrap(err, "js run")
	}
	v1 := vm.Get("body")
	v2 := vm.Get("headers")
	//log.Warnf("v1:=%v v2=%v exp1=%v exp2=%v", v1, v2, v1.Export(), v2.Export())
	//TODO type checks to avoid panic!!!
	data := v1.Export().(string)
	hdrs := v2.Export().(map[string]string)
	headers := map[string][]byte{}
	for k, v := range hdrs {
		headers[k] = []byte(v)
	}
	return Message{Data: []byte(data), Headers: headers}, nil
}
