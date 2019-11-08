package dhcpserver

import (
	"log"
	"net"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
)

func handler(conn net.PacketConn, peer net.Addr, m *dhcpv4.DHCPv4) {
	// this function will just print the received DHCPv4 message, without replying
	log.Print(m.Summary())
}

func severDHCP() {
	laddr := net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 67,
	}
	server, err := server4.NewServer(&laddr, handler)
	if err != nil {
		log.Fatal(err)
	}
	// This never returns. If you want to do other stuff, dump it into a
	// goroutine.
	server.Serve()
}
