package adapters

import (
	"encoding/json"
	"net/http"

	"github.com/dsablic/faye-go"
	"github.com/dsablic/faye-go/transport"
	"github.com/gorilla/websocket"
)

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
			if r.Method == "POST" {
				var v interface{}
				dec := json.NewDecoder(r.Body)
				if err := dec.Decode(&v); err == nil {
					transport.MakeLongPoll(v, server, w)
				} else {
					server.Logger().Errorf("%v", r)
				}
			}
		}
	})
}
