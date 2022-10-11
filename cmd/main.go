package main

import (
	"os/exec"
	"time"

	"github.com/metal-stack/metal-hammer/pkg/api"
	"github.com/metal-stack/v"
	"github.com/spf13/afero"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

func main() {
	start := time.Now()
	log, err := newLogger(zapcore.InfoLevel)
	if err != nil {
		panic(err)
	}
	log.Infof("running install version: %s", v.V.String())

	fs := afero.OsFs{}

	oss, err := detectOS(fs)
	if err != nil {
		log.Fatal(err)
	}

	config, err := parseInstallYAML(fs)
	if err != nil {
		log.Fatal(err)
	}

	i := installer{
		log:    log.Named("install-go"),
		fs:     fs,
		oss:    oss,
		config: config,
		exec: &cmdexec{
			log: log.Named("cmdexec"),
			c:   exec.CommandContext,
		},
	}

	err = i.do()
	if err != nil {
		i.log.Errorw("installation failed", "duration", time.Since(start))
		i.log.Fatal(err)
	}

	i.log.Infow("installation succeeded", "duration", time.Since(start))
}

func parseInstallYAML(fs afero.Fs) (*api.InstallerConfig, error) {
	var config api.InstallerConfig
	content, err := afero.ReadFile(fs, installYAML)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
