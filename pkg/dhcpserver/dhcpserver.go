package dhcpserver

import (
	"fmt"
	"log"
	"net"
	"time"

	dhcp "github.com/krolaw/dhcp4"
)

type DHCPLease struct {
	IPAddr   net.IP
	MACAddr  net.HardwareAddr
	Creation time.Time
}

type Server struct {
	serverIP      net.IP
	options       dhcp.Options
	leaseDuration time.Duration
	leases        []DHCPLease
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

func (s Server) findLease(req dhcp.Packet) (DHCPLease, error) {
	for _, lease := range s.leases {
		if lease.MACAddr.String() == req.CHAddr().String() {
			return lease, nil
		}
	}
	return DHCPLease{}, fmt.Errorf("No preassigned addr found on Leases database")
}

func (s Server) getEmptyAddrFromPool() net.IP {
	var result net.IP
	if len(s.leases) == 0 {
		return result
	} else {
		startAddr := s.leases[len(s.leases)-1].IPAddr
		result = startAddr
	}
	return result
}

func (s Server) createLease(req dhcp.Packet) DHCPLease {
	var lease DHCPLease
	lease.MACAddr = req.CHAddr()
	lease.Creation = time.Now()
	return lease
}

func (s Server) dhcpDiscover(req dhcp.Packet, options dhcp.Options) dhcp.Packet {

	lease, err := s.findLease(req)
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

func (s Server) dhcpRequest(req dhcp.Packet, options dhcp.Options) dhcp.Packet {

	if server, ok := options[dhcp.OptionServerIdentifier]; ok && !net.IP(server).Equal(s.serverIP) {
		log.Printf("message for a different server?")
		return nil
	}
	reqIP := net.IP(options[dhcp.OptionRequestedIPAddress])
	if reqIP == nil {
		reqIP = net.IP(req.CIAddr())
	}

	if len(reqIP) == 4 && !reqIP.Equal(net.IPv4zero) {
		lease, err := s.findLease(req)
		if err != nil {
			log.Printf("NAK to %s: %s\n", req.CHAddr().String(), err)
			return dhcp.ReplyPacket(req, dhcp.NAK, s.serverIP, nil, 0, nil)
		}
		if lease.IPAddr.String() != reqIP.String() {
			log.Printf("NAK to %s: expected %s, requested %s\n", req.CHAddr().String(), lease.IPAddr.String(), reqIP.String())
			return dhcp.ReplyPacket(req, dhcp.NAK, s.serverIP, nil, 0, nil)
		}
		return dhcp.ReplyPacket(req, dhcp.ACK, s.serverIP, reqIP, s.leaseDuration,
			s.options.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]))
	}
	return dhcp.ReplyPacket(req, dhcp.NAK, s.serverIP, nil, 0, nil)
}

// ServeDHCP handles incoming dhcp requests.
func (s Server) ServeDHCP(req dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) dhcp.Packet {

	switch msgType {
	case dhcp.Discover:
		return s.dhcpDiscover(req, options)
	case dhcp.Request:
		return s.dhcpRequest(req, options)
	}

	return nil
}
