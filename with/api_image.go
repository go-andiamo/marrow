package with

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"os"
	"strconv"
)

func ApiImage(imageName string, tag string, port int, env map[string]any, leaveRunning bool) ImageApi {
	return &apiImage{
		imageName:    imageName,
		tag:          tag,
		port:         strconv.Itoa(port),
		env:          env,
		leaveRunning: leaveRunning,
	}
}

type ImageApi interface {
	With
	Image
	MappedPort() string
	Container() testcontainers.Container
}

type apiImage struct {
	imageName    string
	tag          string
	env          map[string]any
	port         string
	mappedPort   string
	container    testcontainers.Container
	leaveRunning bool
}

var _ ImageApi = (*apiImage)(nil)
var _ With = (*apiImage)(nil)
var _ Image = (*apiImage)(nil)

func (a *apiImage) Container() testcontainers.Container {
	return a.container
}

func (a *apiImage) Host() string {
	return "localhost"
}

func (a *apiImage) Port() string {
	return a.port
}

func (a *apiImage) MappedPort() string {
	return a.mappedPort
}

func (a *apiImage) IsDocker() bool {
	return true
}

func (a *apiImage) Username() string {
	return ""
}

func (a *apiImage) Password() string {
	return ""
}

func (a *apiImage) Init(init SuiteInit) error {
	if err := a.start(init); err != nil {
		return fmt.Errorf("with api image init error: %w", err)
	}
	port, _ := strconv.ParseInt(a.mappedPort, 10, 64)
	init.SetApiHost(a.Host(), int(port))
	return nil
}

func (a *apiImage) Stage() Stage {
	return Final
}

func (a *apiImage) Shutdown() func() {
	return func() {
		a.shutdown()
	}
}

func (a *apiImage) Name() string {
	return a.imageName + ":" + a.tag
}

const envRyukDisable = "TESTCONTAINERS_RYUK_DISABLED"

func (a *apiImage) start(init SuiteInit) (err error) {
	defer func() {
		_ = os.Setenv(envRyukDisable, "false")
		if err != nil {
			err = fmt.Errorf("start container: %w", err)
		}
	}()
	var actualEnv map[string]string
	if actualEnv, err = a.actualEnv(init); err == nil {
		ctx := context.Background()
		port := a.port
		natPort := nat.Port(port + "/tcp")
		req := testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        a.imageName + ":" + a.tag,
				ExposedPorts: []string{port},
				WaitingFor:   wait.ForListeningPort(natPort),
				Env:          actualEnv,
			},
			Started: true,
		}
		if a.leaveRunning {
			_ = os.Setenv(envRyukDisable, "true")
		}
		if a.container, err = testcontainers.GenericContainer(ctx, req); err == nil {
			var ir *container.InspectResponse
			if ir, err = a.container.Inspect(ctx); err == nil {
				if mapped, ok := ir.NetworkSettings.Ports[natPort]; ok {
					a.mappedPort = mapped[0].HostPort
				} else {
					err = fmt.Errorf("could not find port %s in container", port)
				}
			}
		}
	}
	return err
}

func (a *apiImage) actualEnv(init SuiteInit) (map[string]string, error) {
	result := make(map[string]string, len(a.env))
	for k, v := range a.env {
		if av, err := init.ResolveEnv(v); err == nil {
			result[k] = av
		} else {
			return nil, err
		}
	}
	return result, nil
}

func (a *apiImage) shutdown() {
	if a.container != nil && !a.leaveRunning {
		_ = a.container.Terminate(context.Background())
	}
}
