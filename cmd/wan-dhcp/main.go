package main

import (
	"flag"
	"fmt"
	"github.com/maesoser/wan-controller/pkg/config"
	"github.com/maesoser/wan-controller/pkg/dhcpserver"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"os"
	"time"
)

const (
	moduleName = "wan-dhcp"
)

func main() {

	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	var c config.Config
	var monitor metrics.Metric

	ConfigPath := flag.String("config", "/etc/wan-data/routerconfig.json", "Configuration Path")
	PidPath := flag.String("pid", "/etc/wan-data/wan-dhcp.pid", "PID File")
	flag.Parse()

	log.WithFields(log.Fields{"module": moduleName}).Info("Starting wan-dhcp")

	err = ioutil.WriteFile(*PidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0664)
	if err != nil {
		log.WithFields(log.Fields{"module": moduleName, "error": err.Error()}).Fatalln("Error writting PID file")
	}

	err = c.Load(*ConfigPath)
	if err != nil {
		log.WithFields(log.Fields{"module": moduleName, "error": err.Error()}).Fatalln("Error reading config")
	}
	server := dhcpserver.NewServer(
		c.Network.Gateway,
		c.Network.Address,
		c.Network.Gateway,
		c.DNSs,
		"")
	server.ServeDHCP()
}
