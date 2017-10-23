package faye

import (
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
		msg_list := msges.([]interface{})
		for _, msg := range msg_list {
			var m protocol.Message = msg.(map[string]interface{})
			s.handleMessage(&m, conn)
		}
	case map[string]interface{}:
		var m protocol.Message = msges.(map[string]interface{})
		if nested, ok := m["message"]; ok {
			m = nested.(map[string]interface{})
		}
		s.handleMessage(&m, conn)
	default:
		s.logger.Warnf("Invalid message %v", msges)
		s.respondWithError(conn, "Invalid message")
	}
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

	if meta_channel == protocol.META_HANDSHAKE_CHANNEL {
		s.engine.Handshake(msg, conn)
	} else {
		client := s.getClient(msg, conn)
		if client != nil {
			client.SetConnection(conn)

			switch meta_channel {
			case protocol.META_HANDSHAKE_CHANNEL:
				s.engine.Handshake(msg, conn)
			case protocol.META_CONNECT_CHANNEL:
				s.engine.Connect(msg, client, conn)

			case protocol.META_DISCONNECT_CHANNEL:
				s.engine.Disconnect(msg, client, conn)

			case protocol.META_SUBSCRIBE_CHANNEL:
				if s.validator.SubscribeValid(msg) {
					s.engine.SubscribeClient(msg, client)
				} else {
					s.logger.Warnf("Invalid subscription %v", msg)
					s.respondWithError(conn, "Invalid subscription")
				}
			case protocol.META_UNKNOWN_CHANNEL:
				s.logger.Errorf("Message with unknown meta channel received")
			}
		} else {
			s.logger.Warnf("Message %v from unknown client %v", msg.Channel(), msg.ClientId())
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
