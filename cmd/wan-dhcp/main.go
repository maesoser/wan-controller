package main

import (
	"flag"
	"fmt"
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

	PidPath := flag.String("pid", "/etc/wan-data/wan-dhcp.pid", "PID File")
	flag.Parse()

	log.WithFields(log.Fields{"module": moduleName}).Info("Starting wan-dhcp")

	err = ioutil.WriteFile(*PidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0664)
	if err != nil {
		log.WithFields(log.Fields{"module": moduleName, "error": err.Error()}).Fatalln("Error writting PID file")
	}

	server := dhcp.Server{}
	server.ServeDHCP()
}
