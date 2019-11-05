package vppmgr

import (
	"errors"
	"fmt"
	"git.fd.io/govpp.git"
	"git.fd.io/govpp.git/api"
	"git.fd.io/govpp.git/core"
	"github.com/maesoser/wan-controller/binapi/dhcp"
	"github.com/maesoser/wan-controller/binapi/interfaces"
	"github.com/maesoser/wan-controller/binapi/l2"
	"github.com/maesoser/wan-controller/binapi/vpe"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

type VPPManager struct {
	VPPConn  *core.Connection
	VPPChann api.Channel
}

func (v *VPPManager) Init() error {
	var err error
	v.VPPConn, err = govpp.Connect("")
	if err != nil {
		return err
	}

	v.VPPChann, err = v.VPPConn.NewAPIChannel()
	if err != nil {
		return err
	}
	return nil
}

func (v *VPPManager) Close() {
	v.VPPChann.Close()
	v.VPPConn.Disconnect()
}

func (v *VPPManager) DumpIfaces() {
	req := &interfaces.SwInterfaceDump{}
	reqCtx := v.VPPChann.SendMultiRequest(req)
	for {
		msg := &interfaces.SwInterfaceDetails{}
		stop, err := reqCtx.ReceiveReply(msg)
		if stop {
			break
		}
		if err != nil {
			log.WithFields(log.Fields{"module": "vpp-mgr"}).Error(err)
		}
		fmt.Printf("%v: %v\n", string(msg.InterfaceName[:]), msg.L2Address)
	}
}

func (v *VPPManager) DumpBridges() {
	req := &l2.BridgeDomainDump{
		BdID: ^uint32(0),
	}
	reqCtx := v.VPPChann.SendMultiRequest(req)

	for {
		msg := &l2.BridgeDomainDetails{}
		stop, err := reqCtx.ReceiveReply(msg)
		if stop {
			break
		}
		if err != nil {
			log.WithFields(log.Fields{"module": "vpp-mgr"}).Error(err)
		}
		fmt.Printf("\tBridge domain, message id: bridge_domain_details, bd index: %v\n", msg.BdID)
	}
}

func (v *VPPManager) GetIfIndexByName(ifname string) (interfaces.InterfaceIndex, error) {
	req := &interfaces.SwInterfaceDump{}
	reqCtx := v.VPPChann.SendMultiRequest(req)
	for {
		msg := &interfaces.SwInterfaceDetails{}
		stop, err := reqCtx.ReceiveReply(msg)
		if stop {
			break
		}
		if err != nil {
			return 0, err
		}
		if string(msg.InterfaceName[:]) == ifname {
			return msg.SwIfIndex, nil
		}
	}
	return 0, errors.New("Interface not found")
}

func (v *VPPManager) AddLoopback() (interfaces.InterfaceIndex, error) {
	req := &interfaces.CreateLoopback{}
	reply := &interfaces.CreateLoopbackReply{}
	if err := v.VPPChann.SendRequest(req).ReceiveReply(reply); err != nil {
		return 0, err
	}
	return reply.SwIfIndex, nil
}

func (v *VPPManager) AddBridge() {
	req := &l2.BridgeDomainAddDel{
		BdID:  99,
		IsAdd: 1,
	}
	reply := &l2.BridgeDomainAddDelReply{}

	if err := v.VPPChann.SendRequest(req).ReceiveReply(reply); err != nil {
		log.WithFields(log.Fields{"module": "vpp-mgr"}).Error(err)
		return
	}

}
func StringtoAddr(IPAddress string) (interfaces.IP4Address, error) {
	var output [4]uint8
	var err error
	for i, a := range strings.Split(IPAddress, ".") {
		var n uint64
		n, err = strconv.ParseUint(a, 10, 8)
		output[i] = uint8(n)
		if err != nil {
			return output, err
		}
	}
	return output, nil

}

func (v *VPPManager) AddIfaceAddress(ifindex interfaces.InterfaceIndex, IPAddress string) error {
	ipv4Addr, err := StringtoAddr(IPAddress)
	if err != nil {
		return err
	}
	req := &interfaces.SwInterfaceAddDelAddress{
		SwIfIndex: ifindex,
		IsAdd:     true,
		Prefix: interfaces.AddressWithPrefix{
			Address: interfaces.Address{Af: interfaces.ADDRESS_IP4, Un: interfaces.AddressUnionIP4(ipv4Addr)},
			Len:     24,
		},
	}
	reply := &interfaces.SwInterfaceAddDelAddressReply{}

	if err := v.VPPChann.SendRequest(req).ReceiveReply(reply); err != nil {
		return err
	}
	fmt.Printf("reply: %+v\n", reply)
	return nil
}

func (v *VPPManager) AddDHCP(ifindex interfaces.InterfaceIndex, hostname string) error {
	req := &dhcp.DHCPClientConfig{
		IsAdd: true,
		Client: dhcp.DHCPClient{
			SwIfIndex: dhcp.InterfaceIndex(ifindex),
			Hostname:  hostname,
		},
	}
	reply := &dhcp.DHCPClientConfigReply{}
	if err := v.VPPChann.SendRequest(req).ReceiveReply(reply); err != nil {
		return err
	}
	fmt.Printf("reply: %+v\n", reply)
	return nil
}

func (v *VPPManager) IfaceUp(ifindex interfaces.InterfaceIndex) error {
	req := &interfaces.SwInterfaceSetFlags{
		SwIfIndex: ifindex,
		Flags:     interfaces.IF_STATUS_API_FLAG_LINK_UP,
	}
	reply := &interfaces.SwInterfaceSetFlagsReply{}

	if err := v.VPPChann.SendRequest(req).ReceiveReply(reply); err != nil {
		return err
	}
	fmt.Printf("reply: %+v\n", reply)
	return nil
}

func (v *VPPManager) vppVersion() {
	req := &vpe.ShowVersion{}
	reply := &vpe.ShowVersionReply{}

	if err := v.VPPChann.SendRequest(req).ReceiveReply(reply); err != nil {
		log.WithFields(log.Fields{"module": "wan-agent", "error": err.Error()}).Warnln("Unable to retrieve VPP Daemon version")
		return
	}
	log.WithFields(log.Fields{"module": "wan-agent"}).Infof("Connected to VPP Daemon ver %q", reply.Version)
}
