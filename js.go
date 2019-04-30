package main

import (
	"sync"

	"github.com/dop251/goja"
)

var vmPool = sync.Pool{
	New: func() interface{} {
		return goja.New()
	},
}

func eval(id string, code string) (goja.Value, error) {
	vm := vmPool.Get().(*goja.Runtime)
	defer vmPool.Put(vm)
	program, err := goja.Compile("scsd", code, false)
	if err != nil {
		return nil, err
	}
	v, err := vm.RunProgram(program)
	if err != nil {
		panic(err)
	}
	if num := v.Export().(int64); num != 4 {
		panic(num)
	}
	return nil, nil
}

func accept(msg Message) bool {
	return true
}

func process(msg Message) (Message, error) {
	return msg, nil
}
