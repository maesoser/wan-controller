package routemgr

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

func AddRoute(network, nexthop string) error {
	cmd := exec.Command("/usr/sbin/ip", "route", "add", network, "via", nexthop)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.WithFields(log.Fields{"module": "config-mgr"}).Error("cmd.Run() failed with %s\n", err)
	}
	log.WithFields(log.Fields{"module": "config-mgr"}).Info(string(out))
}

func AddDefaultRoute(nexthop string) error {
	AddRoute("default", nexthop)
}

func ListRoutes() error {
	cmd := exec.Command("/usr/sbin/ip", "route", "show")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.WithFields(log.Fields{"module": "config-mgr"}).Error("cmd.Run() failed with %s\n", err)
	}
	log.WithFields(log.Fields{"module": "config-mgr"}).Info(string(out))
}
