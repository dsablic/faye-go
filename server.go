package faye

import (
	"github.com/roncohen/faye-go/protocol"
	"github.com/roncohen/faye-go/utils"
)

type Server struct {
	engine *Engine
	logger utils.Logger
}

func (s *Server) Logger() utils.Logger {
	return s.logger
}

func NewServer(logger utils.Logger, engine *Engine) *Server {
	return &Server{engine, logger}
}

func (s *Server) HandleRequest(msges interface{}, conn protocol.Connection) {
	switch msges.(type) {
	case []interface{}:
		msg_list := msges.([]interface{})
		for _, msg := range msg_list {
			s.handleMessage(msg.(map[string]interface{}), conn)
		}
	case map[string]interface{}:
		s.handleMessage(msges.(map[string]interface{}), conn)
	}
}

func (s *Server) getClient(request protocol.Message, conn protocol.Connection) *protocol.Client {
	clientId := request.ClientId()
	client := s.engine.GetClient(clientId)
	if client == nil {
		s.logger.Warnf("Message %v from unknown client %v", request.Channel(), clientId)
		response := request
		response["successful"] = false
		response["advice"] = protocol.DEFAULT_ADVICE
		conn.Send([]protocol.Message{response})
		conn.Close()
	}
	return client
}

func (s *Server) handleMessage(msg protocol.Message, conn protocol.Connection) {
	channel := msg.Channel()
	if channel.IsMeta() {
		s.handleMeta(msg, conn)
	} else {
		s.engine.Publish(msg)
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
				s.engine.SubscribeClient(msg, client)

			case protocol.META_UNKNOWN_CHANNEL:
				s.logger.Panicf("Message with unknown meta channel received")

			}
		}
	}

	return nil
}
