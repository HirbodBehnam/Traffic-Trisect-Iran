package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"
)

const Token = "" // FILL THIS
const MaxUploadSize = 500 * 1000 * 1000

type ReadLengthReporter struct {
	r         io.Reader
	totalRead uint64
}

func (r *ReadLengthReporter) Read(b []byte) (int, error) {
	l, err := r.r.Read(b)
	atomic.AddUint64(&r.totalRead, uint64(l))
	return l, err
}

var FileName string

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:")
		fmt.Println(filepath.Base(os.Args[0]) + " u/m filename")
		fmt.Println()
		fmt.Println("Usage of upload:")
		fmt.Println(filepath.Base(os.Args[0]) + " u file.ext")
		fmt.Println()
		fmt.Println("Usage of merge:")
		fmt.Println(filepath.Base(os.Args[0]) + " m file.ext")
		fmt.Println("Please do not include .1 for the file!")
		os.Exit(2)
	}
	FileName = os.Args[2]
	if os.Args[1] == "u" {
		Upload()
	} else if os.Args[1] == "m" {
		Merge()
	} else {
		fmt.Println("Invalid mode:", os.Args[1])
		os.Exit(2)
	}
}

func Merge() {
	w, err := os.Create(FileName)
	if err != nil {
		log.Fatal("Cannot write file:", err.Error())
	}

	for counter := 1; ; counter++ {
		ok := func() bool {
			if _, err := os.Stat(FileName + "." + strconv.Itoa(counter)); os.IsNotExist(err) {
				return false
			}
			// read file to destination
			r, err := os.Open(FileName + "." + strconv.Itoa(counter))
			if err != nil {
				log.Println("\nCannot open file for reading:", err.Error())
				return false
			}
			fmt.Printf("\rMerging file number %d", counter)
			defer r.Close()

			_, err = io.Copy(w, r)
			if err != nil {
				log.Println("\nCannot copy file:", err.Error())
				return false
			}
			return true
		}()
		if !ok {
			break
		}
	}

	w.Close() // no need for defer we always reach here
}

func Upload() {
	linksFile, err := os.Create(FileName + ".txt")
	if err != nil {
		log.Fatal("Cannot create link files:", err.Error())
	}
	checksumFile, err := os.Create(FileName + ".md5")
	if err != nil {
		log.Fatal("Cannot create checksum file:", err.Error())
	}
	source, err := os.Open(FileName)
	if err != nil {
		log.Fatal("Cannot read file:", err.Error())
	}
	sourceStat, _ := source.Stat()

	var client http.Client
	reportReader := &ReadLengthReporter{
		r: source,
	}
	partNumber := int64(0)
	checksum := md5.New()
	// report progress
	go func(fileSize float64) {
		for {
			fmt.Printf("\r%.2f%%", float64(atomic.LoadUint64(&reportReader.totalRead))/fileSize*100)
			time.Sleep(time.Second)
		}
	}(float64(sourceStat.Size()))
	for totalParts := ceil(sourceStat.Size(), MaxUploadSize); partNumber < totalParts; partNumber++ {
		r, w := io.Pipe()           // Use pipe to reduce ram usage, and read and write simultaneously
		m := multipart.NewWriter(w) // post using multipart
		checksum.Reset()
		uploadedFilename := FileName + "." + strconv.FormatInt(partNumber, 10)
		go func() { // Write to pipe https://medium.com/@owlwalks/sending-big-file-with-minimal-memory-in-golang-8f3fc280d2c
			defer w.Close()
			defer m.Close()
			part, err := m.CreateFormFile("file", uploadedFilename)
			if err != nil {
				return
			}
			// now read file
			limitReader := io.LimitReader(source, MaxUploadSize)
			// Also calculate the checksum while reading
			writer := io.MultiWriter(part, checksum)
			// Copy to checksum and output
			_, err = io.Copy(writer, limitReader)
			if err != nil {
				_ = w.CloseWithError(err)
			}
		}()
		// Initialize uploader
		req, err := http.NewRequest("POST", "https://api.gap.im/upload", r)
		if err != nil {
			log.Fatal("Initialize uploader:", err.Error())
		}
		req.Header.Set("Content-Type", m.FormDataContentType())
		req.Header.Add("token", Token) // Add the gap token
		// Submit the request
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal("Cannot upload file (client.Do):", err.Error())
		}
		if resp.StatusCode != http.StatusOK { // In Gap 403 means invalid token; 500 invalid file type or big file. 405 means that their server is fucked
			log.Fatal("HTTP status is not ok. It is:", resp.StatusCode)
		}
		// Try to deserialize json
		var jsonRes map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&jsonRes)
		_ = resp.Body.Close()
		if err != nil {
			log.Fatal("Cannot deserialize the web page json:", err.Error())
		}
		if finalLink, ok := jsonRes["path"].(string); ok {
			_, err = linksFile.WriteString(finalLink + "\n")
			if err != nil {
				fmt.Println()
				fmt.Println("Cannot write link to file. Here is the link:")
				fmt.Println(finalLink)
			}
			_, _ = fmt.Fprintf(checksumFile, "%x %s\n", checksum.Sum(nil), uploadedFilename)
		} else {
			log.Fatal("Cannot deserialize the web page json: Cannot find `path` in the json.")
		}
	}
}

func ceil(a, b int64) int64 {
	return (a + b - 1) / b
}
