package adapters

import (
	"encoding/json"
	"net/http"

	"github.com/dsablic/faye-go"
	"github.com/dsablic/faye-go/transport"
	"github.com/gorilla/websocket"
)

var corsHeaders = map[string]string{
	"Access-Control-Allow-Credentials": "false",
	"Access-Control-Allow-Headers":     "Accept, Authorization, Content-Type, Pragma, X-Requested-With",
	"Access-Control-Allow-Methods":     "POST, GET, PUT",
	"Access-Control-Allow-Origin":      "*",
	"Access-Control-Max-Age":           "86400",
}

func handleOptions(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == "OPTIONS" {
		for k, v := range corsHeaders {
			w.Header().Add(k, v)
		}
		return true
	}
	return false
}

func decode(r *http.Request) interface{} {
	switch r.Method {
	case "POST":
	case "GET":
	case "PUT":
	default:
		return nil
	}
	if ct := r.Header.Get("Content-Type"); ct == "application/json" {
		var v interface{}
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&v); err == nil {
			return v
		}
		return nil
	}
	r.ParseForm()
	return r.Form
}

func FayeHandler(server *faye.Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") == "websocket" {

			ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
			if _, ok := err.(websocket.HandshakeError); ok {
				http.Error(w, "Not a websocket handshake", 400)
				return
			} else if err != nil {
				server.Logger().Errorf("Websocket upgrade error: %s", err)
				return
			}

			transport.WebsocketServer(server)(ws)
		} else {
			if handleOptions(w, r) {
				return
			}
			if body := decode(r); body != nil {
				transport.MakeLongPoll(body, server, w)
			} else {
				http.Error(w, "Invalid http request", 400)
				server.Logger().Debugf("Couldn't decode request body: %v", r)
			}
		}
	})
}
