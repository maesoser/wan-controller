package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/maesoser/wan-controller/pkg/config"
	"github.com/maesoser/wan-controller/pkg/metrics"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"
)

func Send(data []byte, address, uuid string) error {

	client := http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
	endpoint := "http://" + address + "/" + uuid + "/metrics"
	resp, err := client.Post(endpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Returned status code is %d (%s)", resp.StatusCode, resp.Status)
	}
	return nil
}

func main() {

	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	var c config.Config
	var monitor metrics.Metric

	Interval := flag.String("interval", "5m", "Update Interval")
	Verbose := flag.Bool("verbose", false, "Verbose output")
	ConfigPath := flag.String("config", "/etc/wan-data/routerconfig.json", "Configuration Path")
	PidPath := flag.String("pid", "/etc/wan-data/wan-metrics.pid", "PID File")
	flag.Parse()

	sleepTime, err := time.ParseDuration(*Interval)
	if err != nil {
		log.WithFields(log.Fields{"module": "wan-metrics", "error": err.Error()}).Fatalln("Error parsing interval")
	}

	log.WithFields(log.Fields{"module": "wan-metrics"}).Info("Starting wan-metrics")

	err = ioutil.WriteFile(*PidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0664)
	if err != nil {
		log.WithFields(log.Fields{"module": "wan-metrics", "error": err.Error()}).Fatalln("Error writting PID file")
	}

	err = c.Load(*ConfigPath)
	if err != nil {
		log.WithFields(log.Fields{"module": "wan-metrics", "error": err.Error()}).Fatalln("Error reading config")
	}
	target := c.GetController()
	monitor.Init()
	for {
		monitor.Update()
		if *Verbose {
			monitor.LogSystem()
		}
		data, _ := monitor.Data()
		err := Send(data, target, c.UUID)
		if err != nil {
			log.WithFields(log.Fields{"module": "wan-metrics", "error": err.Error()}).Errorln("Error sending metrics")
		}
		time.Sleep(sleepTime)
	}
}
