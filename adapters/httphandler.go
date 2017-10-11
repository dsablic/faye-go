package adapters

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/dsablic/faye-go"
	"github.com/dsablic/faye-go/transport"
)

/* HTTP handler that can be dropped into the standard http handlers list */
func FayeHandler(server *faye.Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") == "websocket" {

			ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
			if _, ok := err.(websocket.HandshakeError); ok {
				http.Error(w, "Not a websocket handshake", 400)
				return
			} else if err != nil {
				log.Println(err)
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
					log.Fatal(err)
				}
			}
		}
	})
}
