package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"sort"
	"strings"

	"cloud.google.com/go/storage"

	"github.com/charmbracelet/lipgloss"
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

	distroVersions = "DISTRO_VERSIONS"
	filename       = "FILENAME"
	gcsBucket      = "GCS_BUCKET"
	gitRefName     = "REF_NAME"
	token          = "TOKEN"
)

var (
	dryRun = flag.Bool("dry-run", false, "print info what would happen if run with --dry-run")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		panic(err)
	}
}

func run() error {
	var (
		gcsPrefix   = "metal-os/stable"
		dummyRegion = "dummy" // we don't use AWS S3, we don't need a proper region
		endpoint    = "metal-stack.io"
		bucket      = "images"
		whitelist   []string
	)

	whitelistString, err := getEnvVar(distroVersions)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(whitelistString), &whitelist)
	if err != nil {
		return fmt.Errorf("unable to unmarshal %s: %v", distroVersions, err)
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

	gitRefNameVal, err := getEnvVar(gitRefName)
	if err != nil {
		return err
	}

	err = client.ListObjectsPages(&s3.ListObjectsInput{
		Bucket: &bucket,
		Prefix: &gcsPrefix,
	}, func(objects *s3.ListObjectsOutput, lastPage bool) bool {
		for _, o := range objects.Contents {
			key := *o.Key

			after, found := strings.CutPrefix(key, gcsPrefix)
			if !found {
				continue
			}

			base := path.Dir(key)
			a := res[base]
			url := fmt.Sprintf("https://%s.%s/%s%s", bucket, endpoint, gcsPrefix, after)
			a.image = fmt.Sprintf("%s%s", gcsPrefix, path.Dir(after))

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
				a.gcsDestSuffix = fmt.Sprintf("metal-os/%s/%s/%s", gitRefNameVal, operatingSystem, version)
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

	if *dryRun {
		for i, a := range artifacts {
			err = logRunOutput(a, i == 0)
			if err != nil {
				return err
			}
		}

		return nil
	}

	err = tagImages(artifacts)
	if err != nil {
		return err
	}
	fmt.Println()

	gcsBucketVal, err := getEnvVar(gcsBucket)
	if err != nil {
		return err
	}
	err = copyGcsObjects(artifacts, gcsBucketVal, nil)
	if err != nil {
		return err
	}

	return printDownloadsMarkdown(artifacts)
}

func tagImages(artifacts []*artifact) error {
	var (
		errs []error
	)

	ctx := context.Background()
	cli, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %v", err)
	}
	defer func() {
		if err = cli.Close(); err != nil {
			errs = append(errs, err)
		}
	}()

	tokenVal, err := getEnvVar(token)
	if err != nil {
		errs = append(errs, err)
		return errors.Join(errs...)
	}
	var authConfigBase64 string
	authConfig := registry.AuthConfig{
		Username:      "metal-stack",
		Password:      tokenVal,
		ServerAddress: "ghcr.io",
	}
	authConfigBytes, err := json.Marshal(authConfig)
	if err != nil {
		errs = append(errs, fmt.Errorf("error encoding authConfig: %v", err))
		return errors.Join(errs...)
	}
	authConfigBase64 = base64.URLEncoding.EncodeToString(authConfigBytes)

	for _, a := range artifacts {
		sourceImage := a.dockerImage

		pullReader, err := cli.ImagePull(ctx, sourceImage, image.PullOptions{All: false, RegistryAuth: authConfigBase64})
		if err != nil {
			errs = append(errs, fmt.Errorf("image pull failed: %v", err))
			return errors.Join(errs...)
		}
		defer func() {
			if err = pullReader.Close(); err != nil {
				errs = append(errs, err)
			}
		}()
		err = renderDockerOutput(pullReader)
		if err != nil {
			errs = append(errs, err)
			return errors.Join(errs...)
		}

		for _, t := range a.dockerTags {
			err = cli.ImageTag(ctx, sourceImage, t)
			if err != nil {
				errs = append(errs, fmt.Errorf("image tag failed: %v", err))
				return errors.Join(errs...)
			}

			pushReader, err := cli.ImagePush(ctx, t, image.PushOptions{RegistryAuth: authConfigBase64})
			if err != nil {
				errs = append(errs, fmt.Errorf("image push failed: %v", err))
				return errors.Join(errs...)
			}
			defer func() {
				if err = pushReader.Close(); err != nil {
					errs = append(errs, err)
				}
			}()
			err = renderDockerOutput(pushReader)
			if err != nil {
				errs = append(errs, err)
				return errors.Join(errs...)
			}
		}
	}

	return errors.Join(errs...)
}

func copyGcsObjects(artifacts []*artifact, gcsBucketVal string, client *storage.Client) error {
	var (
		ctx  = context.Background()
		err  error
		errs []error
	)

	if client == nil {
		client, err = storage.NewClient(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("creating a new gcs client failed: %v", err))
			return errors.Join(errs...)
		}
		defer func() {
			if err = client.Close(); err != nil {
				errs = append(errs, err)
			}
		}()
	}

	for _, a := range artifacts {
		bucket := client.Bucket(gcsBucketVal)
		src := bucket.Object(a.gcsSrcSuffix)
		dest := bucket.Object(a.gcsDestSuffix)

		copier := dest.CopierFrom(src)
		_, err := copier.Run(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("copying resources from %s to %s failed: %v", a.gcsSrcSuffix, a.gcsDestSuffix, err))
			return errors.Join(errs...)
		}

		fmt.Println()
	}

	return errors.Join(errs...)
}

func printDownloadsMarkdown(artifacts []*artifact) error {
	var (
		errs []error
	)

	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].url < artifacts[j].url
	})

	fn, err := getEnvVar(filename)
	if err != nil {
		return err
	}
	f, err := os.Create(fn)
	if err != nil {
		errs = append(errs, fmt.Errorf("error creating file %s: %v", fn, err))
		return errors.Join(errs...)
	}
	defer func() {
		if err = f.Close(); err != nil {
			errs = append(errs, err)
		}
	}()

	_, err = f.WriteString("## Downloads\n\n")
	if err != nil {
		errs = append(errs, fmt.Errorf("error writing heading to file %s: %v", fn, err))
		return errors.Join(errs...)
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

		errs = append(errs, fmt.Errorf("unsupported type for printing: %T", data))
		return nil, nil, errors.Join(errs...)
	}

	err = p.Print(artifacts)
	if err != nil {
		errs = append(errs, fmt.Errorf("error printing table: %v", err))
		return errors.Join(errs...)
	}

	return errors.Join(errs...)
}

func getEnvVar(envVarName string) (string, error) {
	envVar := os.Getenv(envVarName)
	if envVar == "" {
		return "", fmt.Errorf("environment variable not set: %s", envVarName)
	}

	return envVar, nil
}

func logRunOutput(a *artifact, isFirst bool) error {
	physicalWidth, err := term.GetWinsize(os.Stdout.Fd())
	if err != nil {
		return err
	}

	contentFormat := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
	dockerGcsBorder := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#0B6623")).
		Render(strings.Repeat("─ ", int(physicalWidth.Width)/2))
	globalBorder := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9B59FF")).
		Render(strings.Repeat("─", int(physicalWidth.Width)-1))

	if !isFirst {
		fmt.Println(globalBorder)
	}
	fmt.Printf("tag docker image: %s\n", a.dockerImage)
	for i, t := range a.dockerTags {
		if i == 0 {
			fmt.Printf("also as %s\n", contentFormat.Render(t))
			continue
		}

		fmt.Printf("and %s\n", contentFormat.Render(t))
	}
	fmt.Println(dockerGcsBorder)
	fmt.Printf("copy gcs data from: %s\n", a.gcsSrcSuffix)
	fmt.Printf("to: %s\n", contentFormat.Render(a.gcsDestSuffix))

	return nil
}

func renderDockerOutput(reader io.ReadCloser) error {
	id, isTerm := term.GetFdInfo(os.Stdout)
	err := jsonmessage.DisplayJSONMessagesStream(reader, os.Stdout, id, isTerm, nil)

	return err
}
