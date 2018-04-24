package cloudstore

import (
	"context"
	"fmt"
	"sync"

	"cloud.google.com/go/datastore"
	"github.com/rs/zerolog"
	"github.com/ubiqueworks/go-clean-architecture/framework"
	"gopkg.in/urfave/cli.v1"
)

const (
	Component = "cloudstore"

	envCloudProjectId  = "CLOUD_PROJECT_ID"
	flagCloudProjectId = "cloud-project-id"
)

var cliFlags = []cli.Flag{
	cli.StringFlag{
		Name:   flagCloudProjectId,
		EnvVar: envCloudProjectId,
		Usage:  "cloud project id",
	},
}

func Create() framework.Component {
	return &cloudStore{}
}

func Get(service framework.Service) (Store, error) {
	component, err := service.Component(Component)
	if err != nil {
		return nil, err
	}
	return component.(Store), nil
}

type Store interface {
	Client() *datastore.Client
}

type cloudStore struct {
	client    *datastore.Client
	logger    *zerolog.Logger
	projectID string
}

func (s *cloudStore) Client() *datastore.Client {
	return s.client
}

func (s *cloudStore) ID() string {
	return Component
}

func (s *cloudStore) DependsOn() []string {
	return nil
}

func (s *cloudStore) Flags() []cli.Flag {
	return cliFlags
}

func (s *cloudStore) Logger() *zerolog.Logger {
	return s.logger
}

func (s *cloudStore) Configure(service framework.Service, cliCtx *cli.Context) error {
	logger := service.Logger().With().Str("component", Component).Logger()
	s.logger = &logger

	projectID := cliCtx.String(flagCloudProjectId)
	if projectID == "" {
		return fmt.Errorf("missing cloud project id")
	}
	s.projectID = projectID

	return nil
}

func (s *cloudStore) Initialize(wg *sync.WaitGroup, startedCh chan<- struct{}, shutdownCh <-chan struct{}, errCh chan<- error) {
	defer wg.Done()

	s.logger.Debug().Msg("connecting...")
	client, err := datastore.NewClient(context.Background(), s.projectID)
	if err != nil {
		errCh <- err
		return
	}
	s.client = client

	s.logger.Info().Msg("connected")
	close(startedCh)

	doneCh := make(chan struct{}, 1)
	stopFunc := func() {
		s.logger.Debug().Msg("disconnecting...")
		s.client.Close()
		close(doneCh)
	}

	<-shutdownCh
	s.logger.Debug().Msg("shutdown signal received...")
	stopFunc()

	<-doneCh
	s.logger.Info().Msg("disconnected")
}
