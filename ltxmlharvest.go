// Package ltxmlharvest provides a MathWebSearch harvester for documents outputted by latexml
package ltxmlharvest

import (
	"io"
	"io/fs"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
)

// HarvestFS recursively harvests all files in fs.FS.
// Each directory will be grouped into a single harvest.
func HarvestFS(fsys fs.FS, accept func(path string) bool, uri func(path string) string, writer func(path string, harvest Harvest) error, logger *log.Logger) {
	var wg sync.WaitGroup
	wg.Add(1)
	harvestFS(&wg, fsys, accept, ".", uri, writer, logger)
	wg.Wait()
}

func harvestFS(wg *sync.WaitGroup, fsys fs.FS, accept func(path string) bool, path string, uri func(path string) string, writer func(path string, harvest Harvest) error, logger *log.Logger) {
	defer wg.Done()

	logger.Printf("[scan    ] [start] %s\n", path)

	// read the directory
	dir, err := fs.ReadDir(fsys, path)
	if err != nil {
		logger.Printf("[scan    ] [status=%s] %s\n", err, path)
		return
	}

	// make new jobs
	jobs := make([]Job, 0, len(dir))
	for _, d := range dir {
		p := filepath.Join(path, d.Name())
		// if we have a directory, deal with it recursively
		if d.IsDir() {
			wg.Add(1)
			go harvestFS(wg, fsys, accept, p, uri, writer, logger)
			continue
		}

		if !accept(p) {
			continue
		}
		jobs = append(jobs, JobFromFile(fsys, p, uri(p)))
	}

	logger.Printf("[scan    ] [status=ok] [%d fragment(s)] %s\n", len(jobs), path)
	if len(jobs) == 0 {
		return
	}

	logger.Printf("[harvest ] [start] %s\n", path)
	harvest := HarvestFragments(jobs, logger)
	logger.Printf("[harvest ] [status=ok] %s\n", path)

	logger.Printf("[writer  ] [start] %s\n", path)
	defer func() {
		if err == nil {
			logger.Printf("[writer  ] [status=ok] %q\n", path)
		} else {
			logger.Printf("[writer  ] [status=%q] %s\n", err, path)
		}
	}()

	err = writer(path, harvest)

}

// HarvestFragments executes jobs and writes them to logger
func HarvestFragments(jobs []Job, logger *log.Logger) Harvest {

	// run all the jobs in seperate goroutines
	var wg sync.WaitGroup
	wg.Add(len(jobs))

	fragments := make(chan HarvestFragment)
	for n, job := range jobs {
		go job.Do(&wg, n, fragments, logger)
	}

	// once all the fragments have been sent, close the fragements channel;
	go func() {
		wg.Wait()
		close(fragments)
	}()

	// receive and sort results
	harvest := Harvest(make([]HarvestFragment, 0, len(jobs)))
	for f := range fragments {
		harvest = append(harvest, f)
	}
	sort.Sort(harvest)

	return harvest
}

// Job describes a job for the harvester
type Job struct {
	Reader func() (io.ReadCloser, error)
	URI    string
}

// JobFromFile creates a new Job from a file and a uribase
func JobFromFile(fsys fs.FS, path string, URI string) Job {
	return Job{
		Reader: func() (io.ReadCloser, error) {
			return fsys.Open(path)
		},
		URI: URI,
	}
}

func (job Job) Do(wg *sync.WaitGroup, n int, fragments chan<- HarvestFragment, logger *log.Logger) (err error) {
	defer wg.Done()

	var fragment HarvestFragment

	logger.Printf("[fragment] [status=start] %q\n", job.URI)
	defer func() {
		if err == nil {
			logger.Printf("[fragment] [status=ok] [%d formula(e)] %q\n", len(fragment.Formulae), job.URI)
		} else {
			logger.Printf("[fragment] [status=%q] %q\n", err, job.URI)
		}
	}()

	// open the job
	reader, err := job.Reader()
	if err != nil {
		return err
	}
	defer reader.Close()

	// read a fragment
	if _, err := fragment.ReadFrom(reader); err != nil {
		return err
	}

	// setup ID and URI
	fragment.ID = strconv.Itoa(n)
	fragment.URI = job.URI

	// and send it to the channel
	fragments <- fragment
	return
}

// HarvestReader harvests a single reader and writes the output to writer
func HarvestReader(reader io.Reader, URI string, writer io.WriteCloser) error {
	// read a fragment
	var fragment HarvestFragment
	if _, err := fragment.ReadFrom(reader); err != nil {
		return err
	}

	// setup ID and URI
	fragment.ID = strconv.Itoa(1)
	fragment.URI = URI

	// and write the harvest to the output
	_, err := Harvest([]HarvestFragment{fragment}).WriteTo(writer)
	return err
}
