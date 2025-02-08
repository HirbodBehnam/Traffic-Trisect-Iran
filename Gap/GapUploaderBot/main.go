package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dustin/go-humanize"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"io"
	"io/ioutil"
	"log"
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"sync"
	"time"
)

type ConfigJson struct {
	GapToken      string
	TelegramToken string
	Admins        []int64
	MaxFileSize   int
}

var Config ConfigJson
var DownloadMap = NewCancelMap()
var client = &http.Client{}
var bot *tgbotapi.BotAPI
var pool sync.Pool

const VERSION = "1.2.0"

func main() {
	{ // Parse argument
		cnf := "config.json"
		if len(os.Args) > 1 {
			cnf = os.Args[1]
		}

		confF, err := ioutil.ReadFile(cnf)
		if err != nil {
			log.Fatalln("Cannot read the config file. (io Error) " + err.Error())
		}

		err = json.Unmarshal(confF, &Config)
		if err != nil {
			log.Fatalln("Cannot read the config file. (Parse Error) " + err.Error())
		}
	}

	log.Println("Gap uploader bot By Hirbod Behnam")
	log.Println("Version", VERSION)
	var err error
	bot, err = tgbotapi.NewBotAPI(Config.TelegramToken)
	if err != nil {
		log.Fatalln("Cannot initialize the bot: " + err.Error())
	}
	log.Printf("Bot authorized on account %s", bot.Self.UserName)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	pool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 32*1024)
		},
	}

	for update := range updates {
		if update.Message == nil && update.CallbackQuery == nil {
			continue
		}
		// Check query on cancel points
		if update.CallbackQuery != nil {
			DownloadMap.Cancel(update.CallbackQuery.Data)
			continue
		}
		// Check commands
		if update.Message.IsCommand() {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			switch update.Message.Command() {
			case "start":
				msg.Text = "Welcome to GapUploader bot.\nYou can use this bot to upload your data to Gap servers to download them at lower costs.\nJust send me the link of the file."
			case "id":
				msg.Text = strconv.FormatInt(update.Message.From.ID, 10)
			case "about":
				msg.Text = "A simple bot by Hirbod Behnam\nSource at https://github.com/HirbodBehnam/Traffic-Trisect-Iran"
			default:
				msg.Text = "I don't know that command"
			}
			_, _ = bot.Send(msg)
			continue
		}

		// Next steps require the user to be admin
		if checkAdmin(update.Message.From.ID) {
			go processUpdate(update)
		}
	}
}

func processUpdate(update tgbotapi.Update) {
	//At first send a message to user
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Getting info about the file...")
	msg.ReplyToMessageID = update.Message.MessageID
	SentMessage, err := bot.Send(msg)
	if err != nil {
		log.Println("Error sending message:", err)
		return
	}
	// Get the file length
	sourceResponse, err := http.Get(update.Message.Text)
	if err != nil {
		edited := tgbotapi.NewEditMessageText(update.Message.Chat.ID, SentMessage.MessageID, "Error on getting file size: "+err.Error())
		_, _ = bot.Send(edited)
		return
	}
	if sourceResponse.StatusCode != http.StatusOK {
		edited := tgbotapi.NewEditMessageText(update.Message.Chat.ID, SentMessage.MessageID, "Error on getting file size: Page returned code "+strconv.FormatInt(int64(sourceResponse.StatusCode), 10))
		_, _ = bot.Send(edited)
		return
	}
	downloadSize, err := strconv.Atoi(sourceResponse.Header.Get("Content-Length"))
	if err != nil {
		edited := tgbotapi.NewEditMessageText(update.Message.Chat.ID, SentMessage.MessageID, "Error on getting file size: "+err.Error()+"\nThe bot might have failed to get Content-Length or maybe the web server does not support it.")
		_, _ = bot.Send(edited)
		return
	}
	if downloadSize > Config.MaxFileSize {
		edited := tgbotapi.NewEditMessageText(update.Message.Chat.ID, SentMessage.MessageID, "The file you are going to download is too big! The server limit is "+humanize.Bytes(uint64(Config.MaxFileSize))+" however the file you requested is "+humanize.Bytes(uint64(downloadSize))+".")
		_, _ = bot.Send(edited)
		return
	}

	// create the id for cancel button
	msgID := uuid.New().String()

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Cancel", msgID)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	DownloadMap.Add(msgID, cancel)
	defer DownloadMap.Delete(msgID)

	writtenInSecond := 0
	downloaded := 0
	r, w := io.Pipe() // Use pipe to reduce ram usage
	m := multipart.NewWriter(w)
	done := make(chan struct{}, 1)
	{
		go func() { // Report download
			var percent float64
			for {
				select {
				case <-done:
					return
				default:
					percent = float64(downloaded) / float64(downloadSize) * 100

					progressbar := "["
					tempPercent := math.Floor(percent / 10)
					for i := 0; i < int(tempPercent); i++ {
						progressbar += "█" // https://www.compart.com/en/unicode/U+2588
					}
					tempPercent = 10 - tempPercent
					for i := 0; i < int(tempPercent); i++ {
						progressbar += "▁" // https://www.compart.com/en/unicode/U+2581
					}
					progressbar += "]"

					text := "Downloading and Uploading:\n" + humanize.Bytes(uint64(downloaded)) + " from " + humanize.Bytes(uint64(downloadSize)) + "\n" + progressbar + "\nSpeed: " + humanize.Bytes(uint64(writtenInSecond)) + "/s"
					if downloaded == downloadSize {
						text += "\n\nFinishing upload might take a while, if you get an 405 error, try at another time."
					}

					edited := tgbotapi.NewEditMessageText(update.Message.Chat.ID, SentMessage.MessageID, text)
					writtenInSecond = 0
					edited.ReplyMarkup = &inlineKeyboard
					_, _ = bot.Send(edited)
				}
				time.Sleep(time.Second)
			}
		}()
		go func() {
			defer sourceResponse.Body.Close()
			defer w.Close()
			defer m.Close()

			// Get filename
			FileName := ""
			if fn := sourceResponse.Header.Get("Content-Disposition"); fn != "" {
				_, params, err := mime.ParseMediaType(fn)
				if err == nil {
					FileName = params["filename"]
					escaped, err := url.QueryUnescape(FileName)
					if err == nil {
						FileName = escaped
					}
				}
			}
			if FileName == "" {
				FileName = getFileName(update.Message.Text)
			}

			part, _ := m.CreateFormFile("file", FileName)
			// This part is mostly like io.copy
			buf := pool.Get().([]byte)
			for {
				nr, er := sourceResponse.Body.Read(buf) // This will encounter an error when closed
				if nr > 0 {
					nw, ew := part.Write(buf[0:nr]) // directly upload to gap
					if nw > 0 {
						downloaded += nw
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
			pool.Put(buf)
		}()
	}

	// create the upload request
	req, err := http.NewRequest("POST", "https://api.gap.im/upload", r)
	if err != nil {
		edited := tgbotapi.NewEditMessageText(update.Message.Chat.ID, SentMessage.MessageID, "Error on initializing upload (request): "+err.Error())
		_, _ = bot.Send(edited)
		return
	}

	req.Header.Set("Content-Type", m.FormDataContentType())
	req.Header.Add("token", Config.GapToken)
	req = req.WithContext(ctx)

	// Submit the request
	gapResponse, err := client.Do(req)
	done <- struct{}{} // Terminate the status reporter
	_ = r.Close()
	_ = w.Close()
	if err != nil {
		edited := tgbotapi.NewEditMessageText(update.Message.Chat.ID, SentMessage.MessageID, "Error on uploading file: "+err.Error())
		_, _ = bot.Send(edited)
		return
	}
	defer gapResponse.Body.Close()
	if gapResponse.StatusCode != http.StatusOK { // In Gap 403 means invalid token; 500 invalid file type or big file. 405 means that their server is fucked
		edited := tgbotapi.NewEditMessageText(update.Message.Chat.ID, SentMessage.MessageID, "Error on uploading file: The web page returned status code "+strconv.FormatInt(int64(gapResponse.StatusCode), 10))
		_, _ = bot.Send(edited)
		return
	}
	// Try to deserialize json
	var jsonRes map[string]interface{}
	err = json.NewDecoder(gapResponse.Body).Decode(&jsonRes)
	if err != nil {
		edited := tgbotapi.NewEditMessageText(update.Message.Chat.ID, SentMessage.MessageID, "Cannot deserialize the web page json: "+err.Error())
		_, _ = bot.Send(edited)
		return
	}
	if finalLink, ok := jsonRes["path"].(string); ok {
		// Finally, send the link
		rmMsg := tgbotapi.NewDeleteMessage(update.Message.Chat.ID, SentMessage.MessageID)
		_, _ = bot.Send(rmMsg)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, finalLink)
		msg.DisableWebPagePreview = true
		msg.ReplyToMessageID = update.Message.MessageID
		_, _ = bot.Send(msg)
	} else {
		edited := tgbotapi.NewEditMessageText(update.Message.Chat.ID, SentMessage.MessageID, "Cannot deserialize the web page json: Cannot find `path` in the json. Json is:\n"+fmt.Sprint(jsonRes))
		_, _ = bot.Send(edited)
	}
}

// https://stackoverflow.com/a/44570361/4213397
func getFileName(url string) string {
	r, _ := http.NewRequest("GET", url, nil)
	return path.Base(r.URL.Path)
}

//Checks if the user is admin or not
func checkAdmin(value int64) bool {
	for _, i := range Config.Admins {
		if i == value {
			return true
		}
	}
	return false
}
