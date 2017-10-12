package faye

import (
	"github.com/dsablic/faye-go/protocol"
	"github.com/dsablic/faye-go/utils"
)

type validator func(*protocol.Message, *protocol.Client) bool

type Server struct {
	engine         *Engine
	logger         utils.Logger
	subscribeValid validator
	publishValid   validator
}

func (s *Server) Logger() utils.Logger {
	return s.logger
}

func NewServer(logger utils.Logger, engine *Engine, subscribe, publish validator) *Server {
	return &Server{engine, logger, subscribe, publish}
}

func (s *Server) HandleRequest(msges interface{}, conn protocol.Connection) {
	switch msges.(type) {
	case []interface{}:
		msg_list := msges.([]interface{})
		for _, msg := range msg_list {
			s.handleMessage(msg.(map[string]interface{}), conn)
		}
	case map[string]interface{}:
		m := msges.(map[string]interface{})
		if nested, ok := m["message"]; ok {
			m = nested.(map[string]interface{})
		}
		s.handleMessage(m, conn)
	default:
		s.logger.Warnf("Invalid message %v", msges)
		s.respondWithError(conn, "Invalid message")
	}
}

func (s *Server) getClient(request protocol.Message, conn protocol.Connection) *protocol.Client {
	clientId := request.ClientId()
	return s.engine.GetClient(clientId)
}

func (s *Server) handleMessage(msg protocol.Message, conn protocol.Connection) {
	channel := msg.Channel()
	if channel.IsMeta() {
		s.handleMeta(msg, conn)
	} else {
		if s.publishValid(&msg, s.getClient(msg, conn)) {
			s.engine.Publish(msg, conn)
		} else {
			s.logger.Warnf("Invalid publish %v", msg)
			s.respondWithError(conn, "Invalid publish")
		}
	}
}

func (s *Server) handleMeta(msg protocol.Message, conn protocol.Connection) protocol.Message {
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
				if s.subscribeValid(&msg, client) {
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
			response := msg
			response["successful"] = false
			response["advice"] = protocol.DEFAULT_ADVICE
			conn.Send([]protocol.Message{response})
			conn.Close()
		}
	}

	return nil
}

func (s *Server) respondWithError(conn protocol.Connection, err string) {
	response := protocol.Message{}
	response["error"] = err
	conn.Send([]protocol.Message{response})
	conn.Close()
}
