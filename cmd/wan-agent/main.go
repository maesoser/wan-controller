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

func ApplyConfig(r vppmgr.VPPManager, c config.Config) error {

	log.WithFields(log.Fields{"module": moduleName}).Info("[1/7] Configuring WAN port")
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
		if err := r.AddIfaceAddress(index, c.Network.Uplink.Address); err != nil {
			return err
		}
	} else {
		r.AddDHCP(index, c.Name)
	}

	log.WithFields(log.Fields{"module": moduleName}).Info("[2/7] Creating network Bridge")
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
	if err := r.AddIfaceToBridge(uint32(index), 1, true); err != nil {
		return err
	}
	if err := r.AddIfaceAddress(index, c.Network.Uplink.Address); err != nil {
		return err
	}
	if err := r.IfaceUp(index); err != nil {
		return err
	}

	/* Add Ports to Bridge
	set interface l2 bridge port2 1
	set interface state port2 up
	*/
	log.WithFields(log.Fields{"module": moduleName}).Info("[3/7] Adding ports to the network bridge")
	for _, port := range c.Network.Ports {
		index, err := r.GetIfIndexByName(port)
		if err != nil {
			return err
		}
		if err := r.AddIfaceToBridge(uint32(index), 1, false); err != nil {
			return err
		}
		if err := r.IfaceUp(index); err != nil {
			return err
		}
	}

	log.WithFields(log.Fields{"module": moduleName}).Info("[4/7] Configuring TAP Port")
	/* Configure TAP Port
	create tap host-if-name lstack host-ip4-addr 192.168.2.2/24
	set int l2 bridge tap0 1
	set int state tap0 up
	*/
	gwaddr := net.ParseIP(c.Network.Gateway)
	ifaddr := gwaddr
	ifaddr[3] + 1
	index, err = r.AddTAPIface("lstack", ifaddr, gwaddr)
	if err != nil {
		return err
	}
	if err := r.AddIfaceToBridge(uint32(index), 1, false); err != nil {
		return err
	}
	if err := r.IfaceUp(index); err != nil {
		return err
	}

	log.WithFields(log.Fields{"module": moduleName}).Info("[5/7] Configuring NAT44")
	/* Configure NAT44
	nat44 add interface address port1
	set interface nat44 in loop0 out port1
	*/
	index, err := r.GetIfIndexByName(c.Network.Uplink.Name)
	if err != nil {
		return err
	}
	if err := r.AddNAT(index); err != nil {
		return err
	}

	log.WithFields(log.Fields{"module": moduleName}).Info("[6/7] Adding NAT rules")
	/* Add NAT entries
	nat44 add static mapping local 192.168.2.2 22 external port1 22 tcp
	*/
	gwaddr := net.ParseIP(c.Network.Gateway)
	gwaddr[3] + 1
	err := r.AddNATRule(index, gwaddr, 22, nil, 22, 0x06){
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{"module": moduleName}).Info("[7/7] Configuring Linux network")
	c.WriteDNS()
	c.WriteHostname()
	if err := route.CheckDefaultGatewayRoute(net.ParseIP(c.Network.Gateway)); err != nil {
		if err := route.AddDefaultRoute(net.ParseIP(c.Network.Gateway)); err != nil {
			return err
		}
	}

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
		log.WithFields(log.Fields{"module": moduleName, "error": err.Error()}).Fatalln("Error writting PID file")
	}

	log.WithFields(log.Fields{"module": moduleName}).Info("Starting wan-agent")

	var routerConfig config.Config
	var vppManager vppmgr.VPPManager
	if err := vppManager.Init(); err != nil {
		log.WithFields(log.Fields{"module": moduleName, "error": err.Error()}).Error("Unable to open a channel with VPP daemon")
	}

	err = routerConfig.Load(*ConfigPath)
	for err != nil {
		log.WithFields(log.Fields{"module": moduleName, "error": err.Error()}).Warnln("Unable to load config, router is not active.")
		time.Sleep(30 * time.Second)
	}
	log.WithFields(log.Fields{"module": moduleName}).Infof("Configuration file loaded, applying it, applying it")

	ApplyConfig(vppManager, routerConfig)
}
