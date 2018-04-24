package microrpc

import (
	"fmt"
	"net"
	"sync"

	"github.com/rs/zerolog"
	"github.com/ubiqueworks/go-clean-architecture/framework"
	"github.com/ubiqueworks/go-clean-architecture/framework/util"
	"google.golang.org/grpc"
	"gopkg.in/urfave/cli.v1"
)

const (
	Component = "rpc-server"

	envRpcPort  = "RPC_PORT"
	flagRpcPort = "rpc-port"
)

var cliFlags = []cli.Flag{
	cli.IntFlag{
		Name:   flagRpcPort,
		EnvVar: envRpcPort,
		Value:  framework.DefaultRpcPort,
		Usage:  "rpc server port",
	},
}

type InitServerFunc func(framework.Service, framework.Component, *grpc.Server) error

func Create(initFunc InitServerFunc) framework.Component {
	return &rpcServer{
		initFunc: initFunc,
	}
}

type rpcServer struct {
	initFunc   InitServerFunc
	grpcServer *grpc.Server
	logger     *zerolog.Logger
	port       int
}

func (s *rpcServer) ID() string {
	return Component
}

func (s *rpcServer) DependsOn() []string {
	return nil
}

func (s *rpcServer) Flags() []cli.Flag {
	return cliFlags
}

func (s *rpcServer) Logger() *zerolog.Logger {
	return s.logger
}

func (s *rpcServer) Configure(service framework.Service, cliCtx *cli.Context) error {
	logger := service.Logger().With().Str("component", Component).Logger()
	s.logger = &logger

	rpcPort := cliCtx.Int(flagRpcPort)
	if !util.IsValidPort(rpcPort) {
		return fmt.Errorf("invalid listen port: %d", rpcPort)
	}
	s.port = rpcPort

	if err := s.configureServer(service); err != nil {
		return err
	}
	return nil
}

func (s *rpcServer) Initialize(wg *sync.WaitGroup, startedCh chan<- struct{}, shutdownCh <-chan struct{}, errCh chan<- error) {
	defer wg.Done()

	doneCh := make(chan struct{}, 1)
	startFunc := func() {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
		if err != nil {
			errCh <- err
			return
		}

		s.logger.Info().Msgf("rpc listening on %v", listener.Addr().String())
		close(startedCh)
		s.grpcServer.Serve(listener)
	}
	stopFunc := func() {
		s.logger.Debug().Msg("stopping server...")
		s.grpcServer.GracefulStop()
		close(doneCh)
	}

	go startFunc()

	<-shutdownCh
	s.logger.Debug().Msg("shutdown signal received...")
	stopFunc()

	<-doneCh
	s.logger.Info().Msg("server stopped")
}

func (s *rpcServer) configureServer(service framework.Service) error {
	if s.initFunc == nil {
		return fmt.Errorf("missing init function for RPC server")
	}

	grpcServer := grpc.NewServer()
	s.logger.Info().Msg("configuring rpc server...")
	if err := s.initFunc(service, s, grpcServer); err != nil {
		return err
	}
	s.grpcServer = grpcServer

	return nil
}
