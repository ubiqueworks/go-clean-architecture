package usecase

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/ubiqueworks/go-clean-architecture/framework/component/natsbroker"
	"github.com/ubiqueworks/go-clean-architecture/service/producer/domain"
	"github.com/ubiqueworks/go-clean-architecture/service/producer/repository"
	"github.com/ubiqueworks/go-clean-architecture/service/shared/messaging"
)

func NewStoreAndPublishMessageUseCase(broker natsbroker.Broker, repo repository.MessageRepository) *storeAndPublishMessageUseCase {
	return &storeAndPublishMessageUseCase{
		broker: broker,
		repo:   repo,
	}
}

type StoreAndPublishMessageFunc func(logger *zerolog.Logger, requestId string, msg *domain.Message) error

type storeAndPublishMessageUseCase struct {
	broker natsbroker.Broker
	repo   repository.MessageRepository
}

func (uc *storeAndPublishMessageUseCase) Execute(logger *zerolog.Logger, requestId string, msg *domain.Message) error {
	if msg == nil {
		return fmt.Errorf("message cannot be NIL")
	}

	// Store message in datastore
	if err := uc.repo.Store(msg); err != nil {
		return fmt.Errorf("error storing message: %v", err)
	}

	// Publish message event to broker
	event := &messaging.EventUserMessage{
		RequestId: requestId,
		Name:      msg.Name,
		Message:   msg.Message,
	}
	if err := uc.broker.Publish(messaging.ChannelUserMessage, event); err != nil {
		logger.Error().Err(err).Msg("error publishing event")
		// We should still return even if the event has failed
	}

	return nil
}
