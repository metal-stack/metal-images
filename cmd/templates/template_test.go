package templates

import (
	_ "embed"
	"os"
	"testing"

	"github.com/metal-stack/metal-go/api/models"
	"github.com/stretchr/testify/require"
)

func TestDefaultChronyTemplate(t *testing.T) {
	defaultNTPServer := "time.cloudflare.com"
	ntpServers := []*models.MetalNTPServer{
		{
			Address: &defaultNTPServer,
		},
	}

	rendered := renderToString(t, Chrony{NTPServers: ntpServers})
	expected := readExpected(t, "test_data/defaultntp/chrony.conf")

	require.Equal(t, expected, rendered, "Wanted: %s\nGot: %s", expected, rendered)
}

func TestCustomChronyTemplate(t *testing.T) {
	customNTPServer := "custom.1.ntp.org"
	ntpServers := []*models.MetalNTPServer{
		{
			Address: &customNTPServer,
		},
	}

	rendered := renderToString(t, Chrony{NTPServers: ntpServers})
	expected := readExpected(t, "test_data/customntp/chrony.conf")

	require.Equal(t, expected, rendered, "Wanted: %s\nGot: %s", expected, rendered)
}

func readExpected(t *testing.T, e string) string {
	ex, err := os.ReadFile(e)
	require.NoError(t, err, "Couldn't read %s", e)
	return string(ex)
}

func renderToString(t *testing.T, c Chrony) string {
	r, err := RenderChronyTemplate(c)
	require.NoError(t, err, "Could not render chrony configuration")
	return r
}
