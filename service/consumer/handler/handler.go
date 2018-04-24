package handler

import (
	"sync"

	"github.com/nats-io/go-nats"
	"github.com/rs/zerolog"
	"github.com/ubiqueworks/go-clean-architecture/framework"
	"github.com/ubiqueworks/go-clean-architecture/framework/component/natsbroker"
	"github.com/ubiqueworks/go-clean-architecture/service/consumer/usecase"
	"github.com/ubiqueworks/go-clean-architecture/service/shared/messaging"
	"gopkg.in/urfave/cli.v1"
)

func ServiceHandler() framework.Component {
	return &serviceHandler{}
}

type serviceHandler struct {
	service       framework.Service
	logger        *zerolog.Logger
	handleMessage usecase.HandleMessageFunc
}

func (h *serviceHandler) ID() string {
	return framework.HandlerComponent
}

func (h *serviceHandler) DependsOn() []string {
	return []string{
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
	// Notify shutdown complete on exit
	defer wg.Done()

	broker, err := natsbroker.Get(h.service)
	if err != nil {
		errCh <- err
		return
	}

	h.handleMessage = usecase.NewHandleMessageUseCase().Execute

	go h.monitorEvents(broker, shutdownCh)
	close(startedCh)

	// Wait for shutdown
	<-shutdownCh
}

func (h *serviceHandler) monitorEvents(broker natsbroker.Broker, shutdownCh <-chan struct{}) {
	h.logger.Info().Msg("start monitoring topics")

	client := broker.Client()

	userMessageCh := make(chan *nats.Msg, 10)
	if _, err := client.ChanQueueSubscribe(messaging.ChannelUserMessage, h.service.Name(), userMessageCh); err != nil {
		h.logger.Error().Err(err).Msg("error subscribing to user message channel")
	}

	for {
		select {
		case msg := <-userMessageCh:
			go h.handleMessage(h.logger, msg)
		case <-shutdownCh:
			h.logger.Info().Msg("stopped monitoring topics")
			return
		}
	}
}
