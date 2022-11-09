package main

import (
	_ "image/jpeg"
	"net/http"
	"net/url"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func main() {
	loadConfig()
	// loadDB()

	log.Info("connecting telegram api server...")

	proxyUrl, err := url.Parse(HttpProxy)
	cli := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}

	bot, err := tgbotapi.NewBotAPIWithClient(TGBotToken, cli)
	if err != nil {
		for {
			bot, err = tgbotapi.NewBotAPIWithClient(TGBotToken, cli)
			if err != nil {
				switch err.(type) {
				case *url.Error:
					log.Errorf("Internet is dead :( retrying to connect in 2 minutes")
					time.Sleep(1 * time.Minute)
				default:
					log.Fatal(err)
				}
			} else {
				break
			}
		}
	}

	bot.Debug = true

	log.Infof("Authorized on account %s", bot.Self.UserName)

	{
		mqttClient := getMQTTClient()
		wg := sync.WaitGroup{}
		wg.Add(1)
		topic := "frigate/events"

		if token := mqttClient.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
			eventHandler(msg.Payload(), bot)
		}); token.Wait() && token.Error() != nil {
			log.Errorf("mqtt event failed", token.Error())
			wg.Done()
		}

		log.Infof("Subscribed to topic %s\n", topic)
		wg.Wait()
	}
}
