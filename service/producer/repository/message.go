package repository

import (
	"context"

	"cloud.google.com/go/datastore"
	"github.com/rs/zerolog"
	"github.com/ubiqueworks/go-clean-architecture/framework/component/cloudstore"
	"github.com/ubiqueworks/go-clean-architecture/service/producer/domain"
)

func NewMessageRepository(logger *zerolog.Logger, cloudstore cloudstore.Store) MessageRepository {
	return &messageRepository{
		logger:     logger,
		cloudstore: cloudstore,
	}
}

type MessageRepository interface {
	GetAll() ([]domain.Message, error)
	Store(*domain.Message) error
}

type messageRepository struct {
	cloudstore cloudstore.Store
	logger     *zerolog.Logger
}

func (r *messageRepository) GetAll() ([]domain.Message, error) {
	client := r.cloudstore.Client()

	query := datastore.NewQuery(domain.MessageKind)
	var result []domain.Message
	if _, err := client.GetAll(context.Background(), query, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *messageRepository) Store(entity *domain.Message) error {
	client := r.cloudstore.Client()

	if _, err := client.Put(context.Background(), entity.ID, entity); err != nil {
		return err
	}
	return nil
}
