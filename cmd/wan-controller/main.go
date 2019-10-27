package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/maesoser/wan-agent/pkg/metrics"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{})

	log.WithFields(log.Fields{"module": "wan-agent"}).Info("Starting wan-controller")

}
