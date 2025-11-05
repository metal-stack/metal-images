package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/storage"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	docker "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
)

type artifact struct {
	os          string
	version     string
	image       string
	dockerImage string
	dockerTags  []string
	url         string
	checksumURL string
	packagesURL string

	gcsSrcSuffix  string
	gcsDestSuffix string
}

const (
	ghcrPrefix = "ghcr.io/metal-stack"
)

var (
	dryRun = flag.Bool("dry-run", false, "print info what would happen if run with --dry-run")
)

func main() {
	flag.Parse()
	err := generate()
	if err != nil {
		panic(err)
	}
}

func generate() error {
	var (
		dummyRegion = "dummy" // we don't use AWS S3, we don't need a proper region
		endpoint    = "metal-stack.io"
		bucket      = "images"
		prefix      = os.Getenv("PREFIX") // "metal-os/20230710" or "metal-os/stable"
		whitelist   []string
	)

	err := json.Unmarshal([]byte(os.Getenv("DISTRO_VERSIONS")), &whitelist)
	if err != nil {
		return fmt.Errorf("unable to unmarshal DISTRO_VERSIONS: %v", err)
	}

	ss, err := session.NewSession(&aws.Config{
		Endpoint:    &endpoint,
		Region:      &dummyRegion,
		Credentials: credentials.AnonymousCredentials,
		Retryer: client.DefaultRetryer{
			NumMaxRetries: 3,
		},
	})
	if err != nil {
		return err
	}

	var (
		client = s3.New(ss)
		res    = map[string]artifact{}
	)

	err = client.ListObjectsPages(&s3.ListObjectsInput{
		Bucket: &bucket,
		Prefix: &prefix,
	}, func(objects *s3.ListObjectsOutput, lastPage bool) bool {
		for _, o := range objects.Contents {
			key := *o.Key

			after, found := strings.CutPrefix(key, prefix)
			if !found {
				continue
			}

			base := path.Dir(key)
			a := res[base]
			url := fmt.Sprintf("https://%s.%s/%s%s", bucket, endpoint, prefix, after)
			a.image = fmt.Sprintf("%s%s", prefix, path.Dir(after))

			parts := strings.Split(strings.TrimPrefix(after, "/"), "/")
			if len(parts) > 2 {
				operatingSystem := parts[0]
				version := parts[1]

				osVersion := strings.Join([]string{operatingSystem, version}, "/")
				if !slices.Contains(whitelist, osVersion) {
					continue
				}

				a.dockerImage = fmt.Sprintf("%s/%s:%s-stable", ghcrPrefix, operatingSystem, version)
				a.os = operatingSystem
				a.version = version

				a.dockerTags = []string{
					strings.TrimSuffix(a.dockerImage, "-stable"),
					fmt.Sprintf("%s/%s:latest", ghcrPrefix, a.os),
				}

				a.gcsSrcSuffix = fmt.Sprintf("metal-os/stable/%s/%s", operatingSystem, version)
				a.gcsDestSuffix = fmt.Sprintf("metal-os/%s/%s/%s", os.Getenv("REF"), operatingSystem, version)
			}

			switch {
			case strings.HasSuffix(key, ".tar.lz4"):
				a.url = url
			case strings.HasSuffix(key, ".md5"):
				a.checksumURL = url
			case strings.HasSuffix(key, ".txt"):
				a.packagesURL = url
			}

			res[base] = a
		}

		return true
	})
	if err != nil {
		return fmt.Errorf("cannot list s3 objects:%w", err)
	}

	var artifacts []*artifact
	for _, a := range res {
		artifacts = append(artifacts, &a)
	}

	err = release(artifacts)
	if err != nil {
		return err
	}

	return print(artifacts)
}

func release(artifacts []*artifact) error {
	if *dryRun {
		for _, a := range artifacts {
			logRunOutput(a)
		}

		return nil
	}

	ctx := context.Background()
	cli, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %v", err)
	}
	defer cli.Close() // nolint:errcheck

	token := os.Getenv("TOKEN")
	if token == "" {
		return fmt.Errorf("registry token is missing. Please provide TOKEN env variable")
	}

	var authConfigBase64 string
	authConfig := registry.AuthConfig{
		Username:      "metal-stack",
		Password:      token,
		ServerAddress: "ghcr.io",
	}
	authConfigBytes, err := json.Marshal(authConfig)
	if err != nil {
		return fmt.Errorf("error encoding authConfig: %v", err)
	}
	authConfigBase64 = base64.URLEncoding.EncodeToString(authConfigBytes)

	for _, a := range artifacts {
		logRunOutput(a)
		sourceImage := a.dockerImage

		pullReader, err := cli.ImagePull(ctx, sourceImage, image.PullOptions{RegistryAuth: authConfigBase64})
		if err != nil {
			return fmt.Errorf("image pull failed: %v", err)
		}
		defer pullReader.Close() // nolint:errcheck
		renderDockerOutput(pullReader)

		for _, t := range a.dockerTags {
			err = cli.ImageTag(ctx, sourceImage, t)
			if err != nil {
				return fmt.Errorf("image tag failed: %v", err)
			}

			pushReader, err := cli.ImagePush(ctx, t, image.PushOptions{RegistryAuth: authConfigBase64})
			if err != nil {
				return fmt.Errorf("image push failed: %v", err)
			}
			defer pushReader.Close() // nolint:errcheck
			renderDockerOutput(pushReader)
		}

		fmt.Println()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		client, err := storage.NewClient(ctx)
		if err != nil {
			return fmt.Errorf("creating a new gcs client failed: %v", err)
		}
		defer client.Close() // nolint:errcheck

		bucket := client.Bucket(os.Getenv("GCS_BUCKET"))
		src := bucket.Object(a.gcsSrcSuffix)
		dest := bucket.Object(a.gcsDestSuffix)

		copier := dest.CopierFrom(src)
		_, err = copier.Run(ctx)
		if err != nil {
			return fmt.Errorf("copying resources from %s to %s failed: %v", a.gcsSrcSuffix, a.gcsDestSuffix, err)
		}

		fmt.Println()
	}

	return nil
}

func print(artifacts []*artifact) error {
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].url < artifacts[j].url
	})

	filename := os.Getenv("FILENAME")
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", filename, err)
	}
	defer f.Close() // nolint:errcheck

	_, err = f.WriteString("## Downloads\n\n")
	if err != nil {
		return fmt.Errorf("error writing heading to file %s: %v", filename, err)
	}

	printerConfig := &printers.TablePrinterConfig{
		Markdown: true,
		Out:      f,
	}

	p := printers.NewTablePrinter(printerConfig)

	printerConfig.ToHeaderAndRows = func(data any, wide bool) ([]string, [][]string, error) {
		p.DisableAutoWrap(true)

		switch data.(type) {
		case []*artifact:
			var (
				header = []string{"IMAGE", "URL", "CHECKSUM", "PACKAGES"}
				rows   [][]string
			)

			for _, a := range artifacts {
				url := fmt.Sprintf("[%s](%s)", path.Base(a.url), a.url)
				checksum := fmt.Sprintf("[%s](%s)", path.Base(a.checksumURL), a.checksumURL)
				packages := fmt.Sprintf("[%s](%s)", path.Base(a.packagesURL), a.packagesURL)

				rows = append(rows, []string{a.image, url, checksum, packages})
			}

			return header, rows, nil
		}

		return nil, nil, fmt.Errorf("unsupported type for printing: %T", data)
	}

	err = p.Print(artifacts)
	if err != nil {
		return fmt.Errorf("error printing table: %v", err)
	}

	return nil
}

func logRunOutput(a *artifact) {
	fmt.Printf("tagging docker image: %s", a.dockerImage)
	fmt.Println()
	for _, t := range a.dockerTags {
		fmt.Printf("with %s", t)
		fmt.Println()
	}
	fmt.Println()
	fmt.Printf("copying gcs data from: %s", a.gcsSrcSuffix)
	fmt.Println()
	fmt.Printf("copying gcs data to: %s", a.gcsDestSuffix)
	fmt.Println()
	fmt.Println()
	fmt.Println()
}

func renderDockerOutput(reader io.ReadCloser) {
	id, isTerm := term.GetFdInfo(os.Stdout)
	_ = jsonmessage.DisplayJSONMessagesStream(reader, os.Stdout, id, isTerm, nil)
}
