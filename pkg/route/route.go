package route

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
)

func ListRoutes() {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		log.Fatal(err)
	}
	for _, route := range routes {
		iface, err := net.InterfaceByIndex(route.LinkIndex)
		if err != nil {
			log.Println(err)
		}
		if route.Src == nil && route.Dst == nil && route.Gw != nil {
			log.WithFields(log.Fields{"module": "route-mgr"}).Info("default via %v dev %s", route.Gw, iface.Name)
		} else {
			log.WithFields(log.Fields{"module": "route-mgr"}).Info("%v via %v dev %s", route.Src, route.Dst, iface.Name)
		}
	}
}

func AddDefaultRoute(gw net.IP) error {
	defaultRoute := netlink.Route{
		Dst: nil,
		Gw:  gw,
	}
	if err := netlink.RouteAdd(&defaultRoute); err != nil {
		return err
	}
	return nil
}

func DelDefaultRoute(gw net.IP) error {
	defaultRoute := netlink.Route{
		Dst: nil,
		Gw:  gw,
	}
	if err := netlink.RouteDel(&defaultRoute); err != nil {
		return err
	}
	return nil
}

func CheckDefaultGatewayRoute(gw net.IP) error {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		log.Fatal(err)
	}
	for _, route := range routes {
		if route.Src == nil && route.Dst == nil && route.Gw.Equal(gw) {
			return nil
		}
	}
	return fmt.Errorf("Defult route with gwateway %s not installed", gw.String())
}
