package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/maesoser/wan-agent/pkg/config"
	"time"
)

func main() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	log.WithFields(log.Fields{"module": "wan-agent"}).Info("Starting wan-agent")

	var routerConfig config.Config

	err := routerConfig.Load("/etc/wan-data")
	for err != nil {
		log.WithFields(log.Fields{"module": "wan-agent", "error": err.Error()}).Warnln("Unable to load config, preactivation-state")
		time.Sleep(30 * time.Second)
	}
	log.WithFields(log.Fields{"module": "wan-agent"}).Infof("Configuration loaded, applying it")

}
