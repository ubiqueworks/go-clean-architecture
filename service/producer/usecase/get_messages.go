package usecase

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/ubiqueworks/go-clean-architecture/service/producer/domain"
	"github.com/ubiqueworks/go-clean-architecture/service/producer/repository"
)

func NewGetMessagesUseCase(repo repository.MessageRepository) *getMessagesUseCase {
	return &getMessagesUseCase{
		repo: repo,
	}
}

type GetMessagesUseCaseFunc func(logger *zerolog.Logger) ([]domain.Message, error)

type getMessagesUseCase struct {
	repo repository.MessageRepository
}

func (uc *getMessagesUseCase) Execute(logger *zerolog.Logger) ([]domain.Message, error) {
	logger.Debug().Msg("loading messages from repository")

	messages, err := uc.repo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("error loading messages from repository: %v", err)
	}
	return messages, nil
}
