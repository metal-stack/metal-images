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
)

const (
	sampleDiskJSON = `{
	"Device": "legacy",
	"Partitions": [
		{
		"Label": "root",
		"Filesystem": "ext4",
		"Properties": {
			"UUID": "ace079b5-06be-4429-bbf0-081ea4d7d0d9"
		}
		},
		{
		"Label": "efi",
		"Filesystem": "vfat",
		"Properties": {
			"UUID": "C236-297F"
		}
		},
		{
		"Label": "varlib",
		"Filesystem": "ext4",
		"Properties": {
			"UUID": "385e8e8e-dbfd-481e-93a4-cba7f4d5fa02"
		}
		}
	]
	}`
	sampleBlkidOutput = `
 /dev/sda1: UUID="42d10089-ee1e-0399-445e-755062b63ec8" UUID_SUB="cc57c456-0b2f-6345-c597-d861cc6dd8ac" LABEL="any:0" TYPE="linux_raid_member" PARTLABEL="efi" PARTUUID="273985c8-d097-4123-bcd0-80b4e4e14728"
 /dev/sda2: UUID="543eb7f8-98d4-d986-e669-824dbebe69e5" UUID_SUB="54748c60-b566-f391-142c-fb78bb1fc6a9" LABEL="any:1" TYPE="linux_raid_member" PARTLABEL="root" PARTUUID="d7863f4e-af7c-47fc-8c03-6ecdc69bc72d"
 /dev/sda3: UUID="fc32a6f0-ee40-d9db-87c8-c9f3a8400c8b" UUID_SUB="582e9b4f-f191-e01e-85fd-2f7d969fbef6" LABEL="any:2" TYPE="linux_raid_member" PARTLABEL="varlib" PARTUUID="e8b44f09-b7f7-4e0d-a7c3-d909617d1f05"
 /dev/sdb1: UUID="42d10089-ee1e-0399-445e-755062b63ec8" UUID_SUB="61bd5d8b-1bb8-673b-9e61-8c28dccc3812" LABEL="any:0" TYPE="linux_raid_member" PARTLABEL="efi" PARTUUID="13a4c568-57b0-4259-9927-9ac023aaa5f0"
 /dev/sdb2: UUID="543eb7f8-98d4-d986-e669-824dbebe69e5" UUID_SUB="e7d01e93-9340-5b90-68f8-d8f815595132" LABEL="any:1" TYPE="linux_raid_member" PARTLABEL="root" PARTUUID="ab11cd86-37b8-4bae-81e5-21fe0a9c9ae0"
 /dev/sdb3: UUID="fc32a6f0-ee40-d9db-87c8-c9f3a8400c8b" UUID_SUB="764217ad-1591-a83a-c799-23397f968729" LABEL="any:2" TYPE="linux_raid_member" PARTLABEL="varlib" PARTUUID="9afbf9c1-b2ba-4b46-8db1-e802d26c93b6"
 /dev/md1: LABEL="root" UUID="ace079b5-06be-4429-bbf0-081ea4d7d0d9" TYPE="ext4"
 /dev/md0: LABEL="efi" UUID="C236-297F" TYPE="vfat"
 /dev/md2: LABEL="varlib" UUID="385e8e8e-dbfd-481e-93a4-cba7f4d5fa02" TYPE="ext4"`
)

func Test_installer_detectFirmware(t *testing.T) {
	tests := []struct {
		name    string
		fsMocks func(fs afero.Fs)
		wantErr error
	}{
		{
			name: "is efi",
			fsMocks: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, "/sys/firmware/efi", []byte(""), 0755))
			},
			wantErr: nil,
		},
		{
			name:    "is not efi",
			wantErr: fmt.Errorf("not running efi mode"),
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

func Test_installer_rootUUID(t *testing.T) {
	tests := []struct {
		name     string
		diskJSON string
		want     string
		wantErr  error
	}{
		{
			name:     "find root uuid in disk.json",
			diskJSON: sampleDiskJSON,
			want:     "ace079b5-06be-4429-bbf0-081ea4d7d0d9",
			wantErr:  nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			i := &installer{
				log:  zaptest.NewLogger(t).Sugar(),
				fs:   fs,
				disk: mustParseDiskJSON(t, fs, tt.diskJSON),
			}

			got, err := i.rootUUID()
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}
		})
	}
}

func mustParseDiskJSON(t *testing.T, fs afero.Fs, json string) *api.Disk {
	require.NoError(t, afero.WriteFile(fs, "/etc/metal/disk.json", []byte(json), 0700))
	disk, err := parseDiskJSON(fs)
	require.NoError(t, err)
	return disk
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

func Test_installer_checkForMD(t *testing.T) {
	tests := []struct {
		name      string
		execMocks []fakeexecparams
		want      bool
	}{
		{
			name: "no mdadm command found",
			execMocks: []fakeexecparams{
				{
					WantCmd:  []string{"mdadm", "--examine", "--scan"},
					ExitCode: 127,
				},
			},
			want: false,
		},
		{
			name: "has mdadm",
			execMocks: []fakeexecparams{
				{
					WantCmd:  []string{"mdadm", "--examine", "--scan"},
					ExitCode: 0,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			log := zaptest.NewLogger(t).Sugar()
			i := &installer{
				log: log,
				exec: &cmdexec{
					log: log,
					c:   fakeCmd(t, tt.execMocks...),
				},
			}

			got := i.checkForMD()
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_installer_findMDUUID(t *testing.T) {
	tests := []struct {
		name      string
		execMocks []fakeexecparams
		diskJSON  string
		want      string
		wantFound bool
	}{
		{
			name: "no mdadm command found",
			execMocks: []fakeexecparams{
				{
					WantCmd:  []string{"mdadm", "--examine", "--scan"},
					Output:   "",
					ExitCode: 127,
				},
			},
			want:      "",
			wantFound: false,
		},
		{
			name: "has mdadm",
			execMocks: []fakeexecparams{
				{
					WantCmd:  []string{"mdadm", "--examine", "--scan"},
					Output:   "",
					ExitCode: 0,
				},
				{
					WantCmd:  []string{"blkid"},
					Output:   sampleBlkidOutput,
					ExitCode: 0,
				},
				{
					WantCmd: []string{"mdadm", "--detail", "--export", "/dev/md1"},
					Output: `MD_LEVEL=raid1
MD_DEVICES=2
MD_METADATA=1.0
MD_UUID=543eb7f8:98d4d986:e669824d:bebe69e5
MD_DEVNAME=1
MD_NAME=any:1
MD_DEVICE_dev_sdb2_ROLE=1
MD_DEVICE_dev_sdb2_DEV=/dev/sdb2
MD_DEVICE_dev_sda2_ROLE=0
MD_DEVICE_dev_sda2_DEV=/dev/sda2`,
					ExitCode: 0,
				},
			},
			diskJSON:  sampleDiskJSON,
			want:      "543eb7f8:98d4d986:e669824d:bebe69e5",
			wantFound: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			log := zaptest.NewLogger(t).Sugar()

			i := &installer{
				log: log,
				exec: &cmdexec{
					log: log,
					c:   fakeCmd(t, tt.execMocks...),
				},
				fs: fs,
			}

			if tt.diskJSON != "" {
				i.disk = mustParseDiskJSON(t, fs, tt.diskJSON)
			}

			uuid, found := i.findMDUUID()
			assert.Equal(t, tt.wantFound, found)
			if diff := cmp.Diff(tt.want, uuid); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}
		})
	}
}
