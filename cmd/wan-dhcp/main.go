package main

import (
	"flag"
	"fmt"
	dhcp "github.com/krolaw/dhcp4"
	dhcpeng "github.com/maesoser/wan-controller/pkg/dhcpengine"
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

	PidPath := flag.String("pid", "/etc/wan-data/wan-dhcp.pid", "PID File")
	ListenAddr := flag.String("listen", "127.0.0.1:9610", "Server Addr")
	flag.Parse()

	log.WithFields(log.Fields{"module": moduleName}).Info("Starting wan-dhcp")

	err := ioutil.WriteFile(*PidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0664)
	if err != nil {
		log.WithFields(log.Fields{"module": moduleName, "error": err.Error()}).Fatalln("Error writting PID file")
	}

	engine := dhcpeng.Server{}
	go func() {
		log.WithFields(log.Fields{"module": moduleName}).Infof("DHCP Listening at 0.0.0.0:67")
		err := dhcp.ListenAndServe(&engine)
		log.Panic(err)
	}()
	log.WithFields(log.Fields{"module": moduleName}).Infof("API Listening at %s", *ListenAddr)
	err = http.ListenAndServe(*ListenAddr, &engine)
	log.Panic(err)
}
