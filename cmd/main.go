package main

import (
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/metal-stack/metal-hammer/pkg/api"
	"github.com/metal-stack/v"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

func main() {
	start := time.Now()
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})
	log := slog.New(jsonHandler)

	log.Info("running install", "version", v.V.String())

	fs := afero.OsFs{}

	oss, err := detectOS(fs)
	if err != nil {
		panic(err)
	}

	config, err := parseInstallYAML(fs)
	if err != nil {
		panic(err)
	}

	i := installer{
		log:    log.WithGroup("install-go"),
		fs:     fs,
		oss:    oss,
		config: config,
		exec: &cmdexec{
			log: log.WithGroup("cmdexec"),
			c:   exec.CommandContext,
		},
	}

	err = i.do()
	if err != nil {
		i.log.Error("installation failed", "duration", time.Since(start))
		panic(err)
	}

	i.log.Info("installation succeeded", "duration", time.Since(start))
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
