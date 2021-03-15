package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/MathWebSearch/ltxmlharvest"
)

func main() {
	logger := log.Default()

	logger.Printf("scanPath=%s\n", scanPath)
	logger.Printf("scanExt=%s\n", scanExt)
	logger.Printf("uriBase=%s\n", uriBase)
	logger.Printf("outPath=%s\n", outPath)

	fsys := os.DirFS(scanPath)
	accept := func(path string) bool {
		return strings.HasSuffix(path, "."+scanExt)
	}

	base, err := url.Parse(uriBase)
	if err != nil {
		log.Fatal(err)
	}

	uri := func(p string) string {
		copy := *base
		copy.Path = path.Join(copy.Path, p)
		return copy.String()
	}

	writer := func(path string, harvest ltxmlharvest.Harvest) error {
		out := strings.ReplaceAll(strings.TrimPrefix(path, "."), string(os.PathSeparator), "_") + ".harvest"
		out = filepath.Join(outPath, out)

		f, err := os.Create(out)
		if err != nil {
			return err
		}

		_, err = harvest.WriteTo(f)
		if err == nil {
			logger.Printf("[writer] wrote %s\n", out)
		}
		return err
	}

	ltxmlharvest.HarvestFS(fsys, accept, uri, writer, logger)
}

var scanPath string
var scanExt string

var uriBase string
var outPath string

func joinurl(base string, p string) string {
	u, err := url.Parse(base)
	if err != nil { // ugly fallback, but it'll work
		return base + p
	}
	u.Path = path.Join(u.Path, p)
	return u.String()
}

func init() {
	defer flag.Parse()

	flag.StringVar(&scanPath, "root", ".", "Directory to start scanning at")
	flag.StringVar(&scanExt, "ext", "xhtml", "File Extension to read input from")
	flag.StringVar(&outPath, "out", "", "Output path, each directory will correspond to a single '.harvest' file")
	flag.StringVar(&uriBase, "uri", "file://./", "URI base to use for harvest ouput")

}
