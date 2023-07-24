package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/schollz/progressbar/v3"
)

// WriteCounter counts the number of bytes written to it. It implements to the io.Writer interface
// and we can pass this into io.TeeReader() which will report progress on each write cycle.
type WriteCounter struct {
	Total       uint64
	LastCounter uint64
	StartTime   time.Time
	Progress    *progressbar.ProgressBar
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.Progress.Add(n)
	wc.PrintProgress()
	return n, nil
}

func (wc *WriteCounter) PrintProgress() {
	elapsed := time.Since(wc.StartTime).Seconds()
	downloaded := wc.Total - wc.LastCounter
	downloadSpeed := float64(downloaded) / elapsed

	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	wc.Progress.Clear()

	// Return again and print current status of download with download speed
	wc.Progress.Describe(fmt.Sprintf("Downloading... %s complete | Speed: %s/s", humanize.Bytes(wc.Total), humanize.Bytes(uint64(downloadSpeed))))

	wc.LastCounter = wc.Total
}

func main() {
	fmt.Println("Download Started")

	fileUrl := "https://upload.wikimedia.org/wikipedia/commons/e/e7/Everest_North_Face_toward_Base_Camp_Tibet_Luca_Galuzzi_2006.jpg"
	err := DownloadFile("mountain.jpg", fileUrl)
	if err != nil {
		panic(err)
	}

	fmt.Println("Download Finished")
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory. We pass an io.TeeReader
// into Copy() to report progress on the download.
func DownloadFile(filepath string, url string) error {

	// Create the file, but give it a tmp file extension, this means we won't overwrite a
	// file until it's downloaded, but we'll remove the tmp extension once downloaded.
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		out.Close()
		return err
	}
	defer resp.Body.Close()

	// Create our progress reporter and pass it to be used alongside our writer
	fileSize, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
	counter := &WriteCounter{
		Progress: progressbar.NewOptions(
			fileSize,
			progressbar.OptionSetDescription("Downloading..."),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionShowCount(),
			progressbar.OptionSetWidth(15),
		),
	}

	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		out.Close()
		return err
	}

	// The progress use the same line so print a new line once it's finished downloading
	fmt.Print("\n")

	// Close the file without defer so it can happen before Rename()
	out.Close()

	if err = os.Rename(filepath+".tmp", filepath); err != nil {
		return err
	}
	return nil
}
