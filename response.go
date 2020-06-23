package main

import (
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/sparrc/go-ping"
	"github.com/valyala/fasthttp"
)

type config struct {
	IpPing         string
	MaxPing        int64
	WarningPing    int
	DownCar        int
	AliveCar       int
	SwitchTelegram bool
	TelegramBotKey string
	ChatID         string
}

func main() {
	var conf config

	if _, err := toml.DecodeFile("conf.toml", &conf); err != nil {
		fmt.Println(err)
		return
	}

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
					if !conf.SwitchTelegram {
						_, _, err := client.Get(body, "https://api.telegram.org/bot"+conf.TelegramBotKey+"/sendMessage?chat_id="+conf.ChatID+"&text="+fmt.Sprintf("Ваша тачка упала, GG."))
						if err != nil {
							fmt.Println(err)
						}
					}
					countTimeOut = 0
					timeOut = true
				}
			}

			if stats.MaxRtt.Milliseconds() > conf.MaxPing {
				count++
				if count > conf.WarningPing {
					if !conf.SwitchTelegram {
						_, _, err := client.Get(body, "https://api.telegram.org/bot"+conf.TelegramBotKey+"/sendMessage?chat_id="+conf.ChatID+"&text="+fmt.Sprintf("Пинг выше нормы: %d ms.", stats.MaxRtt.Milliseconds()))
						if err != nil {
							fmt.Println(err)
						}
					}
					count = 0
				}
			}
		} else {
			if stats.MaxRtt.Milliseconds() > int64(0) {
				countAlive++
				if count > conf.AliveCar {
					if !conf.SwitchTelegram {
						_, _, err := client.Get(body, "https://api.telegram.org/bot"+conf.TelegramBotKey+"/sendMessage?chat_id="+conf.ChatID+"&text="+fmt.Sprintf("Ваша тачка ожила."))
						if err != nil {
							fmt.Println(err)
						}
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
