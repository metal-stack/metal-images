package main

import (
	"fmt"
	"io/fs"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-hammer/pkg/api"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"gopkg.in/yaml.v3"
)

const (
	sampleInstallYAML = `---
hostname: test-machine
networks:
-   asn: 4210000000
    destinationprefixes: []
    ips:
    - 192.168.0.1
    nat: false
    networkid: 931b1568-9f2b-4b83-8bcb-cfc8f2a99e85
    networktype: privateprimaryshared
    prefixes:
    - 192.168.0.0/24
    private: true
    underlay: false
    vrf: 1
-   asn: 4210000000
    destinationprefixes:
    - 0.0.0.0/0
    ips:
    - 192.168.1.1
    nat: true
    networkid: internet
    networktype: external
    prefixes:
    - 192.168.1.0/24
    private: false
    underlay: false
    vrf: 104009
machineuuid: c647818b-0573-45a1-bac4-e311db1df753
sshpublickey: ssh-ed25519 key
password: a-password
devmode: false
console: ttyS1,115200n8
raidenabled: false
root_uuid: "543eb7f8-98d4-d986-e669-824dbebe69e5"
timestamp: "2022-02-24T14:54:58Z"
nics:
-   mac: b4:96:91:cb:64:e0
    name: eth4
    neighbors:
    -   mac: b8:6a:97:73:f8:5f
        name: null
        neighbors: []
-   mac: b4:96:91:cb:64:e1
    name: eth5
    neighbors:
    -   mac: b8:6a:97:74:00:5f
        name: null
        neighbors: []`
	sampleInstallWithRaidYAML = `---
hostname: test-machine
networks:
-   asn: 4210000000
    destinationprefixes: []
    ips:
    - 192.168.0.1
    nat: false
    networkid: 931b1568-9f2b-4b83-8bcb-cfc8f2a99e85
    networktype: privateprimaryshared
    prefixes:
    - 192.168.0.0/24
    private: true
    underlay: false
    vrf: 1
-   asn: 4210000000
    destinationprefixes:
    - 0.0.0.0/0
    ips:
    - 192.168.1.1
    nat: true
    networkid: internet
    networktype: external
    prefixes:
    - 192.168.1.0/24
    private: false
    underlay: false
    vrf: 104009
machineuuid: c647818b-0573-45a1-bac4-e311db1df753
sshpublickey: ssh-ed25519 key
password: a-password
devmode: false
console: ttyS1,115200n8
raidenabled: true
root_uuid: "ace079b5-06be-4429-bbf0-081ea4d7d0d9"
timestamp: "2022-02-24T14:54:58Z"
nics:
-   mac: b4:96:91:cb:64:e0
    name: eth4
    neighbors:
    -   mac: b8:6a:97:73:f8:5f
        name: null
        neighbors: []
-   mac: b4:96:91:cb:64:e1
    name: eth5
    neighbors:
    -   mac: b8:6a:97:74:00:5f
        name: null
        neighbors: []`
	sampleBlkidOutput = `/dev/sda1: UUID="42d10089-ee1e-0399-445e-755062b63ec8" UUID_SUB="cc57c456-0b2f-6345-c597-d861cc6dd8ac" LABEL="any:0" TYPE="linux_raid_member" PARTLABEL="efi" PARTUUID="273985c8-d097-4123-bcd0-80b4e4e14728"
/dev/sda2: UUID="543eb7f8-98d4-d986-e669-824dbebe69e5" UUID_SUB="54748c60-b566-f391-142c-fb78bb1fc6a9" LABEL="any:1" TYPE="linux_raid_member" PARTLABEL="root" PARTUUID="d7863f4e-af7c-47fc-8c03-6ecdc69bc72d"
/dev/sda3: UUID="fc32a6f0-ee40-d9db-87c8-c9f3a8400c8b" UUID_SUB="582e9b4f-f191-e01e-85fd-2f7d969fbef6" LABEL="any:2" TYPE="linux_raid_member" PARTLABEL="varlib" PARTUUID="e8b44f09-b7f7-4e0d-a7c3-d909617d1f05"
/dev/sdb1: UUID="42d10089-ee1e-0399-445e-755062b63ec8" UUID_SUB="61bd5d8b-1bb8-673b-9e61-8c28dccc3812" LABEL="any:0" TYPE="linux_raid_member" PARTLABEL="efi" PARTUUID="13a4c568-57b0-4259-9927-9ac023aaa5f0"
/dev/sdb2: UUID="543eb7f8-98d4-d986-e669-824dbebe69e5" UUID_SUB="e7d01e93-9340-5b90-68f8-d8f815595132" LABEL="any:1" TYPE="linux_raid_member" PARTLABEL="root" PARTUUID="ab11cd86-37b8-4bae-81e5-21fe0a9c9ae0"
/dev/sdb3: UUID="fc32a6f0-ee40-d9db-87c8-c9f3a8400c8b" UUID_SUB="764217ad-1591-a83a-c799-23397f968729" LABEL="any:2" TYPE="linux_raid_member" PARTLABEL="varlib" PARTUUID="9afbf9c1-b2ba-4b46-8db1-e802d26c93b6"
/dev/md1: LABEL="root" UUID="ace079b5-06be-4429-bbf0-081ea4d7d0d9" TYPE="ext4"
/dev/md0: LABEL="efi" UUID="C236-297F" TYPE="vfat"
/dev/md2: LABEL="varlib" UUID="385e8e8e-dbfd-481e-93a4-cba7f4d5fa02" TYPE="ext4"`
	sampleMdadmDetailOutput = `MD_LEVEL=raid1
MD_DEVICES=2
MD_METADATA=1.0
MD_UUID=543eb7f8:98d4d986:e669824d:bebe69e5
MD_DEVNAME=1
MD_NAME=any:1
MD_DEVICE_dev_sdb2_ROLE=1
MD_DEVICE_dev_sdb2_DEV=/dev/sdb2
MD_DEVICE_dev_sda2_ROLE=0
MD_DEVICE_dev_sda2_DEV=/dev/sda2`
	sampleHostnamectlPhysical = `Static hostname: badile
Icon name: computer-laptop
  Chassis: laptop
Machine ID: 2ded0d71f52d4a73aeada73b65feda60
  Boot ID: 1930639060b743a7907c7906a23b49d9
Operating System: Ubuntu 22.04.1 LTS
   Kernel: Linux 5.15.0-48-generic
Architecture: x86-64
Hardware Vendor: Lenovo
Hardware Model: ThinkPad X1 Carbon Gen 10`
	sampleHostnamectlVirtual = `Static hostname: badile
Icon name: computer-laptop
  Chassis: laptop
Machine ID: 2ded0d71f52d4a73aeada73b65feda60
  Boot ID: 1930639060b743a7907c7906a23b49d9
Operating System: Ubuntu 22.04.1 LTS
   Kernel: Linux 5.15.0-48-generic
   Virtualization: kvm
Architecture: x86-64
Hardware Vendor: Lenovo
Hardware Model: ThinkPad X1 Carbon Gen 10`
)

type linkMock struct {
	mocks map[string]string
}

func (l *linkMock) ReadlinkIfPossible(name string) (string, error) {
	v, ok := l.mocks[name]
	if !ok {
		return "", fmt.Errorf("no mock for %s", name)
	}
	return v, nil
}

func mustParseInstallYAML(t *testing.T, fs afero.Fs) *api.InstallerConfig {
	config, err := parseInstallYAML(fs)
	require.NoError(t, err)
	return config
}

func Test_installer_detectFirmware(t *testing.T) {
	tests := []struct {
		name      string
		fsMocks   func(fs afero.Fs)
		execMocks []fakeexecparams
		wantErr   error
	}{
		{
			name: "is efi",
			fsMocks: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, "/sys/firmware/efi", []byte(""), 0755))
			},
			execMocks: []fakeexecparams{
				{
					WantCmd:  []string{"hostnamectl", "status"},
					Output:   sampleHostnamectlPhysical,
					ExitCode: 0,
				},
			},
			wantErr: nil,
		},
		{
			name: "is not efi",
			execMocks: []fakeexecparams{
				{
					WantCmd:  []string{"hostnamectl", "status"},
					Output:   sampleHostnamectlPhysical,
					ExitCode: 0,
				},
			},
			wantErr: fmt.Errorf("not running efi mode"),
		},
		{
			name: "is not efi but virtual",
			execMocks: []fakeexecparams{
				{
					WantCmd:  []string{"hostnamectl", "status"},
					Output:   sampleHostnamectlVirtual,
					ExitCode: 0,
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			log := zaptest.NewLogger(t).Sugar()

			i := &installer{
				log: log,
				fs:  afero.NewMemMapFs(),
				exec: &cmdexec{
					log: log,
					c:   fakeCmd(t, tt.execMocks...),
				},
			}

			if tt.fsMocks != nil {
				tt.fsMocks(i.fs)
			}

			err := i.detectFirmware()
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}
		})
	}
}

func Test_installer_writeResolvConf(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr error
	}{
		{
			name: "resolv.conf gets written",
			want: `nameserver 8.8.8.8
nameserver 8.8.4.4
`,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			i := &installer{
				log: zaptest.NewLogger(t).Sugar(),
				fs:  afero.NewMemMapFs(),
			}

			err := i.writeResolvConf()
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}

			content, err := afero.ReadFile(i.fs, "/etc/resolv.conf")
			require.NoError(t, err)

			if diff := cmp.Diff(tt.want, string(content)); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}
		})
	}
}

func Test_installer_fixPermissions(t *testing.T) {
	tests := []struct {
		name    string
		fsMocks func(fs afero.Fs)
		wantErr error
	}{
		{
			name: "fix permissions",
			fsMocks: func(fs afero.Fs) {
				require.NoError(t, fs.MkdirAll("/var/tmp", 0000))
				require.NoError(t, afero.WriteFile(fs, "/etc/hosts", []byte("127.0.0.1"), 0000))
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			i := &installer{
				log: zaptest.NewLogger(t).Sugar(),
				fs:  afero.NewMemMapFs(),
			}

			if tt.fsMocks != nil {
				tt.fsMocks(i.fs)
			}

			err := i.fixPermissions()
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}

			info, err := i.fs.Stat("/var/tmp")
			require.NoError(t, err)
			assert.Equal(t, fs.FileMode(1777).Perm(), info.Mode().Perm())

			info, err = i.fs.Stat("/etc/hosts")
			require.NoError(t, err)
			assert.Equal(t, fs.FileMode(0644).Perm(), info.Mode().Perm())
		})
	}
}

func Test_installer_findMDUUID(t *testing.T) {
	tests := []struct {
		name      string
		fsMocks   func(fs afero.Fs)
		execMocks []fakeexecparams
		want      string
		wantFound bool
	}{
		{
			name: "has mdadm",
			execMocks: []fakeexecparams{
				{
					WantCmd:  []string{"blkid"},
					Output:   sampleBlkidOutput,
					ExitCode: 0,
				},
				{
					WantCmd:  []string{"mdadm", "--detail", "--export", "/dev/md1"},
					Output:   sampleMdadmDetailOutput,
					ExitCode: 0,
				},
			},
			want:      "543eb7f8:98d4d986:e669824d:bebe69e5",
			wantFound: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.fsMocks != nil {
				tt.fsMocks(fs)
			}

			log := zaptest.NewLogger(t).Sugar()

			i := &installer{
				log: log,
				exec: &cmdexec{
					log: log,
					c:   fakeCmd(t, tt.execMocks...),
				},
				fs:     fs,
				config: &api.InstallerConfig{RaidEnabled: true, RootUUID: "ace079b5-06be-4429-bbf0-081ea4d7d0d9"},
			}

			uuid, found := i.findMDUUID()
			assert.Equal(t, tt.wantFound, found)
			if diff := cmp.Diff(tt.want, uuid); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}
		})
	}
}

func Test_installer_buildCMDLine(t *testing.T) {
	tests := []struct {
		name      string
		fsMocks   func(fs afero.Fs)
		execMocks []fakeexecparams
		want      string
		wantErr   error
	}{
		{
			name: "without raid",
			fsMocks: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, "/etc/metal/install.yaml", []byte(sampleInstallYAML), 0700))
			},
			execMocks: []fakeexecparams{
				{
					WantCmd:  []string{"blkid"},
					Output:   sampleBlkidOutput,
					ExitCode: 0,
				},
				{
					WantCmd:  []string{"mdadm", "--detail", "--export", "/dev/md1"},
					Output:   sampleMdadmDetailOutput,
					ExitCode: 0,
				},
			},
			// CMDLINE="console=${CONSOLE} root=UUID=${ROOT_UUID} init=/bin/systemd net.ifnames=0 biosdevname=0 nvme_core.io_timeout=4294967295 systemd.unified_cgroup_hierarchy=0"
			want: "console=ttyS1,115200n8 root=UUID=543eb7f8-98d4-d986-e669-824dbebe69e5 init=/bin/systemd net.ifnames=0 biosdevname=0 nvme_core.io_timeout=4294967295 systemd.unified_cgroup_hierarchy=0",
		},
		{
			name: "with raid",
			fsMocks: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, "/etc/metal/install.yaml", []byte(sampleInstallWithRaidYAML), 0700))
			},
			execMocks: []fakeexecparams{
				{
					WantCmd:  []string{"blkid"},
					Output:   sampleBlkidOutput,
					ExitCode: 0,
				},
				{
					WantCmd:  []string{"mdadm", "--detail", "--export", "/dev/md1"},
					Output:   sampleMdadmDetailOutput,
					ExitCode: 0,
				},
			},
			// CMDLINE="console=${CONSOLE} root=UUID=${ROOT_UUID} init=/bin/systemd net.ifnames=0 biosdevname=0 nvme_core.io_timeout=4294967295 systemd.unified_cgroup_hierarchy=0"
			want: "console=ttyS1,115200n8 root=UUID=ace079b5-06be-4429-bbf0-081ea4d7d0d9 init=/bin/systemd net.ifnames=0 biosdevname=0 nvme_core.io_timeout=4294967295 systemd.unified_cgroup_hierarchy=0 rdloaddriver=raid0 rdloaddriver=raid1 rd.md.uuid=543eb7f8:98d4d986:e669824d:bebe69e5",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.fsMocks != nil {
				tt.fsMocks(fs)
			}

			log := zaptest.NewLogger(t).Sugar()

			i := &installer{
				log: log,
				exec: &cmdexec{
					log: log,
					c:   fakeCmd(t, tt.execMocks...),
				},
				fs:     fs,
				config: mustParseInstallYAML(t, fs),
			}

			got, err := i.buildCMDLine()
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}
		})
	}
}

func Test_installer_unsetMachineID(t *testing.T) {
	tests := []struct {
		name    string
		fsMocks func(fs afero.Fs)
		wantErr error
	}{
		{
			name: "unset",
			fsMocks: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, "/etc/machine-id", []byte("uuid"), 0700))
				require.NoError(t, afero.WriteFile(fs, "/var/lib/dbus/machine-id", []byte("uuid"), 0700))
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.fsMocks != nil {
				tt.fsMocks(fs)
			}

			i := &installer{
				log: zaptest.NewLogger(t).Sugar(),
				fs:  fs,
			}

			err := i.unsetMachineID()
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}

			content, err := afero.ReadFile(i.fs, "/etc/machine-id")
			require.NoError(t, err)
			assert.Empty(t, content)

			content, err = afero.ReadFile(i.fs, "/var/lib/dbus/machine-id")
			require.NoError(t, err)
			assert.Empty(t, content)
		})
	}
}

func Test_installer_writeBootInfo(t *testing.T) {
	tests := []struct {
		name    string
		cmdline string
		oss     operatingsystem
		link    afero.LinkReader
		want    *api.Bootinfo
		wantErr error
	}{
		{
			name:    "boot-info",
			cmdline: "a-cmd-line",
			oss:     osUbuntu,
			link: &linkMock{
				mocks: map[string]string{
					"/vmlinuz":         "linuz-test",
					"/initrd.img":      "init-test",
					"/boot/vmlinuz":    "linuz-test",
					"/boot/initrd.img": "init-test",
				},
			},
			want: &api.Bootinfo{
				Initrd:       "init-test",
				Cmdline:      "a-cmd-line",
				Kernel:       "linuz-test",
				BootloaderID: "metal-ubuntu",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			i := &installer{
				log:  zaptest.NewLogger(t).Sugar(),
				fs:   fs,
				link: tt.link,
				oss:  tt.oss,
			}

			err := i.writeBootInfo(tt.cmdline)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}

			content, err := afero.ReadFile(i.fs, "/etc/metal/boot-info.yaml")
			require.NoError(t, err)

			var bootInfo api.Bootinfo
			err = yaml.Unmarshal(content, &bootInfo)
			require.NoError(t, err)

			if diff := cmp.Diff(tt.want, &bootInfo); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}
		})
	}
}
