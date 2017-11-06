package faye

import (
	"encoding/json"
	"net/url"

	"github.com/dsablic/faye-go/protocol"
	"github.com/dsablic/faye-go/utils"
)

type Validator interface {
	SubscribeValid(*protocol.Message) bool
	PublishValid(*protocol.Message) bool
}

type Server struct {
	engine    *Engine
	logger    utils.Logger
	validator Validator
}

func (s *Server) Logger() utils.Logger {
	return s.logger
}

func NewServer(logger utils.Logger, engine *Engine, validator Validator) *Server {
	return &Server{engine, logger, validator}
}

func (s *Server) HandleRequest(msges interface{}, conn protocol.Connection) {
	switch msges.(type) {
	case []interface{}:
		for _, msg := range msges.([]interface{}) {
			var m protocol.Message = msg.(map[string]interface{})
			s.handleMessage(&m, conn)
			return
		}
	case map[string]interface{}:
		var m protocol.Message = msges.(map[string]interface{})
		if nested, ok := m["message"]; ok {
			switch nested.(type) {
			case string:
				var data interface{}
				if err := json.Unmarshal([]byte(nested.(string)), &data); err != nil {
					goto InvalidMessage
				}
				m = data.(map[string]interface{})
			case map[string]interface{}:
				m = nested.(map[string]interface{})
			default:
				goto InvalidMessage
			}
		}
		s.handleMessage(&m, conn)
		return
	case url.Values:
		var msgList []map[string]interface{}
		vals := msges.(url.Values)
		message := vals.Get("message")
		if err := json.Unmarshal([]byte(message), &msgList); err != nil {
			var single map[string]interface{}
			if err := json.Unmarshal([]byte(message), &single); err != nil {
				goto InvalidMessage
			} else {
				msgList = append(msgList, single)
			}
		}
		for _, msg := range msgList {
			msg["jsonp"] = vals.Get("jsonp")
			var m protocol.Message = msg
			s.handleMessage(&m, conn)
		}
		return
	}
InvalidMessage:
	s.logger.Debugf("Invalid message %v", msges)
	s.respondWithError(conn, "Invalid message")
}

func (s *Server) getClient(request *protocol.Message, conn protocol.Connection) *protocol.Client {
	clientId := request.ClientId()
	return s.engine.GetClient(clientId)
}

func (s *Server) handleMessage(msg *protocol.Message, conn protocol.Connection) {
	channel := msg.Channel()
	if channel.IsMeta() {
		s.handleMeta(msg, conn)
	} else {
		if s.validator.PublishValid(msg) {
			s.engine.Publish(msg, conn)
		} else {
			s.logger.Warnf("Invalid publish %v", msg)
			s.respondWithError(conn, "Invalid publish")
		}
	}
}

func (s *Server) handleMeta(msg *protocol.Message, conn protocol.Connection) protocol.Message {
	meta_channel := msg.Channel().MetaType()

	if meta_channel == protocol.MetaHandshakeChannel {
		s.engine.Handshake(msg, conn)
	} else {
		client := s.getClient(msg, conn)
		if client != nil {
			client.SetConnection(conn)

			switch meta_channel {
			case protocol.MetaHandshakeChannel:
				s.engine.Handshake(msg, conn)
			case protocol.MetaConnectChannel:
				s.engine.Connect(msg, client, conn)
			case protocol.MetaDisconnectChannel:
				s.engine.Disconnect(msg, client, conn)
			case protocol.MetaUnsubscribeChannel:
				s.engine.UnsubscribeClient(msg, client)
			case protocol.MetaSubscribeChannel:
				if s.validator.SubscribeValid(msg) {
					s.engine.SubscribeClient(msg, client)
				} else {
					s.logger.Warnf("Invalid subscription %v", msg)
					s.respondWithError(conn, "Invalid subscription")
				}
			case protocol.MetaUnknownChannel:
				s.logger.Errorf("Message with unknown meta channel received")
			}
		} else {
			s.logger.Debugf("Message %v from unknown client %v", msg.Channel(), msg.ClientId())
			response := *msg
			response["successful"] = false
			response["advice"] = map[string]interface{}{"reconnect": "handshake", "interval": 1000}
			conn.Send([]protocol.Message{response})
		}
	}

	return nil
}

func (s *Server) respondWithError(conn protocol.Connection, err string) {
	response := protocol.Message{}
	response["error"] = err
	conn.Send([]protocol.Message{response})
}
