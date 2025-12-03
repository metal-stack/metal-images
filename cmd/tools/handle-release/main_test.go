package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"testing"

	"cloud.google.com/go/storage"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tlog "github.com/testcontainers/testcontainers-go/log"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/api/option"
)

type connectionDetails struct {
	Endpoint string
}

const (
	testBucket        = "test-bucket"
	testProjectID     = "test-project"
	srcObjectName     = "path/to/src-file.txt"
	destObjectName    = "dest/file.txt"
	nonexistentSrc    = "path/to/missing-file.txt"
	invalidDestName   = "" // invalid empty object name to force failure
	testObjectContent = "hello from src"
)

func Test_CopyGcsObjects(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	c, conn := startFakeGcsContainer(t, ctx)
	defer func() {
		if t.Failed() {
			r, err := c.Logs(ctx)
			require.NoError(t, err)

			if err == nil {
				logs, err := io.ReadAll(r)
				require.NoError(t, err)

				fmt.Println(string(logs))
			}
		}
		err := c.Terminate(ctx)
		require.NoError(t, err)
	}()

	var (
		endpoint = conn.Endpoint + "/storage/v1/"
		tlsCfg   = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
		httpClient = &http.Client{Transport: tlsCfg}
	)

	client, err := storage.NewClient(ctx, option.WithEndpoint(endpoint), option.WithHTTPClient(httpClient))
	require.NoError(t, err)
	defer func() {
		err = client.Close()
		require.NoError(t, err)
	}()

	bucket := client.Bucket(testBucket)
	err = bucket.Create(ctx, testProjectID, nil)
	require.NoError(t, err)

	t.Run("copy successfully", func(t *testing.T) {
		src := bucket.Object(srcObjectName)
		w := src.NewWriter(ctx)
		_, err = fmt.Fprint(w, testObjectContent)
		require.NoError(t, err)
		err = w.Close()
		require.NoError(t, err)

		artifacts := []*artifact{
			{
				gcsSrcSuffix:  srcObjectName,
				gcsDestSuffix: destObjectName,
			},
		}
		err = copyGcsObjects(artifacts, testBucket, client)
		require.NoError(t, err)

		// read destination and verify contents
		dest := bucket.Object(destObjectName)
		rc, err := dest.NewReader(ctx)
		require.NotNil(t, rc)
		require.NoError(t, err)

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, rc)
		require.NoError(t, err)
		assert.Equal(t, testObjectContent, buf.String())

		err = rc.Close()
		require.NoError(t, err)
	})

	t.Run("copy non-existent file", func(t *testing.T) {
		artifacts := []*artifact{
			{
				gcsSrcSuffix:  nonexistentSrc,
				gcsDestSuffix: destObjectName,
			},
		}
		err = copyGcsObjects(artifacts, testBucket, client)
		require.Error(t, err) // FIXME: compare if error is of type storage.ErrObjectNotExist
	})

	t.Run("copy to non-existent destination", func(t *testing.T) {
		artifacts := []*artifact{
			{
				gcsSrcSuffix:  srcObjectName,
				gcsDestSuffix: invalidDestName,
			},
		}
		err = copyGcsObjects(artifacts, testBucket, client)
		require.Error(t, err) // FIXME: compare to "object name is empty"
	})
}

// HELPER FUNCTIONS
func startFakeGcsContainer(t testing.TB, ctx context.Context) (testcontainers.Container, *connectionDetails) {
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "fsouza/fake-gcs-server", // tested with fsouza/fake-gcs-server:1.47.4
			// ExposedPorts: []string{"4443"},
			HostConfigModifier: func(hc *container.HostConfig) {
				// Unfortunately we must use host network as the public host must exactly match the client endpoint
				// see for example: https://github.com/fsouza/fake-gcs-server/issues/196
				//
				// without it the download does not work because the server directs to the wrong (public?) endpoint
				hc.NetworkMode = "host"
			},
			Cmd: []string{"-backend", "memory", "-log-level", "debug", "-public-host", "localhost:4443"},
			WaitingFor: wait.ForAll(
				// wait.ForListeningPort("4443/tcp"),
				wait.ForLog("server started"),
			),
		},
		Started: true,
		Logger:  tlog.TestLogger(t),
	})
	require.NoError(t, err)

	conn := &connectionDetails{
		Endpoint: "https://localhost:4443",
	}

	return c, conn
}
