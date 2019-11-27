package dhcpserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	dhcp "github.com/krolaw/dhcp4"
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
	serverIP      net.IP
	options       dhcp.Options
	leaseDuration time.Duration
	leases        []DHCPLease
}

type DHCPConfig struct {
	serverIP      net.IP
	subnet        net.IPMask
	gateway       net.IP
	dns           net.IP
	domainName    string
	leaseDuration time.Duration
}

func NewServer(
	serverIP net.IP,
	subnet net.IPMask,
	gateway net.IP,
	dns net.IP,
	domainName string) Server {
	options := dhcp.Options{
		dhcp.OptionSubnetMask:       subnet,
		dhcp.OptionRouter:           gateway,
		dhcp.OptionDomainNameServer: dns,
	}
	if domainName != "" {
		options[dhcp.OptionDomainName] = []byte(domainName)
	}
	log.Printf("serving as id %s, subnet %s gw %s dns %s domain %s\n", serverIP, subnet, gateway, dns, domainName)
	return Server{
		serverIP:      serverIP,
		options:       options,
		leaseDuration: 24 * time.Hour,
	}
}

func (s *Server) getGateway() net.IP {
	return net.IP(s.options[dhcp.OptionRouter])
}

func (s *Server) getDNS() net.IP {
	return net.IP(s.options[dhcp.OptionDomainNameServer])
}

func (s *Server) getMask() net.IPMask {
	return net.IPMask(s.options[dhcp.OptionDomainNameServer])
}

func (s *Server) getNetwork() net.IPNet {
	return net.IPNet{
		IP:   s.getGateway(),
		Mask: s.getMask(),
	}
}

// Get's
func (s *Server) findLeaseByMac(addr net.HardwareAddr) (DHCPLease, error) {
	for _, lease := range s.leases {
		if lease.MACAddr.String() == addr.String() {
			return lease, nil
		}
	}
	return DHCPLease{}, fmt.Errorf("no preassigned lease found for %s", addr.String())
}

func (s *Server) findLeaseByIPAddr(addr net.IP) (DHCPLease, error) {
	for _, lease := range s.leases {
		if lease.IPAddr.Equal(addr) {
			return lease, nil
		}
	}
	return DHCPLease{}, fmt.Errorf("no preassigned lease found for %s", addr.String())
}

func (s *Server) releaseOutdated() int {
	var leases []DHCPLease
	deleted := 0
	for _, lease := range s.leases {
		if lease.Creation.Add(s.leaseDuration).After(time.Now()) || lease.Static {
			leases = append(leases, lease)
		} else {
			deleted++
		}
	}
	s.leases = leases
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
		log.Println(err)
		lease = s.createLease(req)
		s.leases = append(s.leases, lease)
	}
	log.Printf("Offering %s to %s", lease.IPAddr, req.CHAddr().String())
	return dhcp.ReplyPacket(req,
		dhcp.Offer,
		s.serverIP,
		lease.IPAddr,
		s.leaseDuration,
		s.options.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]),
	)
}

func (s *Server) dhcpRequest(req dhcp.Packet, options dhcp.Options) dhcp.Packet {
	if server, ok := options[dhcp.OptionServerIdentifier]; ok && !net.IP(server).Equal(s.serverIP) {
		log.Printf("message for a different server?")
		return nil
	}
	reqIP := net.IP(options[dhcp.OptionRequestedIPAddress])
	if reqIP == nil {
		reqIP = net.IP(req.CIAddr())
	}

	if len(reqIP) == 4 && !reqIP.Equal(net.IPv4zero) {
		lease, err := s.findLeaseByMac(req.CHAddr())
		if err != nil {
			log.Printf("NAK to %s: %s\n", req.CHAddr().String(), err)
			return dhcp.ReplyPacket(req, dhcp.NAK, s.serverIP, nil, 0, nil)
		}
		if lease.IPAddr.Equal(reqIP) == false {
			log.Printf("NAK to %s: expected %s, requested %s\n", req.CHAddr().String(), lease.IPAddr.String(), reqIP.String())
			return dhcp.ReplyPacket(req, dhcp.NAK, s.serverIP, nil, 0, nil)
		}
		return dhcp.ReplyPacket(req, dhcp.ACK, s.serverIP, reqIP, s.leaseDuration,
			s.options.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]))
	}
	return dhcp.ReplyPacket(req, dhcp.NAK, s.serverIP, nil, 0, nil)
}

// ServeDHCP handles incoming dhcp requests.
func (s *Server) ServeDHCP(req dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) dhcp.Packet {
	if s.serverIP == nil {
		log.Printf("Server not yet configuring, waiting for a POST request with the configuration")
		return nil
	}
	switch msgType {
	case dhcp.Discover:
		return s.dhcpDiscover(req, options)
	case dhcp.Request:
		return s.dhcpRequest(req, options)
	}

	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		fmt.Fprintf(w, "%v", s)
	case "POST":
		var config DHCPConfig
		err := json.NewDecoder(r.Body).Decode(&config)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.options = dhcp.Options{
			dhcp.OptionSubnetMask:       config.subnet,
			dhcp.OptionRouter:           config.gateway,
			dhcp.OptionDomainNameServer: config.dns,
		}
		if config.domainName != "" {
			s.options[dhcp.OptionDomainName] = []byte(config.domainName)
		}
		log.Printf("serving as id %s, subnet %s gw %s dns %s domain %s\n",
			config.serverIP,
			config.subnet,
			config.gateway,
			config.dns,
			config.domainName,
		)
		s.serverIP = config.serverIP
		s.leaseDuration = config.leaseDuration
	default:
		fmt.Fprintf(w, "%v", s)

	}
}
