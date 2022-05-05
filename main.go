package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"sync"

	"golang.org/x/sync/semaphore"
)

const URL = "https://majalahsekolah.com/books/%s/files/large/%d.png"

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

func downloader(bookId string, page int, onNotOkResponse func()) {
	url := fmt.Sprintf(URL, bookId, page)
	res, err := http.Get(url)

	if err != nil {
		log.Fatal(err)
	}
	if res.StatusCode != 200 {
		onNotOkResponse()
		return
	}

	if err := saveToDisk(bookId, page, res.Body); err != nil {
		log.Fatal(err)
	}
}

func main() {
	bookId := os.Args[1]
	if len(bookId) != 4 {
		log.Fatal("Book Id must be exactly 4 characters")
	}

	// drwxr-xr-x
	if err := os.Mkdir(bookId, 0755); err != nil {
		log.Fatal(err)
	}

	maxWorkers := int64(runtime.NumCPU())
	sem := semaphore.NewWeighted(maxWorkers)
	ctx := context.Background()
	var lock sync.Mutex
	log.Println("Number of workers: ", maxWorkers)

	foundLastPage := false
	endLoop := func() {
		foundLastPage = true
	}
	page := 0
	for !foundLastPage {
		if err := sem.Acquire(ctx, 1); err != nil {
			log.Fatal(err)
		}
		go func() {
			// make sure this is atomic
			lock.Lock()
			page += 1
			lock.Unlock()

			downloader(bookId, page, endLoop)
			log.Printf("Downloaded  [book:%s][page:%d]\n", bookId, page)
			sem.Release(1)
		}()
	}
	log.Printf("Found %d pages for book id: %s\n", page, bookId)

	// Wait for all remaining workers to exit
	sem.Acquire(ctx, maxWorkers)
	sem.Release(maxWorkers)

	log.Printf("Saved %d pages for book id: %s\n", page, bookId)
}
