package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"go.uber.org/ratelimit"

	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
)

func getString(model Model, name string, deflt ...string) (string, error) {
	x := model.Details[name]
	if x == nil {
		if len(deflt) == 0 {
			return "", fmt.Errorf("mandatory parameter not provided: %s", name)
		}
		log.Debugf("param not found: %s, defaulting to: %s", name, deflt[0])
		return deflt[0], nil
	}
	xt := reflect.TypeOf(x).Kind()
	if xt != reflect.String {
		return "", fmt.Errorf("parameter is not string: %s", name)
	}
	return model.Details[name].(string), nil
}

func getInt(model Model, name string, deflt ...int) (int, error) {
	x := model.Details[name]
	if x == nil {
		if len(deflt) == 0 {
			return 0, fmt.Errorf("mandatory parameter not provided: %s", name)
		}
		log.Debug(model.Details)
		log.Debugf("param not found: %s, defaulting to: %d", name, deflt[0])
		return deflt[0], nil
	}
	xt := reflect.TypeOf(x).Kind()
	if xt != reflect.Int {
		return 0, fmt.Errorf("parameter is not int: %s", name)
	}
	return model.Details[name].(int), nil
}

func getBool(model Model, name string, deflt ...bool) (bool, error) {
	x := model.Details[name]
	if x == nil {
		if len(deflt) == 0 {
			return false, fmt.Errorf("mandatory parameter not provided: %s", name)
		}
		log.Debugf("param not found: %s, defaulting to: %v", name, deflt[0])
		return deflt[0], nil
	}
	xt := reflect.TypeOf(x).Kind()
	if xt != reflect.Bool {
		return false, fmt.Errorf("parameter is not bool: %s", name)
	}
	return model.Details[name].(bool), nil
}

func newLimiter(rate int) ratelimit.Limiter {
	log.Debugf("rate: %v", rate)
	if rate == 0 {
		return nil
	}
	return ratelimit.New(rate)
}

func defaultIfZero(x int, deflt int) int {
	if x == 0 {
		return deflt
	}
	return x
}

// now in millis
func Now() int64 {
	return time.Now().UnixNano() / 1e6
}

// NewUUID generates a random UUID according to RFC 4122
func NewUUID() string {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return ""
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

func getMetrics(mesh *Mesh) (string, error) {
	r, err := http.Get("http://localhost" + mesh.server.Addr + "/metrics")
	if err != nil {
		return "", err
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	var errCnt string
	var msgCnt string
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "mesh_err_count") {
			errCnt = strings.Split(line, " ")[1]
		}
		if strings.HasPrefix(line, "mesh_msg_count") {
			msgCnt = strings.Split(line, " ")[1]
		}
	}
	return fmt.Sprintf("Messages processed: %s, Errors: %s", msgCnt, errCnt), nil
}

func dump(a ...interface{}) {
	scs := spew.ConfigState{MaxDepth: 3, Indent: "\t"}
	scs.Dump(a)
}

func GetenvOr(key string, deflt string) string {
	val := os.Getenv(key)
	if val == "" {
		return deflt
	}
	return val
}
