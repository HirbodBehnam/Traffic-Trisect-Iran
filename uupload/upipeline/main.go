package main

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"sync"

	"github.com/schollz/progressbar/v3"
)

// Number of threads which upload files to uupload
const uploadThreads = 5
const uploadHash = ""
const fileTTL = "259200"

// The client which we do our uploads with
var httpClient = &http.Client{}

// The regex which we can use to extract the download link from the page result
var resultRegex = regexp.MustCompile(`<img src="css/images/file.png" style="margin-bottom:4px;" alt="([0-9a-zA-Z_.\-]+)" />`)

func main() {
	// Check arguments
	if len(os.Args) <= 1 {
		fmt.Println("Please pass the filenames as the arguments")
		os.Exit(1)
	}
	// Open each file
	var totalUploadSize int64
	toUploadFiles := make([]*os.File, len(os.Args)-1)
	for i := 1; i < len(os.Args); i++ {
		var err error
		toUploadFiles[i-1], err = os.Open(os.Args[i])
		if err != nil {
			fmt.Printf("cannot open file %s: %v\n", os.Args[i], err)
			os.Exit(1)
		}
		// Caclualte the upload size
		stat, err := toUploadFiles[i-1].Stat()
		if err != nil {
			fmt.Printf("cannot get file status of %s: %v\n", os.Args[i], err)
			os.Exit(1)
		}
		totalUploadSize += stat.Size()
	}

	// Create the progress bar and uploader goroutine
	bar := progressbar.DefaultBytes(totalUploadSize, "Uploading")
	wg := new(sync.WaitGroup)
	wg.Add(uploadThreads)
	toUploadFilesChannel := make(chan *os.File)
	for range uploadThreads {
		go uploaderThread(toUploadFilesChannel, bar, wg)
	}
	// Schedule each file to a single goroutine
	for _, toUploadFile := range toUploadFiles {
		toUploadFilesChannel <- toUploadFile
	}
	close(toUploadFilesChannel) // signal the threads which we are done
	wg.Wait()                   // Wait for upload to finish
	// Done?
}

func uploaderThread(toUploadFiles <-chan *os.File, bar *progressbar.ProgressBar, wg *sync.WaitGroup) {
	defer wg.Done() // Finish the wait group when the goroutine finishes
	// For each file...
	for toUploadFile := range toUploadFiles {
		uploadFile(toUploadFile, bar)
	}
}

func uploadFile(file *os.File, bar *progressbar.ProgressBar) {
	defer file.Close()
	// Create a pipe to reduce memory usage
	body, writer := io.Pipe()
	// Create the request
	req, err := http.NewRequest(http.MethodPost, "https://s6.uupload.ir/sv_process.php", body)
	if err != nil {
		fmt.Println("\nrequest creation failed:", err)
		return
	}
	mwriter := multipart.NewWriter(writer)
	req.Header.Set("Content-Type", mwriter.FormDataContentType())
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:139.0) Gecko/20100101 Firefox/139.0")
	req.Header.Set("Origin", "https://uupload.ir")
	req.Header.Set("Referer", "https://uupload.ir/")
	// Upload the file and report progress
	var copyErr error
	go func() {
		// Add the hash to form data
		{
			hashWriter, _ := mwriter.CreateFormField("hash")
			hashWriter.Write([]byte(uploadHash))
		}
		// Add the TTL
		{
			ttlWriter, _ := mwriter.CreateFormField("ittl")
			ttlWriter.Write([]byte(fileTTL))
		}
		// Add the file and upload it
		fileWriter, _ := mwriter.CreateFormFile("__userfile[]", file.Name())
		_, copyErr = io.Copy(io.MultiWriter(fileWriter, bar), file)
		mwriter.Close() // This will finalize the request
		writer.Close()  // This will finish off the request
	}()
	// Fire the request
	resp, err := httpClient.Do(req)
	if err != nil || copyErr != nil {
		fmt.Printf("\ncannot fire the request: %v with copyErr: %v", err, copyErr)
		return
	}
	// Read the response
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("\ncannot read the body of the response:", err)
		return
	}
	matches := resultRegex.FindSubmatch(respBody)
	fmt.Printf("\n%s uploaded at https://uupload.ir/view/%s\n", file.Name(), string(matches[1]))
}
