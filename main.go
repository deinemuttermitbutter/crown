package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/pelletier/go-toml"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"syscall"
)

var (
	Token string
	Regex = regexp.MustCompile(`(?m)(?:discord\.gift|discord\.com/gifts)/\b([0-9a-zA-Z]{16,24})\b`)
)

func init() {
	if len(os.Args) == 1 {
		fileBytes, err := ioutil.ReadFile("config.toml")

		if err != nil {
			log.Fatalln("💔 Couldn't read config file, is it missing?")
		}

		var config = struct {
			Token string
		}{}

		err = toml.Unmarshal(fileBytes, &config)

		if err != nil {
			log.Fatalln("💔 Couldn't parse config file, exiting")
		}

		Token = config.Token
	} else {
		flag.StringVar(&Token, "t", "", "Discord Token")
		flag.Parse()
	}

	if Token == "" {
		fmt.Fprintln(os.Stderr, "💔 No Discord token provided, exiting")
		os.Exit(1)
	}
}

func main() {
	bot, err := discordgo.New(Token)

	if err != nil {
		fmt.Fprintln(os.Stderr, "💔 Couldn't create Discord session:", err)
		os.Exit(1)
	}

	fmt.Printf("\n")

	bot.AddHandler(ready)
	bot.AddHandler(messageCreate)

	err = bot.Open()

	if err != nil {
		fmt.Fprintln(os.Stderr, "💔 Couldn't establish WebSocket connection:", err)
		os.Exit(1)
	}

	fmt.Println("👑 Bot running, press CTRL+C to exit")
	syscalls := make(chan os.Signal, 1)
	signal.Notify(syscalls, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, os.Interrupt, os.Kill)
	fmt.Printf("🔺 Signal `%v` detected, disconnecting bot and exiting\n\n", <-syscalls)

	_ = bot.Close()
}

func ready(_ *discordgo.Session, event *discordgo.Ready) {
	fmt.Println("👤 Logged in as", event.User.String())
}

func messageCreate(_ *discordgo.Session, event *discordgo.MessageCreate) {
	matches := Regex.FindAllStringSubmatch(event.Content, -1)

	for i := 0; i < len(matches); i++ {
		code := matches[i][1]

		if length := len(code); length != 16 && length != 24 {
			return
		}
		if code == "" {
			return
		}

		go redeemNitroGift(code, event.ChannelID)
	}
}

func redeemNitroGift(code string, channelID string) {
	requestBody := "{\"channel_id\": \"" + channelID + "\", \"payment_source_id\":null}"
	url := "https://discordapp.com/api/v8/entitlements/gift-codes/" + code + "/redeem"

	request, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(requestBody)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", Token)

	response, _ := http.DefaultClient.Do(request)
	body, _ := ioutil.ReadAll(response.Body)

	if response.StatusCode > 199 && response.StatusCode < 300 {
		fmt.Println("✨ Successfully claimed code:", code, string(body))
	} else {
		fmt.Println("⛔ Couldn't claim code:", code, string(body))
	}
}
