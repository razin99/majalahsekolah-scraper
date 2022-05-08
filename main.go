package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"sync"

	pdfcpuApi "github.com/pdfcpu/pdfcpu/pkg/api"
	"golang.org/x/sync/semaphore"
)

const URL = "https://majalahsekolah.com/books/%s/files/large/%d.png"

// Ensure error is nil
func must[K any](val K, err error) K {
	if err != nil {
		log.Fatal(err)
	}
	return val
}

func saveToDisk(bookId string, page int, blob io.Reader) error {
	file := fmt.Sprintf("%d.png", page)
	out, err := os.Create(path.Join(bookId, file))
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, blob)
	if err != nil {
		return err
	}
	return nil
}

func downloader(bookId string, page int) bool {
	if _, err := os.Stat(path.Join(bookId, fmt.Sprintf("%d.png", page))); err == nil {
		return true
	}
	url := fmt.Sprintf(URL, bookId, page)
	res := must(http.Get(url))
	if res.StatusCode != 200 {
		return false
	}
	must(saveToDisk(bookId, page, res.Body), nil)
	return true
}

func downloadAll(bookId string) error {
	maxWorkers := int64(runtime.NumCPU())
	sem := semaphore.NewWeighted(maxWorkers)
	ctx := context.Background()
	var lock sync.Mutex
	log.Println("Number of workers: ", maxWorkers)

	page := 1
	hasNextPage := true
	for hasNextPage {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}
		go func() {
			defer sem.Release(1)

			lock.Lock()
			if !hasNextPage {
				lock.Unlock()
				return
			}
			lock.Unlock()

			if downloadOk := downloader(bookId, page); !downloadOk {
				lock.Lock()
				hasNextPage = false
				lock.Unlock()
				return
			}
			// make sure this is atomic
			lock.Lock()
			page += 1
			lock.Unlock()
			log.Printf("Downloaded  [book:%s][page:%d]\n", bookId, page)
		}()
	}
	log.Printf("Found %d pages for book id: %s\n", page-1, bookId)

	// Wait for all remaining workers to exit
	sem.Acquire(ctx, maxWorkers)
	sem.Release(maxWorkers)

	log.Printf("Saved %d pages for book id: %s\n", page-1, bookId)

	return nil
}

func main() {
	noPdfFlagPtr := flag.Bool("no-pdf", false, "if set, pdf would not be generated")
	pdfGenWorkersPtr := flag.Int(
		"pdf-workers",
		runtime.NumCPU(),
		"number of processes that handles PDF generation, reduce if there's not enough memory",
	)
	flag.Parse()
	noPdfFlag := *noPdfFlagPtr
	pdfWorkers := *pdfGenWorkersPtr

	if pdfWorkers <= 0 {
		log.Fatal("pdf-workers must be greater than 0")
	}

	bookId := flag.Arg(0)
	if len(bookId) != 4 {
		log.Fatal("Book Id must be exactly 4 characters")
	}

	// drwxr-xr-x
	if err := os.Mkdir(bookId, 0755); errors.Is(err, os.ErrExist) {
		log.Printf("Folder %s already exists\n", bookId)
	} else if err != nil {
		log.Fatal(err)
	}
	must("", downloadAll(bookId))

	if noPdfFlag {
		return
	}

	files := must(os.ReadDir(bookId))
	chunkSize := 25
	fileChunks := [][]string{}
	for i := 0; i < len(files); i += chunkSize {
		end := i + chunkSize
		if end > len(files) {
			end = len(files)
		}
		fileList := []string{}
		for j := i; j < end; j++ {
			fileList = append(fileList, path.Join(bookId, fmt.Sprintf("%d.png", j+1)))
		}
		fileChunks = append(fileChunks, fileList)
	}

	tmpDir := must(ioutil.TempDir(bookId, "pdfgen"))
	defer func() {
		log.Println("Clean up temporary files")
		os.RemoveAll(tmpDir)
	}()

	maxWorkers := int64(pdfWorkers)
	sem := semaphore.NewWeighted(maxWorkers)
	ctx := context.Background()

	tmpPdfs := []string{}
	for id, chunk := range fileChunks {
		must("", sem.Acquire(ctx, 1))
		outPath := path.Join(tmpDir, fmt.Sprintf("%d.pdf", id))
		go func(id int, chunk []string) {
			defer sem.Release(1)
			head := chunk[0]
			tail := chunk[len(chunk)-1]
			log.Printf("Processing chunk %d [%s - %s]\n", id, head, tail)
			must("", pdfcpuApi.ImportImagesFile(chunk, outPath, nil, nil))
			log.Printf("Completed chunk %d [%s - %s]\n", id, head, tail)
		}(id, chunk)
		tmpPdfs = append(tmpPdfs, outPath)
	}
	must("", sem.Acquire(ctx, maxWorkers))

	outPath := path.Join(bookId, "output.pdf")
	log.Printf("Merging %s\n", bookId)
	must("", pdfcpuApi.MergeCreateFile(tmpPdfs, outPath, nil))
	log.Printf("Generated pdf at: %s\n", outPath)
}
