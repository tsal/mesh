package main

import (
	"fmt"
	"strings"
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

func precompile(code string) error {
	t := strings.Split(code, ";")
	code = strings.Join(t[:len(t)-2], ";")
	if !(strings.Contains(code, "process()") ||
		strings.Contains(code, "filter()")) {
		return fmt.Errorf("handler contains neither filter() nor process()")
	}
	if strings.Contains(code, "filter()") && !strings.Contains(code, "return") {
		return fmt.Errorf("handler contains filter(), but there is not return statement")
	}
	return nil
}

func getProgram(id string, code string) (*goja.Program, error) {
	programsMx.RLock()
	if p, ok := programsCache[id]; ok {
		return p, nil
	}
	programsMx.RUnlock()

	err := precompile(code)
	if err != nil {
		return nil, err
	}

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
	code += ";_result=filter();"
	log.Debug(code)
	program, err := getProgram(id, code)
	if err != nil {
		return false, errors.Wrapf(err, "js compile: |%s|", code)
	}
	_, err = vm.RunProgram(program)
	if err != nil {
		return false, errors.Wrap(err, "js run")
	}
	result := vm.Get("_result")
	log.Debugf("result: %v", result)
	return result.ToBoolean(), nil
}

func evalProcess(id string, code string, msg Message) (Message, error) {
	vm := vmPool.Get().(*goja.Runtime)
	defer vmPool.Put(vm)
	setVars(vm, msg)
	code += ";process();"
	program, err := getProgram(id, code)
	if err != nil {
		return Message{}, errors.Wrapf(err, "js compile: |%s|", code)
	}
	_, err = vm.RunProgram(program)
	if err != nil {
		return Message{}, errors.Wrap(err, "js run")
	}
	jsBody := vm.Get("body")
	jsHdrs := vm.Get("headers")
	//TODO type checks to avoid panic!!!
	data := jsBody.Export().(string)
	hdrs := jsHdrs.Export().(map[string]string)
	headers := map[string][]byte{}
	for k, v := range hdrs {
		headers[k] = []byte(v)
	}
	return Message{Data: []byte(data), Headers: headers}, nil
}
