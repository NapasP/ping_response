package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/sparrc/go-ping"
	"github.com/valyala/fasthttp"
)

type config struct {
	IpPing      string
	MaxPing     int64
	WarningPing int
	DownCar     int
	AliveCar    int

	SwitchTelegram bool
	TelegramBotKey string
	ChatID         string

	SwitchDiscord  bool
	DiscordWebHook string
}

type discordMessage struct {
	AvatarURL string `json:"avatar_url"`
	Content   string `json:"content"`
}

func main() {
	var conf config
	var discordMessages discordMessage
	var creatorJSON []byte

	if _, err := toml.DecodeFile("conf.toml", &conf); err != nil {
		fmt.Println(err)
		return
	}

	discordMessages.AvatarURL = "https://media.discordapp.net/attachments/442265055220858880/693198503794442300/unknown.png"

	pinger, err := ping.NewPinger(conf.IpPing)
	if err != nil {
		panic(err)
	}

	pinger.Count = 5
	pinger.Timeout = 2 * time.Second
	pinger.Size = 32
	pinger.SetPrivileged(true)

	var count int
	var countAlive int
	var countTimeOut int
	var body []byte
	var timeOut bool

	client := &fasthttp.Client{MaxConnDuration: time.Second * 5}

	pinger.OnFinish = func(stats *ping.Statistics) {
		if !timeOut {
			if stats.MaxRtt.Milliseconds() == int64(0) {
				countTimeOut++
				if countTimeOut > conf.DownCar {
					if conf.SwitchTelegram {
						_, _, err := client.Get(body, "https://api.telegram.org/bot"+conf.TelegramBotKey+"/sendMessage?chat_id="+conf.ChatID+"&text="+fmt.Sprintf("Ваша тачка упала, GG."))
						if err != nil {
							fmt.Println(err)
						}
					}

					if conf.SwitchDiscord {
						discordMessages.Content = "Ваша тачка упала, GG."

						creatorJSON, _ = json.Marshal(discordMessages)

						req := fasthttp.AcquireRequest()
						req.Header.SetContentType("application/json")
						req.SetBody(creatorJSON)
						req.Header.SetMethodBytes([]byte("POST"))
						req.SetRequestURIBytes([]byte(conf.DiscordWebHook))
						res := fasthttp.AcquireResponse()
						if err := fasthttp.Do(req, res); err != nil {
							panic("handle error")
						}

						fasthttp.ReleaseRequest(req)
						fasthttp.ReleaseResponse(res)
					}

					countTimeOut = 0
					timeOut = true
				}
			}

			if stats.MaxRtt.Milliseconds() > conf.MaxPing {
				count++
				if count > conf.WarningPing {
					if conf.SwitchTelegram {
						_, _, err := client.Get(body, "https://api.telegram.org/bot"+conf.TelegramBotKey+"/sendMessage?chat_id="+conf.ChatID+"&text="+fmt.Sprintf("Пинг выше нормы: %d ms.", stats.MaxRtt.Milliseconds()))
						if err != nil {
							fmt.Println(err)
						}
					}

					if conf.SwitchDiscord {
						discordMessages.Content = fmt.Sprintf("Пинг выше нормы: %d ms.", stats.MaxRtt.Milliseconds())

						creatorJSON, _ = json.Marshal(discordMessages)

						req := fasthttp.AcquireRequest()
						req.Header.SetContentType("application/json")
						req.SetBody(creatorJSON)
						req.Header.SetMethodBytes([]byte("POST"))
						req.SetRequestURIBytes([]byte(conf.DiscordWebHook))
						res := fasthttp.AcquireResponse()
						if err := fasthttp.Do(req, res); err != nil {
							panic("handle error")
						}

						fasthttp.ReleaseRequest(req)
						fasthttp.ReleaseResponse(res)
					}

					count = 0
				}
			}
		} else {
			if stats.MaxRtt.Milliseconds() > int64(0) {
				countAlive++
				if count > conf.AliveCar {
					if conf.SwitchTelegram {
						_, _, err := client.Get(body, "https://api.telegram.org/bot"+conf.TelegramBotKey+"/sendMessage?chat_id="+conf.ChatID+"&text="+fmt.Sprintf("Ваша тачка ожила."))
						if err != nil {
							fmt.Println(err)
						}
					}

					if conf.SwitchDiscord {
						discordMessages.Content = "Ваша тачка ожила."

						creatorJSON, _ = json.Marshal(discordMessages)

						req := fasthttp.AcquireRequest()
						req.Header.SetContentType("application/json")
						req.SetBody(creatorJSON)
						req.Header.SetMethodBytes([]byte("POST"))
						req.SetRequestURIBytes([]byte(conf.DiscordWebHook))
						res := fasthttp.AcquireResponse()
						if err := fasthttp.Do(req, res); err != nil {
							panic("handle error")
						}

						fasthttp.ReleaseRequest(req)
						fasthttp.ReleaseResponse(res)

					}

					timeOut = false
					countAlive = 0
				}
			}
		}
	}

	for {
		pinger.Run()

		timer1 := time.NewTimer(1 * time.Second)
		<-timer1.C
	}
}
