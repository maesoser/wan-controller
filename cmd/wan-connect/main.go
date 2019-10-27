package main

import (
	"github.com/maesoser/wan-agent/pkg/config"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
	"sync"
)

/*
                          +-----[wan-metrics]
                          |
            +---------|---+
<-- [SSL] --|  proxy  |---------[wan-controller]
            +---------|

*/

// SockAddr is the Unix socket created as a ggateway for the rest of güan processes
const (
	SocketAddr = "/etc/wan-data/wan-connector.sock"
	ConfigPath = "/etc/wan-data"
)

func proxyConn(conn net.Conn) {

	var routerConfig config.Config
	err := routerConfig.Load(ConfigPath)
	if err != nil {
		log.WithFields(log.Fields{"module": "wan-connect", "error": err.Error()}).Fatalln("Error reading config")
	}
	remoteAddr := routerConfig.GetController()

	rAddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
	if err != nil {
		panic(err)
	}

	remote, err := net.DialTCP("tcp", nil, rAddr)
	if err != nil {
		panic(err)
	}
	defer remote.Close()

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go copy(remote, conn, wg)
	go copy(conn, remote, wg)
	wg.Wait()
}

func copy(from, to net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	_, err := io.Copy(to, from)
	if err != nil {
		log.WithFields(log.Fields{"module": "wan-connect", "error": err.Error()}).Errorln("Error from copy")
		return
	}
}

func main() {

	if err := os.RemoveAll(SocketAddr); err != nil {
		log.WithFields(log.Fields{"module": "wan-connect", "error": err.Error()}).Errorln("Error removing old socket")
	}

	listener, err := net.Listen("unix", SocketAddr)
	if err != nil {
		log.WithFields(log.Fields{"module": "wan-connect", "error": err.Error()}).Fatalln("Error connecting to UNIX Socket")
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.WithFields(log.Fields{"module": "wan-connect", "error": err.Error()}).Fatalln("Error accepting incoming conn")
		}
		go proxyConn(conn)
	}

}
