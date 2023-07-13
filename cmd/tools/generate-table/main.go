package main

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/olekukonko/tablewriter"
)

type artifact struct {
	image       string
	url         string
	checksumURL string
	packagesURL string
}

func main() {
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
		prefix      = "metal-os/20230710"
	)

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

			if strings.HasSuffix(key, ".tar.lz4") {
				a.url = url
			} else if strings.HasSuffix(key, ".md5") {
				a.checksumURL = url
			} else if strings.HasSuffix(key, ".txt") {
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
		a := a
		artifacts = append(artifacts, &a)
	}

	return print(artifacts)
}

func print(artifacts []*artifact) error {
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].url < artifacts[j].url
	})

	printerConfig := &printers.TablePrinterConfig{
		Markdown: true,
	}

	p := printers.NewTablePrinter(printerConfig)

	printerConfig.ToHeaderAndRows = func(data any, wide bool) ([]string, [][]string, error) {
		p.MutateTable(func(table *tablewriter.Table) {
			table.SetAutoWrapText(false)
		})

		switch data.(type) {
		case []*artifact:
			var (
				header = []string{"IMAGE", "URL", "CHECKSUM", "PACKAGES"}
				rows   [][]string
			)

			for _, a := range artifacts {
				a := a

				url := fmt.Sprintf("[%s](%s)", path.Base(a.url), a.url)
				checksum := fmt.Sprintf("[%s](%s)", path.Base(a.checksumURL), a.checksumURL)
				packages := fmt.Sprintf("[%s](%s)", path.Base(a.packagesURL), a.packagesURL)

				rows = append(rows, []string{a.image, url, checksum, packages})
			}

			return header, rows, nil
		}

		return nil, nil, fmt.Errorf("unsupported type for printing: %T", data)
	}

	fmt.Println("## Downloads")

	return p.Print(artifacts)
}
