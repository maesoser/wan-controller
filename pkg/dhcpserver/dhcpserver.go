package dhcpserver

import (
	"log"
	"net"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
)

// https://en.wikipedia.org/wiki/Dynamic_Host_Configuration_Protocol

const (
	moduleName = "wan-dhcp"
)

func handler(conn net.PacketConn, peer net.Addr, m *dhcpv4.DHCPv4) {
	if m == nil {
		log.WithFields(log.Fields{"module": moduleName}).Error("Packet is empty")
		return
	}
	log.Print(m.Summary())

	reply, err := dhcpv4.NewReplyFromRequest(m)
	if err != nil {
		log.WithFields(log.Fields{"module": moduleName}).Error("NewReplyFromRequest failed: %v", err)
		return
	}
	reply.UpdateOption(dhcpv4.OptServerIdentifier(net.IP{1, 2, 3, 4}))
	switch mt := m.MessageType(); mt {
	case dhcpv4.MessageTypeDiscover:
		reply.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer))
	case dhcpv4.MessageTypeRequest:
		reply.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))
	default:
		log.WithFields(log.Fields{"module": moduleName}).Error("Unhandled message type: %v", mt)
		return
	}

	if _, err := conn.WriteTo(reply.ToBytes(), peer); err != nil {
		log.WithFields(log.Fields{"module": moduleName}).Error("Cannot reply to client: %v", err)
	}
}

func Start() {
	laddr := net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: 67,
	}
	server, err := server4.NewServer("lstack", &laddr, handler, nil)
	if err != nil {
		log.Fatal(err)
	}
	server.Serve()
}
