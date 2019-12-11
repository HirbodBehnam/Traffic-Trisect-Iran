package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dustin/go-humanize"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"io"
	"io/ioutil"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"
)

type ConfigJson struct {
	GapToken      string
	TelegramToken string
	Admins        []int
	MaxFileSize   int
}

var Config ConfigJson
var Downloads = make(map[int]bool) //True is downloading, false is canceled
const VERSION = "1.0.0 / Build 1"

func main() {

	{ //Parse argument
		cnf := "config.json"
		if len(os.Args) > 1 {
			cnf = os.Args[1]
		}

		confF, err := ioutil.ReadFile(cnf)
		if err != nil {
			panic("Cannot read the config file. (io Error) " + err.Error())
		}

		err = json.Unmarshal(confF, &Config)
		if err != nil {
			panic("Cannot read the config file. (Parse Error) " + err.Error())
		}
	}

	log.Println("Gap uploader bot By Hirbod Behnam")
	log.Println("Version", VERSION)
	bot, err := tgbotapi.NewBotAPI(Config.TelegramToken)
	if err != nil {
		panic("Cannot initialize the bot: " + err.Error())
	}
	log.Printf("Bot authorized on account %s", bot.Self.UserName)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil && update.CallbackQuery == nil {
			continue
		}
		//Check query on cancel points
		if update.CallbackQuery != nil && update.CallbackQuery.Data == "Cancel" {
			if _, ok := Downloads[update.CallbackQuery.Message.MessageID]; ok {
				Downloads[update.CallbackQuery.Message.MessageID] = false
			}
			continue
		}
		//Check commands
		if update.Message.IsCommand() {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			switch update.Message.Command() {
			case "start":
				msg.Text = "Welcome to GapUploader bot.\nYou can use this bot to upload your data to Gap servers to download them at lower costs.\nJust send me the link of the file."
			case "id":
				msg.Text = strconv.FormatInt(int64(update.Message.From.ID), 10)
			case "about":
				msg.Text = "A simple bot by Hirbod Behnam\nSource at https://github.com/HirbodBehnam/Traffic-Trisect-Iran"
			default:
				msg.Text = "I don't know that command"
			}
			_, _ = bot.Send(msg)
			continue
		}

		//Next steps requires the user to be admin
		if checkAdmin(update.Message.From.ID) {
			go func(lUpdate tgbotapi.Update) {
				//At first send a message to user
				msg := tgbotapi.NewMessage(lUpdate.Message.Chat.ID, "Getting info about the file...")
				inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Cancel", "Cancel")))
				msg.ReplyToMessageID = lUpdate.Message.MessageID
				msg.ReplyMarkup = inlineKeyboard
				SentMessage, err := bot.Send(msg)
				if err != nil {
					log.Println("Error sending message:", err)
					return
				}
				//Get the file length
				resp, err := http.Head(lUpdate.Message.Text)
				if err != nil {
					edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Error on getting file size: "+err.Error())
					_, _ = bot.Send(edited)
					return
				}
				if resp.StatusCode != http.StatusOK {
					edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Error on getting file size: Page returned code "+strconv.FormatInt(int64(resp.StatusCode), 10))
					_, _ = bot.Send(edited)
					return
				}
				downloadSize, err := strconv.Atoi(resp.Header.Get("Content-Length"))
				if err != nil {
					edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Error on getting file size: "+err.Error()+"\nThe bot might have failed to get Content-Length or maybe the web server does not support it.")
					_, _ = bot.Send(edited)
					return
				}
				if downloadSize > Config.MaxFileSize {
					edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "The file you are going to download is too big! The server limit is "+humanize.Bytes(uint64(Config.MaxFileSize))+" however the file you requested is "+humanize.Bytes(uint64(downloadSize))+".")
					_, _ = bot.Send(edited)
					return
				}
				//Now download the file
				file, err := ioutil.TempFile("", "*.tmp") //Create a temp file, it will be renamed later
				if err != nil {
					edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Error on creating a temp file: "+err.Error())
					_, _ = bot.Send(edited)
					log.Println("Error on creating a temp file:", err.Error())
					return
				}
				defer file.Close()
				defer os.Remove(file.Name())

				resp, err = http.Get(lUpdate.Message.Text)
				if err != nil {
					edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Error on downloading file: "+err.Error())
					_, _ = bot.Send(edited)
					return
				}

				Downloads[SentMessage.MessageID] = true
				writtenInSecond := 0
				done := make(chan int64)
				{
					go func() { //Report download
						var percent float64
						for {
							select {
							case <-done:
								return
							default:
								if err != nil {
									log.Fatal(err)
								}

								fi, err := file.Stat()
								if err != nil {
									log.Fatal(err)
								}

								size := fi.Size()

								if size == 0 {
									size = 1
								}

								percent = float64(size) / float64(downloadSize) * 100

								progressbar := "["
								tempPercent := math.Floor(percent / 10)
								for i := 0; i < int(tempPercent); i++ {
									progressbar += "#"
								}
								progressbar += ">"
								tempPercent = 9 - tempPercent
								for i := 0; i < int(tempPercent); i++ {
									progressbar += "⠀" // This is not space! https://www.compart.com/en/unicode/U+2800
								}
								progressbar += "]"

								edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Downloading:\n"+humanize.Bytes(uint64(size))+" from "+humanize.Bytes(uint64(downloadSize))+"\n"+progressbar+"\nSpeed: "+humanize.Bytes(uint64(writtenInSecond))+"/s")
								writtenInSecond = 0
								edited.ReplyMarkup = &inlineKeyboard
								_, _ = bot.Send(edited)
							}
							time.Sleep(time.Second)
						}
					}()
					{
						//This part is mostly like io.copy
						buf := make([]byte, 32768)
						for {
							if !Downloads[SentMessage.MessageID] {
								done <- 0 //Terminate download statics
								edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Canceled")
								_, _ = bot.Send(edited)
								delete(Downloads, SentMessage.MessageID)
								return
							}
							nr, er := resp.Body.Read(buf)
							if nr > 0 {
								nw, ew := file.Write(buf[0:nr])
								if nw > 0 {
									writtenInSecond += nw
								}
								if ew != nil {
									err = ew
									break
								}
								if nr != nw {
									err = io.ErrShortWrite
									break
								}
							}
							if er != nil {
								if er != io.EOF {
									err = er
								}
								break
							}
						}
					}
					done <- 0 //Terminate download statics
					if err != nil {
						edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Error on downloading file: "+err.Error())
						_, _ = bot.Send(edited)
						return
					}
				}
				edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Done downloading the file.\nPreparing to upload the file...")
				edited.ReplyMarkup = &inlineKeyboard
				_, _ = bot.Send(edited)
				//Now upload the file
				uploaded := 0       //Track uploaded bytes to report process
				writtenInSecond = 0 //Also track the upload speed
				r, w := io.Pipe()   //Use pipe to reduce ram usage
				m := multipart.NewWriter(w)
				go func() { //Write to pipe https://medium.com/@owlwalks/sending-big-file-with-minimal-memory-in-golang-8f3fc280d2c
					defer w.Close()
					defer m.Close()
					part, err := m.CreateFormFile("file", getFileName(lUpdate.Message.Text))
					if err != nil {
						return
					}

					//IDK why but I cannot use file.Read :|
					file1, err := os.Open(file.Name())
					if err != nil {
						return
					}
					defer file1.Close()

					buf := make([]byte, 32768)
					for {
						if !Downloads[SentMessage.MessageID] {
							done <- 0
							edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Canceled")
							_, _ = bot.Send(edited)
							delete(Downloads, SentMessage.MessageID)
							return
						}
						nr, er := file1.Read(buf)
						if nr > 0 {
							nw, ew := part.Write(buf[0:nr])
							if nw > 0 {
								uploaded += nw
								writtenInSecond += nw
							}
							if ew != nil {
								err = ew
								break
							}
							if nr != nw {
								err = io.ErrShortWrite
								break
							}
						}
						if er != nil {
							if er != io.EOF {
								err = er
							}
							break
						}
					}
					if err != nil {
						fmt.Println(err)
					}
				}()

				go func() { //Report process
					var percent float64
					for {
						select {
						case <-done:
							return
						default:
							percent = float64(uploaded) / float64(downloadSize) * 100

							progressbar := "["
							tempPercent := math.Floor(percent / 10)
							for i := 0; i < int(tempPercent); i++ {
								progressbar += "#"
							}
							progressbar += ">"
							tempPercent = 9 - tempPercent
							for i := 0; i < int(tempPercent); i++ {
								progressbar += "⠀" // This is not space! https://www.compart.com/en/unicode/U+2800
							}
							progressbar += "]"

							edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Uploading:\n"+humanize.Bytes(uint64(uploaded))+" from "+humanize.Bytes(uint64(downloadSize))+"\n"+progressbar+"\nSpeed: "+humanize.Bytes(uint64(writtenInSecond))+"/s")
							writtenInSecond = 0
							edited.ReplyMarkup = &inlineKeyboard
							_, _ = bot.Send(edited)
						}
						time.Sleep(time.Second)
					}
				}()
				req, err := http.NewRequest("POST", "https://api.gap.im/upload", r)
				if err != nil {
					edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Error on initializing upload (request): "+err.Error())
					_, _ = bot.Send(edited)
					return
				}

				req.Header.Set("Content-Type", m.FormDataContentType())
				req.Header.Add("token", Config.GapToken)

				// Submit the request
				var client = &http.Client{}
				resp, err = client.Do(req)
				if !Downloads[SentMessage.MessageID] { //Check if the process has been terminated by user
					return
				}
				done <- 0 //Terminate the status reporter if it wasn't canceled
				if err != nil {
					edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Error on uploading file: "+err.Error())
					_, _ = bot.Send(edited)
					return
				}
				if resp.StatusCode != http.StatusOK { //In Gap 403 means invalid token; 500 invalid file type or big file. 405 means that their server is fucked
					edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Error on uploading file: The web page returned status code "+strconv.FormatInt(int64(resp.StatusCode), 10))
					_, _ = bot.Send(edited)
					return
				}
				body := &bytes.Buffer{}
				_, err = body.ReadFrom(resp.Body)
				if err != nil {
					edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Cannot read page body: "+err.Error())
					_, _ = bot.Send(edited)
					return
				}
				_ = resp.Body.Close()
				//Try to deserialize json
				readBuf, err := ioutil.ReadAll(body)
				if err != nil {
					edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Cannot read page body: "+err.Error())
					_, _ = bot.Send(edited)
					return
				}
				var jsonRes map[string]interface{}
				err = json.Unmarshal(readBuf, &jsonRes)
				if err != nil {
					edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Cannot deserialize the web page json: "+err.Error()+"\n\nThe page returned:\n"+string(readBuf))
					_, _ = bot.Send(edited)
					return
				}
				if finalLink, ok := jsonRes["path"].(string); ok {
					//Finally send the link
					rmMsg := tgbotapi.NewDeleteMessage(lUpdate.Message.Chat.ID, SentMessage.MessageID)
					msg := tgbotapi.NewMessage(lUpdate.Message.Chat.ID, finalLink)
					msg.ReplyToMessageID = lUpdate.Message.MessageID
					_, _ = bot.Send(msg)
					_, _ = bot.Send(rmMsg)
				} else {
					edited := tgbotapi.NewEditMessageText(lUpdate.Message.Chat.ID, SentMessage.MessageID, "Cannot deserialize the web page json: Cannot find `path` in the json. Json is:\n"+string(readBuf))
					_, _ = bot.Send(edited)
				}
			}(update)
		}
	}
}

// https://stackoverflow.com/a/44570361/4213397
func getFileName(url string) string {
	r, _ := http.NewRequest("GET", url, nil)
	return path.Base(r.URL.Path)
}

//Checks if the user is admin or not
func checkAdmin(value int) bool {
	for _, i := range Config.Admins {
		if i == value {
			return true
		}
	}
	return false
}
