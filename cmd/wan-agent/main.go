package main

//go:generate binapi-generator --input-file=/usr/share/vpp/api/vpe.api.json --output-dir=./binapi
//go:generate binapi-generator --input-file=/usr/share/vpp/api/interface.api.json --output-dir=./binapi
//go:generate binapi-generator --input-file=/usr/share/vpp/api/l2.api.json --output-dir=./binapi

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/maesoser/wan-controller/pkg/config"
	"github.com/maesoser/wan-controller/pkg/route"
	"github.com/maesoser/wan-controller/pkg/vppmgr"
	log "github.com/sirupsen/logrus"
	"time"
)

const (
	moduleName = "wan-agent"
)

func main() {

	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	ConfigPath := flag.String("config", "/etc/wan-data/routerconfig.json", "Configuration Path")
	PidPath := flag.String("pid", "/etc/wan-data/wan-agent.pid", "PID File")
	flag.Parse()

	err := ioutil.WriteFile(*PidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0664)
	if err != nil {
		log.WithFields(log.Fields{"module": moduleName, "error": err.Error()}).Fatalln("Error writting PID file")
	}

	log.WithFields(log.Fields{"module": moduleName}).Info("Starting wan-agent")

	var routerConfig config.Config
	var vppManager vppmgr.VPPManager
	if err := vppManager.Init(); err != nil {
		log.WithFields(log.Fields{"module": moduleName, "error": err.Error()}).Error("Unable to open a channel with VPP daemon")
	}

	if err := routerConfig.Load(*ConfigPath); err != nil {
		log.WithFields(log.Fields{"module": moduleName, "error": err.Error()}).Warnln("Unable to load config, router is not active.")
		time.Sleep(30 * time.Second)
	}

	log.WithFields(log.Fields{"module": moduleName}).Infof("Configuration file loaded, applying it, applying it")
	if err := ApplyConfig(vppManager, routerConfig); err != nil {
		log.WithFields(log.Fields{"module": moduleName, "error": err.Error()}).Warnln("Unable to apply VPP Config")
	}
}
