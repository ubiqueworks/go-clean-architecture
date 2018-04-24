package handler

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/ubiqueworks/go-clean-architecture/framework"
	"github.com/ubiqueworks/go-clean-architecture/framework/component/cloudstore"
	"github.com/ubiqueworks/go-clean-architecture/framework/component/natsbroker"
	"github.com/ubiqueworks/go-clean-architecture/service/producer/repository"
	"github.com/ubiqueworks/go-clean-architecture/service/producer/usecase"
	"gopkg.in/urfave/cli.v1"
)

func ServiceHandler() framework.Component {
	return &serviceHandler{}
}

type serviceHandler struct {
	service                framework.Service
	logger                 *zerolog.Logger
	storeAndPublishMessage usecase.StoreAndPublishMessageFunc
	getMessages            usecase.GetMessagesUseCaseFunc
}

func (h *serviceHandler) ID() string {
	return framework.HandlerComponent
}

func (h *serviceHandler) DependsOn() []string {
	return []string{
		cloudstore.Component,
		natsbroker.Component,
	}
}

func (h *serviceHandler) Flags() []cli.Flag {
	return nil
}

func (h *serviceHandler) Logger() *zerolog.Logger {
	return h.logger
}

func (h *serviceHandler) Configure(service framework.Service, cliCtx *cli.Context) error {
	h.service = service

	logger := service.Logger().With().Str("component", h.ID()).Logger()
	h.logger = &logger
	return nil
}

func (h *serviceHandler) Initialize(wg *sync.WaitGroup, startedCh chan<- struct{}, shutdownCh <-chan struct{}, errCh chan<- error) {
	defer wg.Done()

	broker, err := natsbroker.Get(h.service)
	if err != nil {
		errCh <- err
		return
	}

	datastore, err := cloudstore.Get(h.service)
	if err != nil {
		errCh <- err
		return
	}

	messageRepo := repository.NewMessageRepository(h.logger, datastore)

	h.getMessages = usecase.NewGetMessagesUseCase(messageRepo).Execute
	h.storeAndPublishMessage = usecase.NewStoreAndPublishMessageUseCase(broker, messageRepo).Execute

	close(startedCh)

	// We don't need to wait for the shutdown signal in this case. The HTTP and RPC components will do that
}
