package framework

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/deckarep/golang-set"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/urfave/cli.v1"
)

const cliAppTemplate = `USAGE:
   {{.Name}} {{if .VisibleFlags}}[options]{{end}}{{if .VisibleFlags}}
OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
`

const (
	logFormatHuman = "human"
	logFormatJSON  = "json"

	flagDebugMode = "debug"
	flagLogFormat = "log-format"
	envDebugMode  = "DEBUG"
	envLogFormat  = "LOG_FORMAT"
)

var defaultFlags = []cli.Flag{
	cli.BoolFlag{
		Name:   flagDebugMode,
		EnvVar: envDebugMode,
		Usage:  "enable debug logging",
	},
	cli.StringFlag{
		Name:   flagLogFormat,
		EnvVar: envLogFormat,
		Value:  logFormatJSON,
		Usage:  "enable human readable logging",
	},
}

func Create(name, version, build string, handler Component) (Service, error) {
	service := &service{
		name:           name,
		cliFlags:       defaultFlags,
		components:     make(map[string]Component),
		componentsDeps: make(map[string]mapset.Set),
		info: &versionInfo{
			Name:    name,
			Version: version,
			Build:   build,
		},
		shutdownCh: make(chan struct{}, 1),
	}

	if handler == nil {
		return nil, fmt.Errorf("missing required service handler")
	}

	// Add handler as default component
	service.AddComponent(handler)

	cli.AppHelpTemplate = cliAppTemplate
	cli.VersionPrinter = service.versionPrinter

	return service, nil
}

type Component interface {
	ID() string
	Configure(Service, *cli.Context) error
	DependsOn() []string
	Flags() []cli.Flag
	Initialize(*sync.WaitGroup, chan<- struct{}, <-chan struct{}, chan<- error)
	Logger() *zerolog.Logger
}

type Service interface {
	AddComponent(Component, ...string) error
	Bootstrap()
	Component(string) (Component, error)
	DebugMode() bool
	Handler() Component
	Logger() *zerolog.Logger
	Name() string
	Shutdown()
}

type versionInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Build   string `json:"build"`
}

type service struct {
	name             string
	cliFlags         []cli.Flag
	components       map[string]Component
	componentsDeps   map[string]mapset.Set
	componentsLock   sync.Mutex
	debugMode        bool
	humanReadableLog bool
	info             *versionInfo
	logger           *zerolog.Logger
	shutdownLock     sync.Mutex
	shutdownCh       chan struct{}
	shutdown         bool
}

func (svc *service) AddComponent(component Component, deps ...string) error {
	svc.componentsLock.Lock()
	defer svc.componentsLock.Unlock()

	// Add component flags
	svc.cliFlags = append(svc.cliFlags, component.Flags()...)

	// Add component to registry
	if _, exists := svc.components[component.ID()]; exists {
		return fmt.Errorf("duplicate component ID: %s", component.ID())
	}
	svc.components[component.ID()] = component

	// Add component dependecies
	alldeps := make([]string, 0)
	alldeps = append(alldeps, component.DependsOn()...)
	alldeps = append(alldeps, deps...)

	depset := mapset.NewSet()
	for _, dep := range alldeps {
		depset.Add(dep)
	}
	svc.componentsDeps[component.ID()] = depset

	return nil
}

func (svc *service) Bootstrap() {
	app := cli.NewApp()
	app.Flags = svc.cliFlags
	app.Name = svc.info.Name
	app.Version = svc.info.Version
	app.Before = svc.configure
	app.Action = svc.bootstrapInternal
	app.Run(os.Args)
}

func (svc *service) Component(id string) (Component, error) {
	svc.componentsLock.Lock()
	defer svc.componentsLock.Unlock()

	component, exists := svc.components[id]
	if !exists {
		return nil, fmt.Errorf("component not found [%s]", id)
	}
	return component.(Component), nil
}

func (svc *service) DebugMode() bool {
	return svc.debugMode
}

func (svc *service) Handler() Component {
	handler, _ := svc.components[HandlerComponent]
	return handler
}

func (svc *service) Logger() *zerolog.Logger {
	return svc.logger
}

func (svc *service) Name() string {
	return svc.name
}

func (svc *service) Shutdown() {
	svc.shutdownLock.Lock()
	defer svc.shutdownLock.Unlock()

	if svc.shutdown {
		return
	}
	svc.shutdown = true
	close(svc.shutdownCh)
}

func (svc *service) bootstrapInternal(_ *cli.Context) error {
	svc.logger.Info().Msg("bootstrapping service components...")

	bootstrapSequence, err := svc.computeBootstrapSequence()
	if err != nil {
		return err
	}
	svc.logger.Info().Msgf("bootstrap sequence: %s", strings.Join(bootstrapSequence, ", "))

	errCh := make(chan error)
	shutdownCh := svc.shutdownCh

	var wg sync.WaitGroup

	for _, id := range bootstrapSequence {
		c, _ := svc.components[id]

		startedCh := make(chan struct{}, 1)
		wg.Add(1)

		svc.logger.Debug().Msgf("initializing component [%s]...", c.ID())
		go c.Initialize(&wg, startedCh, shutdownCh, errCh)

		select {
		case err := <-errCh:
			return err
		case <-startedCh:
			svc.logger.Debug().Msgf("component initialized [%s]", c.ID())
		}
	}

	svc.logger.Info().Msg("service bootstrap completed")

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-quit:
			svc.Shutdown()
			goto quit
		case err = <-errCh:
			svc.logger.Error().Err(err).Msg("caught service error")
			goto quit
		}
	}

quit:
	svc.logger.Info().Msg("waiting for shutdown to complete...")
	wg.Wait()
	svc.logger.Info().Msg("shutdown completed")

	return nil
}

func (svc *service) configure(cliCtx *cli.Context) error {
	svc.debugMode = cliCtx.Bool(flagDebugMode)
	svc.humanReadableLog = cliCtx.String(flagLogFormat) == logFormatHuman
	svc.setupLogger()

	svc.logger.Info().Msg("configuring service...")

	var configErr *multierror.Error
	for _, c := range svc.components {
		if err := c.Configure(svc, cliCtx); err != nil {
			configErr = multierror.Append(configErr, err)
		}
	}

	if err := configErr.ErrorOrNil(); err != nil {
		return cli.NewExitError(err, 1)
	}
	return nil
}

func (svc *service) computeBootstrapSequence() ([]string, error) {
	var resolved []string
	for len(svc.componentsDeps) != 0 {
		readySet := mapset.NewSet()
		for name, deps := range svc.componentsDeps {
			if deps.Cardinality() == 0 {
				readySet.Add(name)
			}
		}

		if readySet.Cardinality() == 0 {
			var circular []string
			for name := range svc.componentsDeps {
				circular = append(circular, name)
			}
			err := multierror.Append(nil, fmt.Errorf("circular dependency found: %v", circular))
			return nil, cli.NewExitError(err, 1)
		}

		for name := range readySet.Iter() {
			delete(svc.componentsDeps, name.(string))
			resolved = append(resolved, name.(string))
		}

		for name, deps := range svc.componentsDeps {
			diff := deps.Difference(readySet)
			svc.componentsDeps[name] = diff
		}
	}
	return resolved, nil
}

func (svc *service) setupLogger() {
	if svc.debugMode {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if svc.humanReadableLog {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	} else {
		zerolog.TimeFieldFormat = ""
	}

	logger := log.With().
		Str("service", svc.info.Name).
		Str("version", svc.info.Version).
		Str("build", svc.info.Build).
		Logger()
	svc.logger = &logger
}

func (svc *service) versionPrinter(_ *cli.Context) {
	versionString, _ := json.Marshal(svc.info)
	fmt.Printf("%s\n", versionString)
}
