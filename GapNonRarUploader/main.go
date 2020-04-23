package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

const TOKEN = "" // FILL THIS
const ReadBuffer = 32 * 1024
const NumberOfParts = 100 * 1000 / ReadBuffer // We read the file 32kb at once. Max file size is 500MB. This is the max number of parts for each file

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

	counter := 0
	for {
		counter++
		if _, err := os.Stat(FileName + "." + strconv.Itoa(counter)); os.IsNotExist(err) {
			break
		}
		// read file to destination
		r, err := os.Open(FileName + "." + strconv.Itoa(counter))
		if err != nil {
			log.Println("Cannot open file for reading:", err.Error())
			break
		}

		_, err = io.Copy(w, r)
		if err != nil {
			log.Println("Cannot copy file:", err.Error())
			break
		}
	}
}

func Upload() {
	f, err := os.Open(FileName)
	if err != nil {
		log.Fatal("Cannot read file:", err.Error())
	}

	partNumber := 0
	for doneUploading := false; !doneUploading; {
		partNumber++
		r, w := io.Pipe()           // Use pipe to reduce ram usage, and read and write simultaneously
		m := multipart.NewWriter(w) // post using multipart
		go func() {                 //Write to pipe https://medium.com/@owlwalks/sending-big-file-with-minimal-memory-in-golang-8f3fc280d2c
			defer w.Close()
			defer m.Close()
			part, err := m.CreateFormFile("file", FileName+"."+strconv.Itoa(partNumber))
			if err != nil {
				return
			}
			// now read file
			buffer := make([]byte, ReadBuffer)
			for i := 0; i < NumberOfParts; i++ { // TODO: LimitReader maybe?
				count, rError := f.Read(buffer)
				if rError != nil {
					if rError == io.EOF {
						if i == 0 { // this means that the file is already read
							os.Exit(0)
						}
						doneUploading = true
						break
					}
					log.Fatal("Cannot read file:", rError.Error())
				}
				_, wError := part.Write(buffer[:count])
				if wError != nil {
					break
				}
			}
		}()
		// Initialize uploader
		req, err := http.NewRequest("POST", "https://api.gap.im/upload", r)
		if err != nil {
			log.Fatal("Initialize uploader:", err.Error())
		}
		req.Header.Set("Content-Type", m.FormDataContentType())
		req.Header.Add("token", TOKEN) // Add the gap token
		// Submit the request
		var client = &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal("Cannot upload file (client.Do):", err.Error())
		}
		if resp.StatusCode != http.StatusOK { //In Gap 403 means invalid token; 500 invalid file type or big file. 405 means that their server is fucked
			log.Fatal("HTTP status is not ok. It is:", resp.StatusCode)
		}
		body := &bytes.Buffer{}
		_, err = body.ReadFrom(resp.Body)
		if err != nil {
			log.Fatal("Cannot read body:", err.Error())
		}
		_ = resp.Body.Close()
		//Try to deserialize json
		readBuf, err := ioutil.ReadAll(body)
		if err != nil {
			log.Fatal("Cannot read body(ioutil.ReadAll):", err.Error())
		}
		var jsonRes map[string]interface{}
		err = json.Unmarshal(readBuf, &jsonRes)
		if err != nil {
			log.Fatal("Cannot deserialize the web page json:", err.Error())
		}
		if finalLink, ok := jsonRes["path"]; ok {
			fmt.Println(finalLink.(string))
		} else {
			log.Fatal("Cannot deserialize the web page json: Cannot find `path` in the json. Json is:\n" + string(readBuf))
		}
	}
}
