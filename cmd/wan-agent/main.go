package main

//go:generate binapi-generator --input-file=/usr/share/vpp/api/vpe.api.json --output-dir=./binapi
//go:generate binapi-generator --input-file=/usr/share/vpp/api/interface.api.json --output-dir=./binapi
//go:generate binapi-generator --input-file=/usr/share/vpp/api/l2.api.json --output-dir=./binapi

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/maesoser/wan-controller/pkg/config"
	"github.com/maesoser/wan-controller/pkg/vppmgr"
	log "github.com/sirupsen/logrus"
	"time"
)

func ApplyConfig(r vppmgr.VPPManager, c config.Config) error {

	log.WithFields(log.Fields{"module": "wan-agent"}).Info("[1/7] Configuring WAN port")
	/* Configure WAN Port
	set interface state port1 up
	set interface ip address port1 192.168.2.1/24
	set dhcp client interface port1 hostname vpprouter
	*/
	index, err := r.GetIfIndexByName(c.Network.Uplink.Name)
	if err != nil {
		return err
	}
	if err := r.IfaceUp(index); err != nil {
		return err
	}
	if c.Network.Uplink.DHCP == false && c.Network.Uplink.Address != "" {
		r.AddIfaceAddress(index, c.Network.Uplink.Address)
	} else {
		r.AddDHCP(index, c.Name)
	}

	log.WithFields(log.Fields{"module": "wan-agent"}).Info("[2/7] Creating network Bridge")
	/* Configure Loopback Port and Bridge
	loopback create
	set interface l2 bridge loop0 1 bvi
	set interface ip address loop0 192.168.2.1/24
	set interface state loop0 up
	*/
	index, err = r.AddLoopback()
	if err != nil {
		return err
	}
	r.AddIfaceAddress(index, c.Network.Uplink.Address)
	if err := r.IfaceUp(index); err != nil {
		return err
	}

	log.WithFields(log.Fields{"module": "wan-agent"}).Info("[3/7] Adding ports to the network bridge")
	/* Add Ports to Bridge
	set int l2 bridge port2 1
	set int state port2 up
	*/

	log.WithFields(log.Fields{"module": "wan-agent"}).Info("[4/7] Configuring TAP Port")
	/* Configure TAP Port
	create tap host-if-name lstack host-ip4-addr 192.168.2.2/24
	set int l2 bridge tap0 1
	set int state tap0 up
	*/

	log.WithFields(log.Fields{"module": "wan-agent"}).Info("[5/7] Configuring NAT44")
	/* Configure NAT44
	nat44 add interface address port1
	set interface nat44 in loop0 out port1
	*/

	log.WithFields(log.Fields{"module": "wan-agent"}).Info("[6/7] Adding NAT rules")
	/* Add NAT entries
	nat44 add static mapping local 192.168.2.2 22 external port1 22 tcp
	*/

	log.WithFields(log.Fields{"module": "wan-agent"}).Info("[7/7] Configuring Linux network")
	c.WriteDNS()
	c.WriteHostname()
	// Add default linux route

	return nil
}

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
		log.WithFields(log.Fields{"module": "wan-metrics", "error": err.Error()}).Fatalln("Error writting PID file")
	}

	log.WithFields(log.Fields{"module": "wan-agent"}).Info("Starting wan-agent")

	var routerConfig config.Config
	var vppManager vppmgr.VPPManager
	if err := vppManager.Init(); err != nil {
		log.WithFields(log.Fields{"module": "wan-agent", "error": err.Error()}).Error("Unable to open a channel with VPP daemon")
	}

	err = routerConfig.Load(*ConfigPath)
	for err != nil {
		log.WithFields(log.Fields{"module": "wan-agent", "error": err.Error()}).Warnln("Unable to load config, router is not active.")
		time.Sleep(30 * time.Second)
	}
	log.WithFields(log.Fields{"module": "wan-agent"}).Infof("Configuration file loaded, applying it, applying it")

	ApplyConfig(vppManager, routerConfig)
}
