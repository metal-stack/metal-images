package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/afero"
)

type operatingsystem string

const (
	osUbuntu = operatingsystem("ubuntu")
	osDebian = operatingsystem("debian")
	osCentos = operatingsystem("centos")
)

func (o operatingsystem) BootloaderID() string {
	return fmt.Sprintf("metal-%s", o)
}

func operatingSystemFromString(s string) (operatingsystem, error) {
	unquoted, err := strconv.Unquote(s)
	if err == nil {
		s = unquoted
	}

	switch operatingsystem(strings.ToLower(s)) {
	case osUbuntu:
		return osUbuntu, nil
	case osDebian:
		return osDebian, nil
	case osCentos:
		return osCentos, nil
	default:
		return operatingsystem(""), fmt.Errorf("unsupported operating system")
	}
}

func detectOS(fs afero.Fs) (operatingsystem, error) {
	content, err := afero.ReadFile(fs, "/etc/os-release")
	if err != nil {
		return operatingsystem(""), err
	}

	env := map[string]string{}
	for _, line := range strings.Split(string(content), "\n") {
		k, v, found := strings.Cut(line, "=")
		if found {
			env[k] = v
		}
	}

	if os, ok := env["ID"]; ok {
		oss, err := operatingSystemFromString(os)
		if err != nil {
			return operatingsystem(""), err
		}
		return oss, nil
	}

	return operatingsystem(""), fmt.Errorf("unable to detect OS")
}
