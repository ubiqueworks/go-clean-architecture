package natsbroker

import (
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/go-nats"
	"github.com/rs/zerolog"
	"github.com/ubiqueworks/go-clean-architecture/framework"
	"gopkg.in/urfave/cli.v1"
)

const (
	Component = "nats-broker"

	envNatsUrl  = "NATS_URL"
	flagNatsUrl = "nats-url"
)

var cliFlags = []cli.Flag{
	cli.StringFlag{
		Name:   flagNatsUrl,
		EnvVar: envNatsUrl,
		Usage:  "nats connection url",
	},
}

func Create(options ...nats.Option) framework.Component {
	return &natsBroker{
		natsOptions: options,
	}
}

func Get(service framework.Service) (Broker, error) {
	component, err := service.Component(Component)
	if err != nil {
		return nil, err
	}
	return component.(Broker), nil
}

type Broker interface {
	Client() *nats.Conn
	Publish(topic string, msg proto.Message) error
}

type natsBroker struct {
	client      *nats.Conn
	logger      *zerolog.Logger
	natsUrl     string
	natsOptions []nats.Option
}

func (b *natsBroker) Client() *nats.Conn {
	return b.client
}

func (b *natsBroker) Publish(subj string, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	return b.client.Publish(subj, data)
}

func (b *natsBroker) ID() string {
	return Component
}

func (b *natsBroker) DependsOn() []string {
	return nil
}

func (b *natsBroker) Flags() []cli.Flag {
	return cliFlags
}

func (b *natsBroker) Logger() *zerolog.Logger {
	return b.logger
}

func (b *natsBroker) Configure(service framework.Service, cliCtx *cli.Context) error {
	logger := service.Logger().With().Str("component", Component).Logger()
	b.logger = &logger

	natsUrl := cliCtx.String(flagNatsUrl)
	if natsUrl == "" {
		return fmt.Errorf("missing nats url")
	}
	b.natsUrl = natsUrl

	return nil
}

func (b *natsBroker) Initialize(wg *sync.WaitGroup, startedCh chan<- struct{}, shutdownCh <-chan struct{}, errCh chan<- error) {
	defer wg.Done()

	b.logger.Debug().Msg("connecting...")
	client, err := nats.Connect(b.natsUrl, b.natsOptions...)
	if err != nil {
		b.logger.Error().Err(err).Msg("connection error")
		errCh <- err
		return
	}
	b.client = client

	b.logger.Info().Msg("connected")
	close(startedCh)

	doneCh := make(chan struct{}, 1)
	stopFunc := func() {
		b.logger.Debug().Msg("disconnecting...")
		b.client.Close()
		close(doneCh)
	}

	<-shutdownCh
	b.logger.Debug().Msg("shutdown signal received...")
	stopFunc()

	<-doneCh
	b.logger.Info().Msg("disconnected")
}
