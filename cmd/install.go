package main

import (
	"context"
	"encoding/json"
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
	"github.com/spf13/afero"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

type operatingsystem string

const (
	OSUbuntu = operatingsystem("ubuntu")
	OSDebian = operatingsystem("debian")
	OSCentos = operatingsystem("centos")
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
	case OSUbuntu:
		return OSUbuntu, nil
	case OSDebian:
		return OSDebian, nil
	case OSCentos:
		return OSCentos, nil
	default:
		return operatingsystem(""), fmt.Errorf("unsupported operating system")
	}
}

type installer struct {
	log    *zap.SugaredLogger
	fs     afero.Fs
	oss    operatingsystem
	config *api.InstallerConfig
	disk   *api.Disk
}

func main() {
	log, err := newLogger(zapcore.InfoLevel)
	if err != nil {
		panic(err)
	}

	fs := afero.NewOsFs()

	oss, err := detectOS(fs)
	if err != nil {
		log.Fatal(err)
	}

	config, err := parseInstallYAML(fs)
	if err != nil {
		log.Fatal(err)
	}
	disk, err := parseDiskJSON(fs)
	if err != nil {
		log.Fatal(err)
	}

	i := installer{
		log:    log,
		fs:     fs,
		oss:    oss,
		config: config,
		disk:   disk,
	}

	// FIXME try without
	os.Setenv("DEBCONF_NONINTERACTIVE_SEEN", "true")
	os.Setenv("DEBIAN_FRONTEND", "noninteractive")

	i.do()
}

func (i *installer) do() {
	err := i.detectFirmware()
	if err != nil {
		i.log.Fatal(err)
	}

	if !i.fileExists("/etc/metal/install.yaml") {
		i.log.Fatalf("no install.yaml found")
	}

	err = i.writeResolvConf()
	if err != nil {
		i.log.Error(err)
	}

	err = i.createMetalUser()
	if err != nil {
		i.log.Error(err)
	}
	err = i.configureNetwork()
	if err != nil {
		i.log.Error(err)
	}

	err = i.copySSHKeys()
	if err != nil {
		i.log.Error(err)
	}

	err = i.fixPermissions()
	if err != nil {
		i.log.Error(err)
	}

	err = i.processUserdata()
	if err != nil {
		i.log.Error(err)
	}

	cmdLine, err := i.buildCMDLine()
	if err != nil {
		i.log.Error(err)
	}

	err = i.writeBootInfo(cmdLine)
	if err != nil {
		i.log.Error(err)
	}

	err = i.unsetMachineID()
	if err != nil {
		i.log.Error(err)
	}
}

func (i *installer) detectFirmware() error {
	if !i.fileExists("/sys/firmware/efi") {
		return fmt.Errorf("not running efi mode")
	}
	return nil
}

func (i *installer) unsetMachineID() error {
	for _, p := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		f, err := i.fs.Create(p)
		if err != nil {
			return err
		}
		f.Close()
	}
	return nil
}

func newLogger(level zapcore.Level) (*zap.SugaredLogger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zlog, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return zlog.Sugar(), nil
}

func (i *installer) fileExists(filename string) bool {
	info, err := i.fs.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
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

func (i *installer) writeResolvConf() error {
	// Must be written here because during docker build this file is synthetic
	// FIXME enable systemd-resolved based approach again once we figured out why it does not work on the firewall
	// most probably because the resolved must be running in the internet facing vrf.
	// ln -sf /run/systemd/resolve/stub-resolv.conf /etc/resolv.conf
	content := []byte(`nameserver 8.8.8.8
nameserver 8.8.4.4
`)
	return afero.WriteFile(i.fs, "/etc/resolv.conf", content, os.ModeDir)
}

func parseInstallYAML(fs afero.Fs) (*api.InstallerConfig, error) {
	var config api.InstallerConfig
	content, err := afero.ReadFile(fs, "/etc/metal/install.yaml")
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func parseDiskJSON(fs afero.Fs) (*api.Disk, error) {
	var disk api.Disk
	content, err := afero.ReadFile(fs, "/etc/metal/disk.json")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(content, &disk)
	if err != nil {
		return nil, err
	}
	return &disk, nil
}

func (i *installer) buildCMDLine() (string, error) {
	// CMDLINE="console=${CONSOLE} root=UUID=${ROOT_UUID} init=/bin/systemd net.ifnames=0 biosdevname=0 nvme_core.io_timeout=4294967295 systemd.unified_cgroup_hierarchy=0"

	rootUUID, err := i.rootUUID()
	if err != nil {
		return "", err
	}

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

func (i *installer) rootUUID() (string, error) {
	rootUUID := ""
	for _, partition := range i.disk.Partitions {
		if partition.Label == "root" {
			rootUUID = partition.Properties["UUID"]
			break
		}
	}
	if rootUUID == "" {
		return "", fmt.Errorf("did not find root uuid")
	}
	return rootUUID, nil
}

func (i *installer) checkForMD() bool {
	_, err := exec.Command("mdadm", "--examine", "--scan").Output()
	if err != nil {
		i.log.Error(err)
		return false
	}

	return true
}

func (i *installer) findMDUUID() (mdUUID string, found bool) {
	if found := i.checkForMD(); !found {
		return "", false
	}

	blkidOut, err := exec.Command("blkid").Output()
	if err != nil {
		i.log.Error(err)
		return "", false
	}
	rootUUID, err := i.rootUUID()
	if err != nil {
		i.log.Error(err)
		return "", false
	}

	var rootDisk string
	for _, line := range strings.Split(string(blkidOut), "\n") {
		if strings.Contains(line, rootUUID) {
			rd, _, found := strings.Cut(line, ":")
			if found {
				rootDisk = rd
				break
			}
		}
	}
	if rootDisk == "" {
		i.log.Errorf("unable to detect rootdisk")
		return "", false
	}

	mdadmOut, err := exec.Command("mdadm", "--detail", "--export", rootDisk).Output()
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := exec.CommandContext(ctx, "useradd", "--create-home", "--gid", "sudo", "--shell", "/bin/bash", "metal").Output()
	if err != nil {
		return err
	}
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
	io.WriteString(stdin, i.config.Password+"\n"+i.config.Password+"\n")
	return passwdCommand.Wait()
}

func (i *installer) configureNetwork() error {
	// FIXME import networker here
	_, err := exec.Command("/etc/metal/networker/metal-networker", i.config.Role, "configure", "/etc/metal/install.yaml").Output()
	return err
}

func (i *installer) copySSHKeys() error {
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
	if ok := i.fileExists("/etc/metal/userdata"); !ok {
		i.log.Infow("no userdata present, not processing userdata", "path", "/etc/metal/userdata")
		return nil
	}

	content, err := afero.ReadFile(i.fs, "/etc/metal/userdata")
	if err != nil {
		return err
	}

	defer func() {
		out, err := exec.Command("systemctl", "preset-all").Output()
		if err != nil {
			i.log.Errorw("error when running systemctl preset-all, continuing anyway", "error", err, "output", string(out))
		}
	}()

	if isCloudInitFile(content) {
		out, err := exec.Command("cloud-init", "devel", "schema", "--config-file", "/etc/metal/userdata").Output()
		i.log.Infow("executed cloud-init userdata", "output", string(out))
		if err != nil {
			i.log.Errorw("error when running cloud-init userdata, continuing anyway", "error", err)
		}

		return nil
	}

	err = i.fs.Rename("/etc/metal/userdata", "/etc/metal/config.ign")
	if err != nil {
		return err
	}

	i.log.Infow("validating ignition config")
	out, err := exec.Command("ignition-validate", "/etc/metal/config.ign").Output()
	i.log.Infow("executed ignition config validation", "output", string(out))
	if err != nil {
		i.log.Errorw("error when validating ignition userdata, continuing anyway", "error", err)
	}

	i.log.Infow("executing ignition")
	cmd := exec.Command("ignition", "-oem", "file", "-stage", "files", "-log-to-stdout")
	cmd.Dir = "/etc/metal"
	out, err = cmd.Output()
	i.log.Infow("executed ignition config validation", "output", string(out))
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
	initrd, err := os.Readlink("/boot/initrd.img")
	if err != nil {
		return err
	}

	kern, err := os.Readlink("/boot/vmlinuz")
	if err != nil {
		return err
	}

	content, err := yaml.Marshal(api.Bootinfo{
		Initrd:       initrd,
		Cmdline:      cmdLine,
		Kernel:       kern,
		BootloaderID: i.oss.BootloaderID(),
	})
	if err != nil {
		return err
	}

	return afero.WriteFile(i.fs, "/etc/metal/boot-info.yaml", content, 0700)
}

func (i *installer) grubInstall(cmdLine string) error {
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

	if i.checkForMD() {
		out, err := exec.Command("mdadm", "--examine", "--scan").Output()
		if err != nil {
			return err
		}

		mail := "\nMAILADDR root\n"
		out = append(out, []byte(mail)...)

		err = afero.WriteFile(i.fs, "/etc/mdadm.conf", out, 0700)
		if err != nil {
			return err
		}

		err = i.fs.MkdirAll("/var/lib/initramfs-tools", os.ModeDir)
		if err != nil {
			return err
		}

		out, err = exec.Command("update-initramfs", "-u").Output()
		i.log.Infow("executed update-initramfs", "output", string(out))
		if err != nil {
			return err
		}

		out, err = exec.Command("blkid").Output()
		if err != nil {
			return err
		}

		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, `PARTLABEL="efi"`) {
				disk, _, found := strings.Cut(line, ":")
				if !found {
					return fmt.Errorf("unable to process blkid output lines")
				}

				out, err = exec.Command("efibootmgr", "-c", "-d", disk, "-p1", "-l", fmt.Sprintf(`\\EFI\\%s\\grubx64.efi`, i.oss.BootloaderID()), "-L", i.oss.BootloaderID()).Output()
				i.log.Infow("executed dpkg-reconfigure", "output", string(out))
				if err != nil {
					return err
				}
			}
		}

		grubInstallArgs = append(grubInstallArgs, "--no-nvram")
	}

	out, err := exec.Command("grub-install", grubInstallArgs...).Output()
	i.log.Infow("executed grub-install", "output", string(out))
	if err != nil {
		return err
	}

	out, err = exec.Command("update-grub2").Output()
	i.log.Infow("executed update-grub2", "output", string(out))
	if err != nil {
		return err
	}

	out, err = exec.Command("dpkg-reconfigure", "grub-efi-amd64-bin").Output()
	i.log.Infow("executed dpkg-reconfigure", "output", string(out))
	if err != nil {
		return err
	}

	return nil
}
