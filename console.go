package main

import "net/http"

const (
	defaultUser = "admin"
	defaultPass = "admin"
)

func dashboard(w http.ResponseWriter, r *http.Request) {
	user, pass, _ := r.BasicAuth()
	if !(user == defaultUser && pass == defaultPass) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Mesh"`)
		w.WriteHeader(401)
		w.Write([]byte("401 Unauthorized\n"))
		return
	}
	w.Write([]byte("OK"))
}
