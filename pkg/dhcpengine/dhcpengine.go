package dhcpengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	dhcp "github.com/krolaw/dhcp4"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"time"
)

const (
	moduleName = "wan-dhcpeng"
)

func incIPv4Addr(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

type DHCPLease struct {
	IPAddr   net.IP
	MACAddr  net.HardwareAddr
	Creation time.Time
	Static   bool
}

type Server struct {
	Config DHCPConfig
	Leases []DHCPLease
}

type DHCPConfig struct {
	ServerIP      net.IP        `json:"server"`
	Subnet        net.IP        `json:"subnet"`
	Gateway       net.IP        `json:"gw"`
	DNS           net.IP        `json:"dns"`
	DomainName    string        `json:"domain"`
	LeaseDuration time.Duration `json:"lease"`
}

func NewServer(
	serverIP net.IP,
	subnet net.IP,
	gateway net.IP,
	dns net.IP,
	domainName string) Server {
	log.WithFields(log.Fields{"module": moduleName}).Info("Serving from %s, net: %s/%s dns: %s domain: %s\n",
		serverIP, gateway, subnet, dns, domainName)
	config := DHCPConfig{
		ServerIP:      serverIP,
		Subnet:        subnet,
		Gateway:       gateway,
		DNS:           dns,
		DomainName:    domainName,
		LeaseDuration: 24 * time.Hour,
	}
	return Server{
		Config: config,
	}
}

func (s *Server) getOptions() dhcp.Options {
	options := dhcp.Options{
		dhcp.OptionSubnetMask:       s.getMask(),
		dhcp.OptionRouter:           s.getGateway(),
		dhcp.OptionDomainNameServer: s.getDNS(),
	}
	if s.Config.DomainName != "" {
		options[dhcp.OptionDomainName] = []byte(s.Config.DomainName)
	}
	return options
}

func (s *Server) getGateway() net.IP {
	return s.Config.Gateway
}

func (s *Server) getDNS() net.IP {
	return s.Config.ServerIP
}

func (s *Server) getMask() net.IPMask {
	return net.IPMask(s.Config.Subnet)
}

func (s *Server) getNetwork() net.IPNet {
	return net.IPNet{
		IP:   s.getGateway(),
		Mask: s.getMask(),
	}
}

// Get's
func (s *Server) findLeaseByMac(addr net.HardwareAddr) (DHCPLease, error) {
	for _, lease := range s.Leases {
		if lease.MACAddr.String() == addr.String() {
			return lease, nil
		}
	}
	return DHCPLease{}, fmt.Errorf("no preassigned lease found for %s", addr.String())
}

func (s *Server) findLeaseByIPAddr(addr net.IP) (DHCPLease, error) {
	for _, lease := range s.Leases {
		if lease.IPAddr.Equal(addr) {
			return lease, nil
		}
	}
	return DHCPLease{}, fmt.Errorf("no preassigned lease found for %s", addr.String())
}

func (s *Server) releaseOutdated() int {
	var leases []DHCPLease
	deleted := 0
	for _, lease := range s.Leases {
		if lease.Creation.Add(s.Config.LeaseDuration).After(time.Now()) || lease.Static {
			leases = append(leases, lease)
		} else {
			deleted++
		}
	}
	s.Leases = leases
	return deleted
}

/*
createLease:
	- Gives a new allocatied leases
	- Gives you the old one, with an updated Creation timestamp
*/
func (s *Server) createLease(req dhcp.Packet) DHCPLease {
	var lease DHCPLease
	network := s.getNetwork()
	for ip := s.getGateway().Mask(s.getMask()); network.Contains(ip); incIPv4Addr(ip) {
		var err error
		lease, err = s.findLeaseByIPAddr(ip)
		if err != nil && lease.IPAddr == nil { // If this addr is empty
			lease.Creation = time.Now()
			lease.MACAddr = req.CHAddr()
			lease.Static = true
			lease.IPAddr = ip
			return lease
		}
		lease.Creation = time.Now()
		return lease
	}
	return lease
}

func (s *Server) dhcpDiscover(req dhcp.Packet, options dhcp.Options) dhcp.Packet {
	lease, err := s.findLeaseByMac(req.CHAddr())
	if err != nil {
		log.WithFields(log.Fields{"module": moduleName, "error": err.Error()}).Errorln("Error finding Lease by MAC Addr")
		lease = s.createLease(req)
		s.Leases = append(s.Leases, lease)
	}
	log.WithFields(log.Fields{"module": moduleName}).Info("Offering %s to %s", lease.IPAddr, req.CHAddr().String())
	opts := s.getOptions()
	return dhcp.ReplyPacket(req,
		dhcp.Offer,
		s.Config.ServerIP,
		lease.IPAddr,
		s.Config.LeaseDuration,
		opts.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]),
	)
}

func (s *Server) dhcpRequest(req dhcp.Packet, options dhcp.Options) dhcp.Packet {
	if server, ok := options[dhcp.OptionServerIdentifier]; ok && !net.IP(server).Equal(s.Config.ServerIP) {
		log.WithFields(log.Fields{"module": moduleName}).Info("Message for a different server?")
		return nil
	}
	reqIP := net.IP(options[dhcp.OptionRequestedIPAddress])
	if reqIP == nil {
		reqIP = net.IP(req.CIAddr())
	}
	opts := s.getOptions()
	if len(reqIP) == 4 && !reqIP.Equal(net.IPv4zero) {
		lease, err := s.findLeaseByMac(req.CHAddr())
		if err != nil {
			log.WithFields(log.Fields{"module": moduleName, "error": err.Error()}).Errorf("NAK to %s\n", req.CHAddr().String())
			return dhcp.ReplyPacket(req, dhcp.NAK, s.Config.ServerIP, nil, 0, nil)
		}
		if lease.IPAddr.Equal(reqIP) == false {
			log.WithFields(log.Fields{"module": moduleName, "error": err.Error()}).Errorf("NAK to %s: expected %s, requested %s\n",
				req.CHAddr().String(), lease.IPAddr.String(), reqIP.String())
			return dhcp.ReplyPacket(req, dhcp.NAK, s.Config.ServerIP, nil, 0, nil)
		}
		return dhcp.ReplyPacket(req, dhcp.ACK, s.Config.ServerIP, reqIP, s.Config.LeaseDuration,
			opts.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]))
	}
	return dhcp.ReplyPacket(req, dhcp.NAK, s.Config.ServerIP, nil, 0, nil)
}

func (s *Server) dhcpRelease(req dhcp.Packet, options dhcp.Options) int {
	var leases []DHCPLease
	deleted := 0
	for _, lease := range s.Leases {
		if lease.MACAddr.String() != eq.CHAddr().String() {
			leases = append(leases, lease)
		} else {
			deleted++
		}
	}
	s.Leases = leases
	return deleted
}

// ServeDHCP handles incoming dhcp requests.
func (s *Server) ServeDHCP(req dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) dhcp.Packet {
	if s.Config.ServerIP == nil {
		log.WithFields(log.Fields{"module": moduleName}).Errorln("Server not yet configuring, waiting for a POST request with the configuration")
		return nil
	}
	log.WithFields(log.Fields{"module": moduleName}).Info("Recv DHCP type %\n", msgType)
	switch msgType {
	case dhcp.Discover:
		return s.dhcpDiscover(req, options)
	case dhcp.Request:
		return s.dhcpRequest(req, options)
	case dhcp.Release:
		return s.dhcpRelease(req, options)
	case dhcp.Decline:
		return s.dhcpRelease(req, options)
	}

	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if r.URL.Path == "/config" {
			b := new(bytes.Buffer)
			json.NewEncoder(b).Encode(s.Config)
			fmt.Fprintf(w, "%v", b)
		} else if r.URL.Path == "/leases" {
			b := new(bytes.Buffer)
			json.NewEncoder(b).Encode(s.Leases)
			fmt.Fprintf(w, "%v", b)
		}
	case "POST":
		if r.URL.Path == "/config" {
			var config DHCPConfig
			err := json.NewDecoder(r.Body).Decode(&config)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			log.WithFields(log.Fields{"module": moduleName}).Info("Serving from %s, Net: %s/%s DNS: %s Domain: %s\n",
				config.ServerIP,
				config.Gateway,
				config.Subnet,
				config.DNS,
				config.DomainName,
			)
			s.Config = config
		} else if r.URL.Path == "/leases" {
			var lease DHCPLease
			err := json.NewDecoder(r.Body).Decode(&lease)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			lease.Creation = time.Now()
			log.WithFields(log.Fields{"module": moduleName}).Info("Adding Lease %v=%v Static: %v\n",
				lease.IPAddr,
				lease.MACAddr,
				lease.Static,
			)
			s.Leases = append(s.Leases, lease)
		}
	default:
		fmt.Fprintf(w, "%v", s)

	}
}
