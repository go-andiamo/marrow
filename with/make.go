package with

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Make executes a make with the supplied args and targets during the Suite.Init
//
// The stage can be either Initial or Supporting (any other causes panic)
//
// It is recommended that the Supporting stage is used, as these are run as goroutines prior to
// Final stage initializers
//
// IMPORTANT NOTE: The file arg must be an absolute path (panics if otherwise)
func Make(stage Stage, file string, timeout time.Duration, showLogs bool, args ...string) With {
	if stage != Initial && stage != Supporting {
		panic("stage for Make must be Initial or Supporting")
	}
	if !filepath.IsAbs(file) {
		panic("file must be absolute")
	}
	s, err := os.Stat(file)
	if err != nil {
		panic(err)
	} else if s.IsDir() {
		panic(errors.New("make file is dir"))
	}
	return &makeWith{
		stage:    stage,
		absFile:  file,
		absPath:  filepath.Dir(file),
		timeout:  timeout,
		showLogs: showLogs,
		args:     append([]string{}, args...),
	}
}

type makeWith struct {
	stage    Stage
	absFile  string
	absPath  string
	timeout  time.Duration
	showLogs bool
	args     []string
}

var _ With = (*makeWith)(nil)

func (m *makeWith) Init(init SuiteInit) error {
	makeExe, err := resolveMakeProgram()
	if err != nil {
		return err
	}
	args := []string{"-f", m.absFile}
	args = append(args, m.args...)

	timeout := time.Minute * 2
	if m.timeout > 0 {
		timeout = m.timeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, makeExe, args...)
	cmd.Dir = m.absPath
	cmd.Env = os.Environ()
	var buf bytes.Buffer
	if m.showLogs {
		cmd.Stdout = io.MultiWriter(os.Stdout, &buf)
		cmd.Stderr = io.MultiWriter(os.Stderr, &buf)
	} else {
		cmd.Stdout = &buf
		cmd.Stderr = &buf
	}

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("make failed: %w\n%s", err, buf.String())
	}
	return nil
}

func (m *makeWith) Stage() Stage {
	return m.stage
}

func (m *makeWith) Shutdown() func() {
	return nil
}

func resolveMakeProgram() (string, error) {
	if mk := os.Getenv("MAKE"); mk != "" {
		if p, err := exec.LookPath(mk); err == nil {
			return p, nil
		}
	}
	// try common names across platforms...
	for _, c := range []string{"make", "gmake", "mingw32-make"} {
		if p, err := exec.LookPath(c); err == nil {
			return p, nil
		}
	}
	return "", errors.New("no suitable make executable found")
}
