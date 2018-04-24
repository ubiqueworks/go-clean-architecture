package microhttp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/ubiqueworks/go-clean-architecture/framework"
	"github.com/ubiqueworks/go-clean-architecture/framework/util"
	"gopkg.in/urfave/cli.v1"
)

const (
	Component = "http-server"

	envHttpPort  = "HTTP_PORT"
	flagHttpPort = "http-port"

	keyRequestLogger = "__request_logger"
	keyRequestId     = "__request_id"
	headerRequestId  = "x-request-id"
)

var cliFlags = []cli.Flag{
	cli.IntFlag{
		Name:   flagHttpPort,
		EnvVar: envHttpPort,
		Value:  framework.DefaultHttpPort,
		Usage:  "http server port",
	},
}

type InitServerFunc func(framework.Service, framework.Component, *gin.Engine) error

func Create(initFunc InitServerFunc) framework.Component {
	return &httpServer{
		initFunc: initFunc,
	}
}

type httpServer struct {
	initFunc InitServerFunc
	logger   *zerolog.Logger
	port     int
	router   *gin.Engine
}

func (s *httpServer) ID() string {
	return Component
}

func (s *httpServer) DependsOn() []string {
	return nil
}

func (s *httpServer) Flags() []cli.Flag {
	return cliFlags
}

func (s *httpServer) Logger() *zerolog.Logger {
	return s.logger
}

func (s *httpServer) Configure(service framework.Service, cliCtx *cli.Context) error {
	logger := service.Logger().With().Str("component", Component).Logger()
	s.logger = &logger

	httpPort := cliCtx.Int(flagHttpPort)
	if !util.IsValidPort(httpPort) {
		return fmt.Errorf("invalid listen port: %d", httpPort)
	}
	s.port = httpPort

	if err := s.configureRouter(service); err != nil {
		return err
	}
	return nil
}

func (s *httpServer) Initialize(wg *sync.WaitGroup, startedCh chan<- struct{}, shutdownCh <-chan struct{}, errCh chan<- error) {
	defer wg.Done()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.router,
	}

	go func() {
		s.logger.Info().Msgf("http listening on %v", server.Addr)
		close(startedCh)
		server.ListenAndServe()
	}()

	doneCh := make(chan struct{}, 1)
	stopFunc := func() {
		s.logger.Debug().Msg("stopping server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		server.Shutdown(ctx)
		close(doneCh)
	}

	<-shutdownCh
	s.logger.Debug().Msg("shutdown signal received...")
	stopFunc()

	<-doneCh
	s.logger.Info().Msg("server stopped")
}

func (s *httpServer) configureRouter(service framework.Service) error {
	if s.initFunc == nil {
		return fmt.Errorf("missing init function for HTTP router")
	}

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(s.requestPreFlight(), gin.Recovery())

	s.logger.Info().Msg("configuring http router...")
	if err := s.initFunc(service, s, router); err != nil {
		return err
	}
	s.router = router

	return nil
}

func (s *httpServer) requestPreFlight() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		req := c.Request

		// Add request ID
		reqId := req.Header.Get(headerRequestId)
		if strings.TrimSpace(reqId) == "" {
			reqId = util.NewUUID()
		}
		c.Set(keyRequestId, reqId)

		// Create request logger
		clientIP := c.ClientIP()
		method := req.Method
		path := req.URL.Path

		logger := s.logger.With().
			Str("request_id", reqId).
			Str("method", method).
			Str("path", path).
			Str("client_ip", clientIP).
			Logger()

		c.Set(keyRequestLogger, &logger)

		// Process request
		c.Next()

		// Stop timer
		end := time.Now()
		latency := end.Sub(start)

		logger.Info().
			Int("status", c.Writer.Status()).
			Dur("latency", latency).
			Str("err", c.Errors.ByType(gin.ErrorTypePrivate).String()).
			Msg("done")
	}
}

func RequestId(c *gin.Context) string {
	return c.MustGet(keyRequestId).(string)
}

func RequestLogger(c *gin.Context) *zerolog.Logger {
	return c.MustGet(keyRequestLogger).(*zerolog.Logger)
}
