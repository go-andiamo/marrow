package with

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"io"
	"os"
	"strconv"
)

// PullOptions is used by PullImage to supply credentials for a private docker hub
//
// credentials are only used if the Username and Password are non-empty strings
type PullOptions struct {
	Username string
	Password string
	// ServerAddress is used to denote the private docker hub address
	//
	// if Username and Password are specified but this is left empty, it defaults to "https://index.docker.io/v1/"
	ServerAddress string
}

// RunOptions is used by PullImage to denote the image should be run once pulled
type RunOptions struct {
	Name         string
	Port         int
	Env          map[string]any
	LeaveRunning bool
}

type ImagePull interface {
	With
	Image
	Container() testcontainers.Container
}

// PullImage pulls a docker image and optional runs it
//
// the image is only run as a container is runOptions is non-nil
//
// The stage can be either Initial or Supporting (any other causes panic)
//
// It is recommended that the Supporting stage is used, as these are run as goroutines prior to
// Final stage initializers
func PullImage(stage Stage, image string, pullOptions PullOptions, runOptions *RunOptions) ImagePull {
	if stage != Initial && stage != Supporting {
		panic("stage for PullImage must be Initial or Supporting")
	}
	return &pullImage{
		stage:       stage,
		image:       image,
		pullOptions: pullOptions,
		runOptions:  runOptions,
	}
}

type pullImage struct {
	stage       Stage
	image       string
	pullOptions PullOptions
	runOptions  *RunOptions
	mappedPort  string
	container   testcontainers.Container
}

func (p *pullImage) Init(init SuiteInit) (err error) {
	if err = p.pull(); err == nil {
		if p.runOptions != nil {
			err = p.run(init)
		}
	}
	return err
}

func (p *pullImage) pull() (err error) {
	var cli *client.Client
	if cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); err == nil {
		defer func() {
			_ = cli.Close()
		}()
		po := image.PullOptions{}
		if p.pullOptions.Username != "" && p.pullOptions.Password != "" {
			sAddress := p.pullOptions.ServerAddress
			if sAddress == "" {
				sAddress = "https://index.docker.io/v1/"
			}
			auth := registry.AuthConfig{
				Username:      p.pullOptions.Username,
				Password:      p.pullOptions.Password,
				ServerAddress: sAddress,
			}
			b, _ := json.Marshal(auth)
			po.RegistryAuth = base64.URLEncoding.EncodeToString(b)
		}
		var r io.ReadCloser
		if r, err = cli.ImagePull(context.Background(), p.image, po); err == nil {
			defer func() {
				_ = r.Close()
			}()
		}
	}
	return err
}

func (p *pullImage) run(init SuiteInit) (err error) {
	defer func() {
		_ = os.Setenv(envRyukDisable, "false")
		if err != nil {
			err = fmt.Errorf("start container: %w", err)
		}
	}()
	var actualEnv map[string]string
	if actualEnv, err = p.actualEnv(init); err == nil {
		ctx := context.Background()
		port := strconv.Itoa(p.runOptions.Port)
		natPort := nat.Port(port + "/tcp")
		req := testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        p.image,
				ExposedPorts: []string{port},
				WaitingFor:   wait.ForListeningPort(natPort),
				Env:          actualEnv,
			},
			Started: true,
		}
		if p.runOptions.LeaveRunning {
			_ = os.Setenv(envRyukDisable, "true")
		}
		if p.container, err = testcontainers.GenericContainer(ctx, req); err == nil {
			var ir *container.InspectResponse
			if ir, err = p.container.Inspect(ctx); err == nil {
				if mapped, ok := ir.NetworkSettings.Ports[natPort]; ok {
					p.mappedPort = mapped[0].HostPort
					init.AddSupportingImage(p)
				} else {
					err = fmt.Errorf("could not find port %s in container", port)
				}
			}
		}
	}
	return err
}

func (p *pullImage) actualEnv(init SuiteInit) (map[string]string, error) {
	result := make(map[string]string, len(p.runOptions.Env))
	for k, v := range p.runOptions.Env {
		if av, err := init.ResolveEnv(v); err == nil {
			result[k] = av
		} else {
			return nil, err
		}
	}
	return result, nil
}

func (p *pullImage) Stage() Stage {
	return p.stage
}

func (p *pullImage) Shutdown() func() {
	return func() {
		if p.container != nil {
			_ = p.container.Terminate(context.Background())
		}
	}
}

func (p *pullImage) Name() (result string) {
	if p.runOptions != nil {
		result = p.runOptions.Name
	}
	return result
}

func (p *pullImage) Host() (result string) {
	if p.runOptions != nil {
		result = "localhost"
	}
	return result
}

func (p *pullImage) Port() (result string) {
	if p.runOptions != nil {
		result = strconv.Itoa(p.runOptions.Port)
	}
	return result
}

func (p *pullImage) MappedPort() string {
	return p.mappedPort
}

func (p *pullImage) IsDocker() bool {
	return true
}

func (p *pullImage) Username() string {
	return ""
}

func (p *pullImage) Password() string {
	return ""
}

func (p *pullImage) Container() testcontainers.Container {
	return p.container
}
