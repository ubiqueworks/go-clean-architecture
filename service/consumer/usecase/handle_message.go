package usecase

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/go-nats"
	"github.com/rs/zerolog"
	"github.com/ubiqueworks/go-clean-architecture/service/shared/messaging"
)

func NewHandleMessageUseCase() *handleMessageUseCase {
	return &handleMessageUseCase{}
}

type HandleMessageFunc func(logger *zerolog.Logger, rawMsg *nats.Msg) error

type handleMessageUseCase struct {
}

func (uc *handleMessageUseCase) Execute(logger *zerolog.Logger, rawMsg *nats.Msg) error {
	if rawMsg == nil {
		return fmt.Errorf("the message cannot be NIL")
	}

	var message messaging.EventUserMessage
	if err := proto.Unmarshal(rawMsg.Data, &message); err != nil {
		return fmt.Errorf("error deserializing user message event")
	}

	logger.Info().Msgf("[%s] says: %s", message.GetName(), message.GetMessage())
	return nil
}
