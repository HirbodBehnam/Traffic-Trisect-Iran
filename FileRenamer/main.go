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
	if len(os.Args) < 3{
		fmt.Println("By Hirbod Behnam")
		fmt.Println("Usage: ./app TOKEN filename.txt")
		return
	}
	file, err := os.Open(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s := strings.Split(scanner.Text(),":")
		link := "https://bot.sapp.ir/"+ os.Args[1] +"/downloadFile/" + s[0]
		resp, err := http.Head(link)
		if err != nil{
			log.Println("Error on header:",err.Error())
			continue
		}
		contentDisposition := resp.Header.Get("Content-Disposition")
		err = os.Rename(contentDisposition[20:],s[1])
		if err != nil{
			log.Println("Error on rename:",err.Error())
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
