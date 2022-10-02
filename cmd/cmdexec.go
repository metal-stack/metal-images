package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
)

type cmdexec struct {
	log *zap.SugaredLogger
	c   func(ctx context.Context, name string, arg ...string) *exec.Cmd
}

type cmdParams struct {
	name     string
	args     []string
	dir      string
	timeout  time.Duration
	combined bool
}

func (i *cmdexec) command(p *cmdParams) (out string, err error) {
	var (
		start  = time.Now()
		output []byte
	)
	i.log.Infow("running command", "commmand", strings.Join(append([]string{p.name}, p.args...), " "), "start", start.String())

	ctx := context.Background()
	if p.timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}

	cmd := i.c(ctx, p.name, p.args...)
	if p.dir != "" {
		cmd.Dir = "/etc/metal"
	}

	// show stderr
	cmd.Stderr = os.Stderr

	if p.combined {
		output, err = cmd.CombinedOutput()
	} else {
		output, err = cmd.Output()
	}

	out = string(output)
	took := time.Since(start)

	if err != nil {
		i.log.Errorw("executed command with error", "output", out, "duration", took.String(), "error", err)
		return "", err
	}

	i.log.Infow("executed command", "output", out, "duration", took.String())

	return
}
