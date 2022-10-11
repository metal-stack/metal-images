package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func Test_detectOS(t *testing.T) {
	tests := []struct {
		name    string
		fsMocks func(fs afero.Fs)
		want    operatingsystem
		wantErr error
	}{
		{
			name: "ubuntu 22.04 os",
			fsMocks: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, "/etc/os-release", []byte(`PRETTY_NAME="Ubuntu 22.04.1 LTS"
NAME="Ubuntu"
VERSION_ID="22.04"
VERSION="22.04.1 LTS (Jammy Jellyfish)"
VERSION_CODENAME=jammy
ID=ubuntu
ID_LIKE=debian
HOME_URL="https://www.ubuntu.com/"
SUPPORT_URL="https://help.ubuntu.com/"
BUG_REPORT_URL="https://bugs.launchpad.net/ubuntu/"
PRIVACY_POLICY_URL="https://www.ubuntu.com/legal/terms-and-policies/privacy-policy"
UBUNTU_CODENAME=jammy`), 0755))
			},
			want:    osUbuntu,
			wantErr: nil,
		},
		{
			name: "centos 7",
			fsMocks: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, "/etc/os-release", []byte(`NAME="CentOS Linux"
VERSION="7 (Core)"
ID="centos"
ID_LIKE="rhel fedora"
VERSION_ID="7"
PRETTY_NAME="CentOS Linux 7 (Core)"
ANSI_COLOR="0;31"
CPE_NAME="cpe:/o:centos:centos:7"
HOME_URL="https://www.centos.org/"
BUG_REPORT_URL="https://bugs.centos.org/"

CENTOS_MANTISBT_PROJECT="CentOS-7"
CENTOS_MANTISBT_PROJECT_VERSION="7"
REDHAT_SUPPORT_PRODUCT="centos"
REDHAT_SUPPORT_PRODUCT_VERSION="7"`), 0755))
			},
			want:    osCentos,
			wantErr: nil,
		},
		{
			name: "debian 10",
			fsMocks: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, "/etc/os-release", []byte(`PRETTY_NAME="Debian GNU/Linux 10 (buster)"
NAME="Debian GNU/Linux"
VERSION_ID="10"
VERSION="10 (buster)"
VERSION_CODENAME=buster
ID=debian
HOME_URL="https://www.debian.org/"
SUPPORT_URL="https://www.debian.org/support"
BUG_REPORT_URL="https://bugs.debian.org/"`), 0755))
			},
			want:    osDebian,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			if tt.fsMocks != nil {
				tt.fsMocks(fs)
			}

			oss, err := detectOS(fs)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}
			if diff := cmp.Diff(tt.want, oss); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}
		})
	}
}
