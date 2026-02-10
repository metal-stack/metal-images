package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/docker/docker/api/types/container"
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
	srcObjectPrefix   = "path/to/"
	destObjectPrefix  = "dest/"
	testObjectContent = "hello from src"
)

func Test_CopyGcsObjects(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
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
		filename := "file.txt"
		src := bucket.Object(filepath.Join(srcObjectPrefix, filename))
		w := src.NewWriter(ctx)
		_, err = fmt.Fprint(w, testObjectContent)
		require.NoError(t, err)
		err = w.Close()
		require.NoError(t, err)

		artifacts := []*artifact{
			{
				gcsSrcPrefix:  srcObjectPrefix,
				gcsDestPrefix: destObjectPrefix,
			},
		}
		err = copyGcsObjects(artifacts, testBucket, client)
		require.NoError(t, err)

		// verify that the file was copied
		dest := bucket.Object(filepath.Join(destObjectPrefix, filename))
		rc, err := dest.NewReader(ctx)
		require.NotNil(t, rc)
		require.NoError(t, err)

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, rc)
		require.NoError(t, err)
		require.Equal(t, testObjectContent, buf.String())

		err = rc.Close()
		require.NoError(t, err)
	})

	t.Run("copy multiple files", func(t *testing.T) {
		for i := range 3 {
			filename := fmt.Sprintf("file_%d.txt", i)
			src := bucket.Object(filepath.Join(srcObjectPrefix, filename))
			w := src.NewWriter(ctx)
			_, err = fmt.Fprintf(w, "content of file %d", i)
			require.NoError(t, err)
			err = w.Close()
			require.NoError(t, err)
		}

		artifacts := []*artifact{
			{
				gcsSrcPrefix:  srcObjectPrefix,
				gcsDestPrefix: destObjectPrefix,
			},
		}
		err = copyGcsObjects(artifacts, testBucket, client)
		require.NoError(t, err)

		// verify that all files were copied
		for i := range 3 {
			filename := fmt.Sprintf("file_%d.txt", i)
			dest := bucket.Object(filepath.Join(destObjectPrefix, filename))
			rc, err := dest.NewReader(ctx)
			require.NoError(t, err)

			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, rc)
			require.NoError(t, err)
			require.Equal(t, fmt.Sprintf("content of file %d", i), buf.String())

			err = rc.Close()
			require.NoError(t, err)
		}
	})

	t.Run("copy with different file types", func(t *testing.T) {
		files := []struct {
			name    string
			content string
		}{
			{"file.txt", "text content"},
			{"image.jpg", "image content"},
			{"data.json", `{"key": "value"}`},
		}

		for _, file := range files {
			src := bucket.Object(filepath.Join(srcObjectPrefix, file.name))
			w := src.NewWriter(ctx)
			_, err = fmt.Fprint(w, file.content)
			require.NoError(t, err)
			err = w.Close()
			require.NoError(t, err)
		}

		artifacts := []*artifact{
			{
				gcsSrcPrefix:  srcObjectPrefix,
				gcsDestPrefix: destObjectPrefix,
			},
		}
		err = copyGcsObjects(artifacts, testBucket, client)
		require.NoError(t, err)

		// verify that all files were copied
		for _, file := range files {
			dest := bucket.Object(filepath.Join(destObjectPrefix, file.name))
			rc, err := dest.NewReader(ctx)
			require.NoError(t, err)

			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, rc)
			require.NoError(t, err)
			require.Equal(t, file.content, buf.String())

			err = rc.Close()
			require.NoError(t, err)
		}
	})

	t.Run("copy with nested prefixes", func(t *testing.T) {
		nestedFiles := []struct {
			name    string
			content string
		}{
			{"subdir/file1.txt", "content of file1"},
			{"subdir/subsubdir/file2.txt", "content of file2"},
		}

		for _, file := range nestedFiles {
			src := bucket.Object(filepath.Join(srcObjectPrefix, file.name))
			w := src.NewWriter(ctx)
			_, err = fmt.Fprint(w, file.content)
			require.NoError(t, err)
			err = w.Close()
			require.NoError(t, err)
		}

		artifacts := []*artifact{
			{
				gcsSrcPrefix:  srcObjectPrefix,
				gcsDestPrefix: destObjectPrefix,
			},
		}
		err = copyGcsObjects(artifacts, testBucket, client)
		require.NoError(t, err)

		// verify that all files were copied
		for _, file := range nestedFiles {
			dest := bucket.Object(filepath.Join(destObjectPrefix, file.name))
			rc, err := dest.NewReader(ctx)
			require.NoError(t, err)

			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, rc)
			require.NoError(t, err)
			require.Equal(t, file.content, buf.String())

			err = rc.Close()
			require.NoError(t, err)
		}
	})

	t.Run("copy with special characters in filenames", func(t *testing.T) {
		files := []struct {
			name    string
			content string
		}{
			{"file with spaces.txt", "content with spaces"},
			{"file-with-dashes.txt", "content with dashes"},
			{"file_with_underscores.txt", "content with underscores"},
		}

		for _, file := range files {
			src := bucket.Object(filepath.Join(srcObjectPrefix, file.name))
			w := src.NewWriter(ctx)
			_, err = fmt.Fprint(w, file.content)
			require.NoError(t, err)
			err = w.Close()
			require.NoError(t, err)
		}

		artifacts := []*artifact{
			{
				gcsSrcPrefix:  srcObjectPrefix,
				gcsDestPrefix: destObjectPrefix,
			},
		}
		err = copyGcsObjects(artifacts, testBucket, client)
		require.NoError(t, err)

		// verify that all files were copied
		for _, file := range files {
			dest := bucket.Object(filepath.Join(destObjectPrefix, file.name))
			rc, err := dest.NewReader(ctx)
			require.NoError(t, err)

			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, rc)
			require.NoError(t, err)
			require.Equal(t, file.content, buf.String())

			err = rc.Close()
			require.NoError(t, err)
		}
	})

	t.Run("copy with overlapping prefixes", func(t *testing.T) {
		filename := "file.txt"
		src := bucket.Object(filepath.Join(srcObjectPrefix, filename))
		w := src.NewWriter(ctx)
		_, err = fmt.Fprint(w, testObjectContent)
		require.NoError(t, err)
		err = w.Close()
		require.NoError(t, err)

		overlapPrefix := "path/to/dest/"
		artifacts := []*artifact{
			{
				gcsSrcPrefix:  srcObjectPrefix,
				gcsDestPrefix: overlapPrefix,
			},
		}
		err = copyGcsObjects(artifacts, testBucket, client)
		require.NoError(t, err)

		// verify that the file was copied
		dest := bucket.Object(filepath.Join(overlapPrefix, filename))
		rc, err := dest.NewReader(ctx)
		require.NoError(t, err)

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, rc)
		require.NoError(t, err)
		require.Equal(t, testObjectContent, buf.String())

		err = rc.Close()
		require.NoError(t, err)
	})

	t.Run("copy with large files", func(t *testing.T) {
		largeContent := make([]byte, 10*1024*1024) // 10MB
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}

		filename := "large_file.dat"
		src := bucket.Object(filepath.Join(srcObjectPrefix, filename))
		w := src.NewWriter(ctx)
		_, err = w.Write(largeContent)
		require.NoError(t, err)
		err = w.Close()
		require.NoError(t, err)

		artifacts := []*artifact{
			{
				gcsSrcPrefix:  srcObjectPrefix,
				gcsDestPrefix: destObjectPrefix,
			},
		}
		err = copyGcsObjects(artifacts, testBucket, client)
		require.NoError(t, err)

		// verify that the file was copied
		dest := bucket.Object(filepath.Join(destObjectPrefix, filename))
		rc, err := dest.NewReader(ctx)
		require.NoError(t, err)

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, rc)
		require.NoError(t, err)
		require.Equal(t, largeContent, buf.Bytes())

		err = rc.Close()
		require.NoError(t, err)
	})
}

// HELPER FUNCTIONS
func startFakeGcsContainer(t testing.TB, ctx context.Context) (testcontainers.Container, *connectionDetails) {
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "fsouza/fake-gcs-server", // tested with fsouza/fake-gcs-server:1.47.4
			HostConfigModifier: func(hc *container.HostConfig) {
				// Unfortunately we must use host network as the public host must exactly match the client endpoint
				// see for example: https://github.com/fsouza/fake-gcs-server/issues/196
				//
				// without it the download does not work because the server directs to the wrong (public?) endpoint
				hc.NetworkMode = "host"
			},
			Cmd: []string{"-backend", "memory", "-log-level", "debug", "-public-host", "localhost:4443"},
			WaitingFor: wait.ForAll(
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
