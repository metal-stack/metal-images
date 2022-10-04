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

func (o operatingsystem) SudoGroup() string {
	switch o {
	case osCentos:
		return "wheel"
	case osDebian, osUbuntu:
		return "sudo"
	default:
		return "sudo"
	}
}

func (o operatingsystem) Initramdisk() string {
	switch o {
	case osCentos:
		return "initramfs.img"
	case osDebian, osUbuntu:
		return "initrd.img"
	default:
		return "initrd.img"
	}
}

func (o operatingsystem) GrubInstallCmd() string {
	switch o {
	case osCentos:
		return "grub2-install"
	case osDebian, osUbuntu:
		return "grub-install"
	default:
		return "grub-install"
	}
}

func (o operatingsystem) SupportsCloudInit() bool {
	switch o {
	case osCentos:
		return false
	case osDebian, osUbuntu:
		return true
	default:
		return false
	}
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
