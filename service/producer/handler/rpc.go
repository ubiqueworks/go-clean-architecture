package handler

import (
	"context"

	"github.com/golang/protobuf/ptypes"
	"github.com/rs/zerolog"
	"github.com/ubiqueworks/go-clean-architecture/framework"
	"github.com/ubiqueworks/go-clean-architecture/framework/util"
	"github.com/ubiqueworks/go-clean-architecture/service/producer/domain"
	"google.golang.org/grpc"
)

func InitRpcFunc(service framework.Service, component framework.Component, server *grpc.Server) error {
	RegisterProducerRPCServer(server, &rpcServer{
		logger:  component.Logger(),
		handler: service.Handler().(*serviceHandler),
	})
	return nil
}

type rpcServer struct {
	logger  *zerolog.Logger
	handler *serviceHandler
}

func (s *rpcServer) GetMessages(context.Context, *Empty) (*GetMessagesReply, error) {
	messageEntities, err := s.handler.getMessages(s.logger)
	if err != nil {
		return nil, err
	}

	messages := make([]*Message, 0)

	for _, m := range messageEntities {
		createdAt, _ := ptypes.TimestampProto(m.CreatedAt)
		msg := &Message{
			Name:      m.Name,
			Message:   m.Message,
			CreatedAt: createdAt,
		}
		messages = append(messages, msg)
	}
	return &GetMessagesReply{
		Messages: messages,
	}, nil
}

func (s *rpcServer) PublishMessage(ctx context.Context, req *PublishMessageRequest) (*Empty, error) {
	requestId := util.NewUUID()

	message := domain.NewMessage(req.GetMessage().Name, req.GetMessage().Message)
	if err := s.handler.storeAndPublishMessage(s.logger, requestId, message); err != nil {
		s.logger.Error().Err(err).Msg("server error")
		return nil, err
	}
	return &Empty{}, nil
}
