package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("By Hirbod Behnam")
		fmt.Println("Usage: ./app filename.txt")
		return
	}
	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		for i := 0; i < 2; i++ {
			s := strings.Split(scanner.Text(), ":")
			resp, err := http.Head("https://" + s[0])
			if err != nil {
				log.Println("Error on header:", err.Error())
				break
			}
			contentDisposition := resp.Header.Get("Content-Disposition")
			if len(contentDisposition) < 20 {
				log.Println("Error on getting the header(Short header)")
				if i == 0 {
					log.Println("Trying once more")
				}
				continue
			}
			err = os.Rename(contentDisposition[20:], s[1])
			if err != nil {
				log.Println("Error on rename:", err.Error())
			}
			break
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
