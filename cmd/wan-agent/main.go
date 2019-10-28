package main

import (
	"flag"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/maesoser/wan-agent/pkg/config"

	"git.fd.io/govpp.git"
	"git.fd.io/govpp.git/adapter/socketclient"
	"git.fd.io/govpp.git/api"
	"git.fd.io/govpp.git/core"
	"git.fd.io/govpp.git/examples/binapi/vpe"

	"time"
)

func main() {

	sockAddr = flag.String("sock", socketclient.DefaultSocketName, "Path to VPP binary API socket file")
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	log.WithFields(log.Fields{"module": "wan-agent"}).Info("Starting wan-agent")

	var routerConfig config.Config

	err := routerConfig.Load("/etc/wan-data/routerconfig.json")
	for err != nil {
		log.WithFields(log.Fields{"module": "wan-agent", "error": err.Error()}).Warnln("Unable to load config, router is not active.")
		time.Sleep(30 * time.Second)
	}
	log.WithFields(log.Fields{"module": "wan-agent"}).Infof("Configuration loaded, applying it")

	conn, conev, err := govpp.AsyncConnect(*sockAddr, core.DefaultMaxReconnectAttempts, core.DefaultReconnectInterval)
	if err != nil {
		log.Fatalln("ERROR:", err)
	}
	defer conn.Disconnect()

	select {
	case e := <-conev:
		if e.State != core.Connected {
			log.Fatalln("ERROR: connecting to VPP failed:", e.Error)
		}
	}

	ch, err := conn.NewAPIChannel()
	if err != nil {
		log.Fatalln("ERROR: creating channel failed:", err)
	}
	defer ch.Close()

}

func vppVersion(ch api.Channel) {
	req := &vpe.ShowVersion{}
	reply := &vpe.ShowVersionReply{}

	if err := ch.SendRequest(req).ReceiveReply(reply); err != nil {
		log.WithFields(log.Fields{"module": "wan-agent", "error": err.Error()}).Warnln("Unable to retrieve VPP Daemon version")
		return
	}
	fmt.Printf("reply: %+v\n", reply)
	log.WithFields(log.Fields{"module": "wan-agent"}).Infof("Connected to VPP Daemon ver %q", cleanString(reply.Version))
}
