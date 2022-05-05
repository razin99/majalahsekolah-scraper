package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
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

func downloader(bookId string, page int, onNotOkResponse func(), wg *sync.WaitGroup) {
	defer wg.Done()

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

	var wg sync.WaitGroup

	foundLastPage := false
	endLoop := func() {
		foundLastPage = true
	}
	page := 1
	for !foundLastPage {
		wg.Add(1)
		go downloader(bookId, page, endLoop, &wg)
		page += 1
		// Prevent flooding the server with traffic
		time.Sleep(time.Millisecond * 200)
	}
	wg.Wait()
	fmt.Printf("Saved %d pages for book id: %s", page, bookId)
}
