package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type config struct {
	IpPing       []string
	MaxPing      int64
	WarningPing  int
	DownCar      int
	AliveCar     int
	DelayMessage time.Duration

	SwitchTelegram bool
	TelegramBotKey string
	ChatID         string

	SwitchDiscord  bool
	DiscordWebHook string
}

type Counter struct {
	Count        int
	CountAlive   int
	CountTimeOut int
	TimeOut      bool
	Delay        *time.Timer
}

var ipTemp = map[string]*Counter{}

type discordMessage struct {
	AvatarURL string `json:"avatar_url"`
	Content   string `json:"content"`
}

const (
	ProtocolICMP = 1
)

// Default to listen on all IPv4 interfaces
var ListenAddr = "0.0.0.0"

func Ping(addr string) (*net.IPAddr, time.Duration, error) {
	// Start listening for icmp replies
	c, err := icmp.ListenPacket("ip4:icmp", ListenAddr)
	if err != nil {
		return nil, 0, err
	}
	defer c.Close()

	// Resolve any DNS (if used) and get the real IP of the target
	dst, err := net.ResolveIPAddr("ip4", addr)
	if err != nil {
		panic(err)
	}

	// Make a new ICMP message
	m := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1, //<< uint(seq), // TODO
			Data: []byte(""),
		},
	}
	b, err := m.Marshal(nil)
	if err != nil {
		return dst, 0, err
	}

	// Send it
	start := time.Now()
	n, err := c.WriteTo(b, dst)
	if err != nil {
		return dst, 0, err
	} else if n != len(b) {
		return dst, 0, fmt.Errorf("got %v; want %v", n, len(b))
	}

	// Wait for a reply
	reply := make([]byte, 256)
	err = c.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		return dst, 0, err
	}
	n, peer, err := c.ReadFrom(reply)
	if err != nil {
		return dst, 0, err
	}
	duration := time.Since(start)

	// Pack it up boys, we're done here
	rm, err := icmp.ParseMessage(ProtocolICMP, reply[:n])
	if err != nil {
		return dst, 0, err
	}
	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		return dst, duration, nil
	default:
		return dst, 0, fmt.Errorf("got %+v from %v; want echo reply", rm, peer)
	}
}

func main() {
	fmt.Println("Скрипт запустился.")
	var conf config
	var discordMessages discordMessage
	var creatorJSON []byte

	if _, err := toml.DecodeFile("conf.toml", &conf); err != nil {
		fmt.Println(err)
		return
	}

	discordMessages.AvatarURL = "https://media.discordapp.net/attachments/442265055220858880/693198503794442300/unknown.png"

	var body []byte

	client := &fasthttp.Client{MaxConnDuration: time.Second * 5}

	p := func(addr string) {
		_, dur, err := Ping(addr)
		if err != nil {
			if !ipTemp[addr].TimeOut {
				ipTemp[addr].CountTimeOut++
				if ipTemp[addr].CountTimeOut > conf.DownCar {
					if conf.SwitchTelegram {
						_, _, err := client.Get(body, fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=Ваш сервер упал:%%0d%%0aIP - %s.", conf.TelegramBotKey, conf.ChatID, addr))

						if err != nil {
							fmt.Println(err)
						}
					}
					if conf.SwitchDiscord {
						discordMessages.Content = fmt.Sprintf("Ваш сервер упал:\nIP - %s.", addr)
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
					ipTemp[addr].CountTimeOut = 0
					ipTemp[addr].TimeOut = true
				}
			}
			return
		}

		if !ipTemp[addr].TimeOut {
			if dur.Milliseconds() > conf.MaxPing {
				ipTemp[addr].Count++
				if ipTemp[addr].Count > conf.WarningPing {
					ipTemp[addr].Delay = time.NewTimer(time.Second * conf.DelayMessage)
					if conf.SwitchTelegram {
						_, _, err := client.Get(body, fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=Пинг выше нормы:%%0d%%0a%d ms.%%0d%%0aIP - %s.", conf.TelegramBotKey, conf.ChatID, dur.Milliseconds(), addr))
						if err != nil {
							fmt.Println(err)
						}
					}
					if conf.SwitchDiscord {
						discordMessages.Content = fmt.Sprintf("Пинг выше нормы: %d ms.\nIP - %s.", dur.Milliseconds(), addr)
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
					ipTemp[addr].Count = 0
					<-ipTemp[addr].Delay.C
				}
			} else {
				ipTemp[addr].Count = 0
			}
		} else {
			if dur.Milliseconds() > int64(0) {
				ipTemp[addr].CountAlive++

				if ipTemp[addr].CountAlive > conf.AliveCar {
					if conf.SwitchTelegram {
						_, _, err := client.Get(body, fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=Ваш сервер проснулся:%%0d%%0aIP - %s.", conf.TelegramBotKey, conf.ChatID, addr))
						if err != nil {
							fmt.Println(err)
						}
					}
					if conf.SwitchDiscord {
						discordMessages.Content = fmt.Sprintf("Ваш сервер проснулся:\nIP - %s.", addr)
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
					ipTemp[addr].TimeOut = false
					ipTemp[addr].CountAlive = 0
				}
			} else {
				ipTemp[addr].TimeOut = false
				ipTemp[addr].CountAlive = 0
			}
		}
	}

	for _, s := range conf.IpPing {
		go func(addrres string) {
			ipTemp[addrres] = &Counter{0, 0, 0, false, time.NewTimer(time.Second)}

			for {
				fmt.Println(addrres)
				fmt.Println(ipTemp[addrres].CountAlive, "Alive count")
				fmt.Println(ipTemp[addrres].Count, "Big ping")
				fmt.Println(ipTemp[addrres].CountTimeOut, "Time out")

				p(addrres)
				time.Sleep(1 * time.Second)
			}
		}(s)
		// p(conf.IpPing)
	}
	select {}
}
