package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/metal-stack/metal-hammer/pkg/api"
	"github.com/metal-stack/metal-networker/pkg/netconf"
	"github.com/spf13/afero"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	installYAML = "/etc/metal/install.yaml"
	diskJSON    = "/etc/metal/disk.json"
	userdata    = "/etc/metal/userdata"
)

type installer struct {
	log    *zap.SugaredLogger
	fs     afero.Fs
	link   afero.LinkReader
	oss    operatingsystem
	config *api.InstallerConfig
	exec   *cmdexec
}

func (i *installer) do() error {
	err := i.detectFirmware()
	if err != nil {
		// FIXME return error, only ignored for goss tests
		i.log.Warnw("no efi detected, ignoring", "error", err)
	}

	if !i.fileExists(installYAML) {
		return fmt.Errorf("no install.yaml found")
	}

	err = i.writeResolvConf()
	if err != nil {
		// FIXME return error, only ignored for goss tests
		i.log.Warnw("writing resolv.conf failed, ignoring", "error", err)
	}

	err = i.createMetalUser()
	if err != nil {
		return err
	}
	err = i.configureNetwork()
	if err != nil {
		return err
	}

	err = i.copySSHKeys()
	if err != nil {
		return err
	}

	err = i.fixPermissions()
	if err != nil {
		return err
	}

	err = i.processUserdata()
	if err != nil {
		return err
	}

	cmdLine, err := i.buildCMDLine()
	if err != nil {
		return err
	}

	err = i.writeBootInfo(cmdLine)
	if err != nil {
		return err
	}

	err = i.grubInstall(cmdLine)
	if err != nil {
		return err
	}

	err = i.unsetMachineID()
	if err != nil {
		return err
	}
	return nil
}

func (i *installer) detectFirmware() error {
	i.log.Infow("detect firmware")
	if !i.fileExists("/sys/firmware/efi") {
		return fmt.Errorf("not running efi mode")
	}
	return nil
}

func (i *installer) unsetMachineID() error {
	i.log.Infow("unset machine-id")
	for _, p := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		f, err := i.fs.Create(p)
		if err != nil {
			return err
		}
		f.Close()
	}
	return nil
}

func (i *installer) fileExists(filename string) bool {
	info, err := i.fs.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func (i *installer) writeResolvConf() error {
	i.log.Infow("write /etc/resolv.conf")
	// Must be written here because during docker build this file is synthetic
	// FIXME enable systemd-resolved based approach again once we figured out why it does not work on the firewall
	// most probably because the resolved must be running in the internet facing vrf.
	// ln -sf /run/systemd/resolve/stub-resolv.conf /etc/resolv.conf
	content := []byte(`nameserver 8.8.8.8
nameserver 8.8.4.4
`)
	return afero.WriteFile(i.fs, "/etc/resolv.conf", content, os.ModeDir)
}

func (i *installer) buildCMDLine() (string, error) {
	i.log.Infow("build kernel cmdline")

	rootUUID := i.config.RootUUID

	parts := []string{
		fmt.Sprintf("console=%s", i.config.Console),
		fmt.Sprintf("root=UUID=%s", rootUUID),
		"init=/bin/systemd",
		"net.ifnames=0",
		"biosdevname=0",
		"nvme_core.io_timeout=4294967295",
		"systemd.unified_cgroup_hierarchy=0",
	}

	mdUUID, found := i.findMDUUID()
	if found {
		mdParts := []string{
			"rdloaddriver=raid0",
			"rdloaddriver=raid1",
			fmt.Sprintf("rd.md.uuid=%s", mdUUID),
		}
		parts = append(parts, mdParts...)
	}

	return strings.Join(parts, " "), nil
}

func (i *installer) findMDUUID() (mdUUID string, found bool) {
	i.log.Infow("detect software raid uuid")
	if !i.config.RaidEnabled {
		return "", false
	}

	blkidOut, err := i.exec.command(&cmdParams{
		name:    "blkid",
		timeout: 10 * time.Second,
	})
	if err != nil {
		i.log.Error(err)
		return "", false
	}
	rootUUID := i.config.RootUUID

	var rootDisk string
	for _, line := range strings.Split(string(blkidOut), "\n") {
		if strings.Contains(line, rootUUID) {
			rd, _, found := strings.Cut(line, ":")
			if found {
				rootDisk = strings.TrimSpace(rd)
				break
			}
		}
	}
	if rootDisk == "" {
		i.log.Errorf("unable to detect rootdisk")
		return "", false
	}

	mdadmOut, err := i.exec.command(&cmdParams{
		name:    "mdadm",
		args:    []string{"--detail", "--export", rootDisk},
		timeout: 10 * time.Second,
	})
	if err != nil {
		i.log.Error(err)
		return "", false
	}

	for _, line := range strings.Split(string(mdadmOut), "\n") {
		_, md, found := strings.Cut(line, "MD_UUID=")
		if found {
			mdUUID = md
			break
		}
	}

	if mdUUID == "" {
		i.log.Errorf("unable to detect md root disk")
		return "", false
	}

	return mdUUID, true
}

func (i *installer) createMetalUser() error {
	i.log.Infow("create user", "user", "metal")

	_, err := i.exec.command(&cmdParams{
		name:    "useradd",
		args:    []string{"--create-home", "--gid", "sudo", "--shell", "/bin/bash", "metal"},
		timeout: 10 * time.Second,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	passwdCommand := exec.CommandContext(ctx, "passwd", "metal")

	stdin, err := passwdCommand.StdinPipe()
	if err != nil {
		return err
	}
	defer stdin.Close()
	err = passwdCommand.Start()
	if err != nil {
		return err
	}
	_, err = io.WriteString(stdin, i.config.Password+"\n"+i.config.Password+"\n")
	if err != nil {
		return err
	}

	return passwdCommand.Wait()
}

func (i *installer) configureNetwork() error {
	i.log.Infow("configure network")
	kb := netconf.NewKnowledgeBase(installYAML)

	var kind netconf.BareMetalType
	switch i.config.Role {
	case "firewall":
		kind = netconf.Firewall
	case "machine":
		kind = netconf.Machine
	default:
		return fmt.Errorf("unknown role:%s", i.config.Role)
	}

	err := kb.Validate(kind)
	if err != nil {
		return err
	}

	netconf.NewConfigurator(kind, kb).Configure()
	return nil
}

func (i *installer) copySSHKeys() error {
	i.log.Infow("copy ssh keys")
	err := i.fs.MkdirAll("/home/metal/.ssh", 0700)
	if err != nil {
		return err
	}

	u, err := user.Lookup("metal")
	if err != nil {
		return err
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return err
	}

	err = i.fs.Chown("/home/metal/.ssh", uid, gid)
	if err != nil {
		return err
	}

	return afero.WriteFile(i.fs, "/home/metal/.ssh/authorized_keys", []byte(i.config.SSHPublicKey), 0600)
}

func (i *installer) fixPermissions() error {
	i.log.Infow("fix permissions")
	for p, perm := range map[string]fs.FileMode{
		"/var/tmp":   1777,
		"/etc/hosts": 0644,
	} {
		err := i.fs.Chmod(p, perm)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *installer) processUserdata() error {
	i.log.Infow("process userdata")
	if ok := i.fileExists(userdata); !ok {
		i.log.Infow("no userdata present, not processing userdata", "path", userdata)
		return nil
	}

	content, err := afero.ReadFile(i.fs, userdata)
	if err != nil {
		return err
	}

	defer func() {
		out, err := i.exec.command(&cmdParams{
			name: "systemctl",
			args: []string{"preset-all"},
		})
		if err != nil {
			i.log.Errorw("error when running systemctl preset-all, continuing anyway", "error", err, "output", string(out))
		}
	}()

	if isCloudInitFile(content) {
		_, err := i.exec.command(&cmdParams{
			name: "cloud-init",
			args: []string{"devel", "schema", "--config-file", userdata},
		})
		if err != nil {
			i.log.Errorw("error when running cloud-init userdata, continuing anyway", "error", err)
		}

		return nil
	}

	err = i.fs.Rename(userdata, "/etc/metal/config.ign")
	if err != nil {
		return err
	}

	i.log.Infow("validating ignition config")
	_, err = i.exec.command(&cmdParams{
		name: "ignition-validate",
		args: []string{"/etc/metal/config.ign"},
	})
	if err != nil {
		i.log.Errorw("error when validating ignition userdata, continuing anyway", "error", err)
	}

	i.log.Infow("executing ignition")
	_, err = i.exec.command(&cmdParams{
		name: "ignition",
		args: []string{"-oem", "file", "-stage", "files", "-log-to-stdout"},
		dir:  "/etc/metal",
	})
	if err != nil {
		i.log.Errorw("error when running ignition, continuing anyway", "error", err)
	}

	return nil
}

func isCloudInitFile(content []byte) bool {
	for i, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, "#cloud-config") {
			return true
		}
		if i > 1 {
			return false
		}
	}
	return false
}

func (i *installer) writeBootInfo(cmdLine string) error {
	i.log.Infow("write boot-info")
	var (
		initrd string
		kern   string
	)

	kernsrc := "/vmlinuz"
	initrdsrc := "/initrd.img"

	if i.fileExists("/boot/vmlinuz") {
		kernsrc = "/boot/vmlinuz"
		initrdsrc = "/boot/initrd.img"
	}

	initrd, err := i.link.ReadlinkIfPossible(initrdsrc)
	if err != nil {
		return fmt.Errorf("unable to detect link source of initrd %w", err)
	}

	kern, err = i.link.ReadlinkIfPossible(kernsrc)
	if err != nil {
		return fmt.Errorf("unable to detect link source of vmlinuz %w", err)
	}

	content, err := yaml.Marshal(api.Bootinfo{
		Initrd:       initrd,
		Cmdline:      cmdLine,
		Kernel:       kern,
		BootloaderID: i.oss.BootloaderID(),
	})
	if err != nil {
		return fmt.Errorf("unable to write boot-info.yaml %w", err)
	}

	return afero.WriteFile(i.fs, "/etc/metal/boot-info.yaml", content, 0700)
}

func (i *installer) grubInstall(cmdLine string) error {
	i.log.Infow("install grub")
	// ttyS1,115200n8
	serialPort, serialSpeed, found := strings.Cut(i.config.Console, ",")
	if !found {
		return fmt.Errorf("serial console could not be split into port and speed")
	}

	_, serialPort, found = strings.Cut(serialPort, "ttyS")
	if !found {
		return fmt.Errorf("serial port could not be split")
	}

	serialSpeed, _, found = strings.Cut(serialSpeed, "n8")
	if !found {
		return fmt.Errorf("serial speed could not be split")
	}

	defaultGrub := fmt.Sprintf(`GRUB_DEFAULT=0
GRUB_TIMEOUT=5
GRUB_DISTRIBUTOR=$(lsb_release -i -s || echo "%s")
GRUB_CMDLINE_LINUX_DEFAULT=""
GRUB_CMDLINE_LINUX="%s"
GRUB_TERMINAL=serial
GRUB_SERIAL_COMMAND="serial --speed=%s --unit=%s --word=8"`, i.oss.BootloaderID(), cmdLine, serialSpeed, serialPort)

	err := afero.WriteFile(i.fs, "/etc/default/grub", []byte(defaultGrub), 0755)
	if err != nil {
		return err
	}

	grubInstallArgs := []string{
		"--target=x86_64-efi",
		"--efi-directory=/boot/efi",
		"--boot-directory=/boot",
		"--bootloader-id=" + i.oss.BootloaderID(),
	}

	if i.config.RaidEnabled {
		out, err := i.exec.command(&cmdParams{
			name:    "mdadm",
			args:    []string{"--examine", "--scan"},
			timeout: 10 * time.Second,
		})
		if err != nil {
			return err
		}

		out += "\nMAILADDR root\n"

		err = afero.WriteFile(i.fs, "/etc/mdadm.conf", []byte(out), 0700)
		if err != nil {
			return err
		}

		err = i.fs.MkdirAll("/var/lib/initramfs-tools", os.ModeDir)
		if err != nil {
			return err
		}

		_, err = i.exec.command(&cmdParams{
			name: "update-initramfs",
			args: []string{"-u"},
		})
		if err != nil {
			return err
		}

		out, err = i.exec.command(&cmdParams{
			name: "blkid",
		})
		if err != nil {
			return err
		}

		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, `PARTLABEL="efi"`) {
				disk, _, found := strings.Cut(line, ":")
				if !found {
					return fmt.Errorf("unable to process blkid output lines")
				}

				_, err = i.exec.command(&cmdParams{
					name: "efibootmgr",
					args: []string{"-c", "-d", disk, "-p1", "-l", fmt.Sprintf(`\\EFI\\%s\\grubx64.efi`, i.oss.BootloaderID()), "-L", i.oss.BootloaderID()},
				})
				if err != nil {
					return err
				}
			}
		}

		grubInstallArgs = append(grubInstallArgs, "--no-nvram")
	}

	_, err = i.exec.command(&cmdParams{
		name: "grub-install",
		args: grubInstallArgs,
	})
	if err != nil {
		return err
	}

	_, err = i.exec.command(&cmdParams{
		name: "update-grub2",
	})
	if err != nil {
		return err
	}

	_, err = i.exec.command(&cmdParams{
		name: "dpkg-reconfigure",
		args: []string{"grub-efi-amd64-bin"},
	})
	if err != nil {
		return err
	}

	return nil
}
