package main

import (	
	"flag"
	"fmt"
	"github.com/maesoser/wan-controller/pkg/metrics"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
)

const (
	moduleName = "wan-dhcp"
)

func main() {

	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	var monitor metrics.Metric

	ListenAddr := flag.String("listen", "127.0.0.1:9600", "Server Addr")
	PidPath := flag.String("pid", "/etc/wan-data/wan-metrics.pid", "PID File")
	flag.Parse()

	log.WithFields(log.Fields{"module": moduleName}).Info("Starting wan-metrics")
	
        err := ioutil.WriteFile(*PidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0664)
	if err != nil {
		log.WithFields(log.Fields{"module": moduleName, "error": err.Error()}).Fatalln("Error writting PID file")
	}

	monitor.Init()
	log.WithFields(log.Fields{"module": moduleName}).Infof("Listening at %s", *ListenAddr)
	err = http.ListenAndServe(*ListenAddr, &monitor)
	log.Panic(err)

}
