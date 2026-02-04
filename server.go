package faye

import (
	"encoding/json"
	"fmt"
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
	if err := s.handleRequestInternal(msges, conn); err != nil {
		s.logger.Debugf("Invalid message %v: %v", msges, err)
		s.respondWithError(conn, "Invalid message")
	}
}

func (s *Server) handleRequestInternal(msges interface{}, conn protocol.Connection) error {
	switch v := msges.(type) {
	case []interface{}:
		for _, msg := range v {
			m, ok := msg.(map[string]interface{})
			if !ok {
				return fmt.Errorf("message is not a map: %T", msg)
			}
			var pm protocol.Message = m
			s.handleMessage(&pm, conn)
			return nil
		}
	case map[string]interface{}:
		var m protocol.Message = v
		if nested, ok := m["message"]; ok {
			switch nestedVal := nested.(type) {
			case string:
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(nestedVal), &data); err != nil {
					return fmt.Errorf("failed to unmarshal nested message: %w", err)
				}
				m = data
			case map[string]interface{}:
				m = nestedVal
			default:
				return fmt.Errorf("unexpected nested message type: %T", nested)
			}
		}
		s.handleMessage(&m, conn)
		return nil
	case url.Values:
		var msgList []map[string]interface{}
		message := v.Get("message")
		if err := json.Unmarshal([]byte(message), &msgList); err != nil {
			var single map[string]interface{}
			if err := json.Unmarshal([]byte(message), &single); err != nil {
				return fmt.Errorf("failed to unmarshal message: %w", err)
			}
			msgList = append(msgList, single)
		}
		for _, msg := range msgList {
			msg["jsonp"] = v.Get("jsonp")
			var m protocol.Message = msg
			s.handleMessage(&m, conn)
		}
		return nil
	}
	return fmt.Errorf("unexpected message type: %T", msges)
}

func (s *Server) getClient(request *protocol.Message, conn protocol.Connection) *protocol.Client {
	return s.engine.GetClient(request.ClientId())
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

func (s *Server) handleMeta(msg *protocol.Message, conn protocol.Connection) {
	metaChannel := msg.Channel().MetaType()

	if metaChannel == protocol.MetaHandshakeChannel {
		s.engine.Handshake(msg, conn)
		return
	}

	client := s.getClient(msg, conn)
	if client == nil {
		s.logger.Debugf("Message %v from unknown client %v", msg.Channel(), msg.ClientId())
		response := *msg
		response["successful"] = false
		response["advice"] = map[string]interface{}{"reconnect": "handshake", "interval": 1000}
		conn.Send([]protocol.Message{response})
		return
	}

	client.SetConnection(conn)

	switch metaChannel {
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
}

func (s *Server) respondWithError(conn protocol.Connection, err string) {
	response := protocol.Message{}
	response["error"] = err
	conn.Send([]protocol.Message{response})
}
