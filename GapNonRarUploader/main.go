package main

import (
	"archive/tar"
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
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const Token = "" // FILL THIS
const MaxUploadSize = 500 * 1000 * 1000

type ReadLengthReporter struct {
	r            io.Reader
	readCallback func(read int)
}

func (r ReadLengthReporter) Read(b []byte) (int, error) {
	l, err := r.r.Read(b)
	if r.readCallback != nil {
		r.readCallback(l)
	}
	return l, err
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:")
		fmt.Println(filepath.Base(os.Args[0]) + " u/m filename")
		fmt.Println()
		fmt.Println("Usage of upload:")
		fmt.Println(filepath.Base(os.Args[0]) + " u [--parts 1,4,6,8-10] filename file1 file2")
		fmt.Println()
		fmt.Println("Usage of merge:")
		fmt.Println(filepath.Base(os.Args[0]) + " m file.ext")
		fmt.Println("Please do not include .1 for the file!")
		os.Exit(2)
	}
	if os.Args[1] == "u" {
		// Check --parts
		if len(os.Args) < 3 {
			fmt.Println("Please pass at least one file to upload")
			os.Exit(2)
		}
		// Check parts
		var parts map[int64]struct{}
		if os.Args[2] == "--parts" {
			if len(os.Args) < 5 {
				fmt.Println("Please pass at least one file to upload")
				os.Exit(2)
			}
			parts = parseParts(os.Args[3])
			Upload(os.Args[4], os.Args[5:], parts)
		} else {
			Upload(os.Args[2], os.Args[3:], parts)
		}
	} else if os.Args[1] == "m" {
		Merge(os.Args[2])
	} else {
		fmt.Println("Invalid mode:", os.Args[1])
		os.Exit(2)
	}
}

func Merge(filename string) {
	w, err := os.Create(filename)
	if err != nil {
		log.Fatal("Cannot write file:", err.Error())
	}

	for counter := 1; ; counter++ {
		ok := func() bool {
			if _, err := os.Stat(filename + "." + strconv.Itoa(counter)); os.IsNotExist(err) {
				return false
			}
			// read file to destination
			r, err := os.Open(filename + "." + strconv.Itoa(counter))
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

func Upload(filename string, files []string, parts map[int64]struct{}) {
	// Create a pipe to send data from tar to output
	tarPipeReader, tarPipeWriter := io.Pipe()
	totalRead := new(int64)
	done := new(uint32)
	tarWriter := tar.NewWriter(tarPipeWriter)
	// Get total file size
	totalFileSizes := getFileSizes(files)
	// Report progress
	go func(fileSize float64) {
		for {
			fmt.Printf("\r%.2f%%", float64(atomic.LoadInt64(totalRead))/fileSize*100)
			time.Sleep(time.Second)
		}
	}(float64(totalFileSizes))
	// Create the tar files
	go func() {
		for _, file := range files {
			tarFile(tarWriter, file, totalRead)
		}
		atomic.StoreUint32(done, 1)
		tarWriter.Close()
		tarPipeWriter.Close()
	}()
	// Upload the tar stream
	uploadStream(tarPipeReader, filename, done, parts)
}

func uploadStream(stream io.Reader, filename string, done *uint32, parts map[int64]struct{}) {
	// Create the link and checksum files
	linksFile, err := os.Create(filename + ".txt")
	if err != nil {
		log.Fatal("Cannot create link files:", err.Error())
	}
	defer linksFile.Close()
	checksumFile, err := os.Create(filename + ".md5")
	if err != nil {
		log.Fatal("Cannot create checksum file:", err.Error())
	}
	defer checksumFile.Close()
	// Some variables
	var client http.Client
	var partNumber int64
	checksum := md5.New()
	// Read until we reach the end of stream
	for atomic.LoadUint32(done) == 0 {
		partNumber++
		// Check if we don't need this part
		if _, exists := parts[partNumber]; !exists && len(parts) != 0 {
			// Just discard the input
			fmt.Println("\nSkipping part ", partNumber)
			io.Copy(io.Discard, io.LimitReader(stream, MaxUploadSize))
			continue
		}
		wg := new(sync.WaitGroup)   // The goroutine below must exit before we can check for done
		r, w := io.Pipe()           // Use pipe to reduce ram usage, and read and write simultaneously
		m := multipart.NewWriter(w) // post using multipart
		checksum.Reset()            // Use the same summer
		uploadedFilename := filename + ".tar." + strconv.FormatInt(partNumber, 10)
		wg.Add(1)
		go func() { // Write to pipe https://medium.com/@owlwalks/sending-big-file-with-minimal-memory-in-golang-8f3fc280d2c
			defer wg.Done()
			defer w.Close()
			defer m.Close()
			part, err := m.CreateFormFile("file", uploadedFilename)
			if err != nil {
				return
			}
			// now read file
			limitReader := io.LimitReader(stream, MaxUploadSize)
			// Also calculate the checksum while reading
			writer := io.MultiWriter(part, checksum)
			// Copy to checksum and output
			_, err = io.Copy(writer, limitReader)
			if err != nil {
				w.CloseWithError(err)
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
				fmt.Println("\nCannot write link to file. Here is the link:\n" + finalLink)
			}
			_, _ = fmt.Fprintf(checksumFile, "%x %s\n", checksum.Sum(nil), uploadedFilename)
		} else {
			log.Fatal("Cannot deserialize the web page json: Cannot find `path` in the json.")
		}
		// Wait for upload goroutine
		wg.Wait()
	}
}

func tarFile(w *tar.Writer, filename string, totalRead *int64) {
	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("\nCannot open file ", filename, ":", err)
		return
	}
	// Create the file in tar
	stat, _ := file.Stat()
	err = w.WriteHeader(&tar.Header{
		Name: filename,
		Size: stat.Size(),
	})
	if err != nil {
		fmt.Println("\nCannot write the tar header:", err)
		return
	}
	// Create the reader
	reportReader := &ReadLengthReporter{
		r: file,
		readCallback: func(read int) {
			atomic.AddInt64(totalRead, int64(read))
		},
	}
	// Copy
	_, err = io.Copy(w, reportReader)
	if err != nil {
		fmt.Println("\nCannot create the tar:", err)
		return
	}
}

func getFileSizes(files []string) int64 {
	var totalSize int64
	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			fmt.Println("Cannot open file ", file, ":", err)
			continue
		}
		totalSize += stat.Size()
	}
	return totalSize
}

func parseParts(argument string) map[int64]struct{} {
	splitData := strings.Split(argument, ",")
	result := make(map[int64]struct{}, len(splitData))
	for _, data := range splitData {
		if strings.Contains(data, "-") { // Range
			ranges := strings.Split(argument, "-")
			start, err := strconv.ParseInt(ranges[0], 10, 64)
			if err != nil {
				fmt.Printf("Invalid number on start of range %s: %s", data, err)
				continue
			}
			end, err := strconv.ParseInt(ranges[1], 10, 64)
			if err != nil {
				fmt.Printf("Invalid number on end of range %s: %s", data, err)
				continue
			}
			for ; start <= end; start++ {
				result[start] = struct{}{}
			}
		} else { // Number
			part, err := strconv.ParseInt(data, 10, 64)
			if err != nil {
				fmt.Printf("Invalid number on parts argument %s: %s", data, err)
				continue
			}
			result[part] = struct{}{}
		}
	}
	return result
}
