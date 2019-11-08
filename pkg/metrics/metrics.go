package metrics

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
	mnet "github.com/shirou/gopsutil/net"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	// "git.fd.io/govpp.git/adapter"
	"git.fd.io/govpp.git/adapter/statsclient"
	"git.fd.io/govpp.git/api"
	"git.fd.io/govpp.git/core"
)

type Filesystem struct {
	Mountpoint string `json:"mount"`
	Size       uint64 `json:"size"`
	Free       uint64 `json:"free"`
	Device     string `json:"dev"`
}

type Iface struct {
	Name      string `json:"name"`
	TxBytes   uint64 `json:"txbytes"`
	TxPackets uint64 `json:"txpkt"`
	TxErrors  uint64 `json:"txerr"`
	TxDropped uint64 `json:"txdrop"`
	RxBytes   uint64 `json:"rxbytes"`
	RxPackets uint64 `json:"rxpkt"`
	RxErrors  uint64 `json:"rxerr"`
	RxDropped uint64 `json:"rxdrop"`
}

type Metric struct {
	UUID          string        `json:"uuid"`
	Load          []float64     `json:"load"`
	Uptime        time.Duration `json:"upt"`
	MemTotal      uint64        `json:"memtotal"`
	MemFree       uint64        `json:"memfree"`
	MemBuff       uint64        `json:"membuff"`
	Disks         []Filesystem  `json:"disks"`
	Ifaces        []Iface       `json:"ifaces"`
	mtx           sync.Mutex
	vppClient     *statsclient.StatsClient
	vppConnection *core.StatsConnection
}

func (m *Metric) Update() {
	m.UpdateSystem()
	m.UpdateInterfaces()
	m.UpdateFilesystems()
}

const (
	kB = 1024
)

func (m *Metric) Init() {
	var err error
	m.vppClient = statsclient.NewStatsClient(statsclient.DefaultSocketName)
	m.vppConnection, err = core.ConnectStats(m.vppClient)
	if err != nil {
		log.WithFields(log.Fields{"module": "wan-metrics", "error": err.Error()}).Errorln("Error connecting to VPP Stats Endpoint")
	}
}

func (m *Metric) Disconnect() {
	m.vppClient.Disconnect()
}

func (m *Metric) UpdateSystem() {
	si := &unix.Sysinfo_t{}

	err := unix.Sysinfo(si)
	if err != nil {
		log.WithFields(log.Fields{"module": "wan-metrics"}).Error("Error at syscall.Sysinfo:" + err.Error())
	}
	// scale := 65536.0 // magic

	defer m.mtx.Unlock()
	m.mtx.Lock()

	unit := uint64(si.Unit) * kB

	m.Uptime = time.Duration(si.Uptime) * time.Second
	m.MemTotal = uint64(si.Totalram) / unit
	m.MemFree = uint64(si.Freeram) / unit
	m.MemBuff = uint64(si.Bufferram) / unit
	m.Load = make([]float64, 3)
	loads, err := load.Avg()
	if err != nil {
		log.WithFields(log.Fields{"module": "wan-metrics"}).Error("Error getting system load:" + err.Error())
	} else {
		m.Load[0] = loads.Load1
		m.Load[1] = loads.Load5
		m.Load[2] = loads.Load15
	}
}

func (m *Metric) StringSystem() string {
	out := ""
	out += fmt.Sprintf("guan_uptime_sec{uuid=\"%s\"} %f\n", m.UUID, m.Uptime)
	out += fmt.Sprintf("guan_total_mem_kb{uuid=\"%s\"} %f\n", m.UUID, m.MemTotal)
	out += fmt.Sprintf("guan_free_mem_kb{uuid=\"%s\"} %f\n", m.UUID, m.MemFree)
	out += fmt.Sprintf("guan_buff_mem_kb{uuid=\"%s\"} %f\n", m.UUID, m.MemBuff)
	out += fmt.Sprintf("guan_load_1min{uuid=\"%s\"} %f\n", m.UUID, m.Load[0])
	out += fmt.Sprintf("guan_load_5min{uuid=\"%s\"} %f\n", m.UUID, m.Load[1])
	out += fmt.Sprintf("guan_load_15min{uuid=\"%s\"} %f\n", m.UUID, m.Load[2])
	return out
}

func (m *Metric) UpdateInterfaces() {
	m.Ifaces = nil
	m.UpdateUnixInterfaces()
	m.UpdateDPDKInterfaces()
}

func (m *Metric) UpdateUnixInterfaces() {
	ifaces, err := mnet.IOCounters(true)
	if err != nil {
		log.Fatalln("Connecting failed:", err)
	}
	defer c.Disconnect()
}
func (m *Metric) UpdateInterfaces() {

	fmt.Println("Listing interface stats..")
	stats := new(api.InterfaceStats)
	if err := c.GetInterfaceStats(stats); err != nil {
		log.Fatalln("getting interface stats failed:", err)
	}
	for _, iface := range stats.Interfaces {
		fmt.Printf(" - %+v\n", iface)

		var newIface Iface
		newIface.Name = iface.Name
		newIface.RxBytes = iface.BytesRecv
		newIface.TxBytes = iface.BytesSent
		newIface.RxDropped = iface.Dropin
		newIface.TxDropped = iface.Dropout
		newIface.RxPackets = iface.PacketsRecv
		newIface.TxPackets = iface.PacketsSent
		newIface.TxErrors = iface.Errout
		newIface.RxErrors = iface.Errin
		m.Ifaces = append(m.Ifaces, newIface)

	}
}

func (m *Metric) UpdateDPDKInterfaces() {
	stats := new(api.InterfaceStats)
	if err := m.vppConnection.GetInterfaceStats(stats); err != nil {
		log.WithFields(log.Fields{"module": "wan-metrics", "error": err.Error()}).Errorln("Error getting DPDK interface stats")
	}
	for _, iface := range stats.Interfaces {
		var newIface Iface
		// newIface.Name = fmt.Sprintf("port%d", iface.InterfaceIndex)
		newIface.Name = iface.InterfaceName
		newIface.RxBytes = iface.Rx.Bytes
		newIface.TxBytes = iface.Tx.Bytes
		newIface.RxDropped = iface.Drops
		newIface.TxDropped = iface.Drops
		newIface.RxPackets = iface.Rx.Packets
		newIface.TxPackets = iface.Tx.Packets
		newIface.TxErrors = iface.TxErrors
		newIface.RxErrors = iface.RxErrors
		m.Ifaces = append(m.Ifaces, newIface)
	}
}

func (m *Metric) StringInterfaces() string {
	out := ""
	for _, iface := range m.Ifaces {
		out += fmt.Sprintf("guan_rx_bytes{uuid=\"%s\",name=\"%s\"\n", m.UUID, iface.Name, iface.RxBytes)
		out += fmt.Sprintf("guan_tx_bytes{uuid=\"%s\",name=\"%s\"\n", m.UUID, iface.Name, iface.TxBytes)
		out += fmt.Sprintf("guan_rx_drop_bytes{uuid=\"%s\",name=\"%s\"\n", m.UUID, iface.Name, iface.RxDropped)
		out += fmt.Sprintf("guan_tx_drop_bytes{uuid=\"%s\",name=\"%s\"\n", m.UUID, iface.Name, iface.TxDropped)
		out += fmt.Sprintf("guan_rx_pkt{uuid=\"%s\",name=\"%s\"\n", m.UUID, iface.Name, iface.RxPackets)
		out += fmt.Sprintf("guan_tx_pkt{uuid=\"%s\",name=\"%s\"\n", m.UUID, iface.Name, iface.TxPackets)
		out += fmt.Sprintf("guan_rx_error{uuid=\"%s\",name=\"%s\"\n", m.UUID, iface.Name, iface.RxErrors)
		out += fmt.Sprintf("guan_tx_error{uuid=\"%s\",name=\"%s\"\n", m.UUID, iface.Name, iface.TxErrors)

	}
	return out
}

func (m *Metric) UpdateFilesystems() {
	m.Disks = nil
	partitions, err := disk.Partitions(false)
	if err != nil {
		log.WithFields(log.Fields{"module": "wan-metrics", "error": err.Error()}).Errorln("Error getting disk data:")
	} else {
		for _, partition := range partitions {
			var newFS Filesystem
			newFS.Device = partition.Device
			newFS.Mountpoint = partition.Mountpoint
			stats, err := disk.Usage(newFS.Mountpoint)
			if err != nil {
				log.WithFields(log.Fields{"module": "wan-metrics"}).Error("Error getting disk data:" + err.Error())
			} else {
				newFS.Free = stats.Free
				newFS.Size = stats.Total
			}
			m.Disks = append(m.Disks, newFS)
		}
	}
}

func (m *Metric) StringFilesystems() string {
	out := ""
	for _, fs := range m.Disks {
		out += fmt.Sprintf("guan_free_disk_sec{uuid=\"%s\",dev=\"%s\",mount=\"%s\",} %f\n", m.UUID, fs.Device, fs.Mountpoint, fs.Free)
		out += fmt.Sprintf("guan_size_disk_sec{uuid=\"%s\",dev=\"%s\",mount=\"%s\",} %f\n", m.UUID, fs.Device, fs.Mountpoint, fs.Size)
	}
	return out
}

func (m *Metric) LogSystem() {
	defer m.mtx.Unlock()
	m.mtx.Lock()
	log.WithFields(log.Fields{"module": "wan-metrics"}).Infof("Uptime: %v  Load: %v  Mem: %d/%d kB", m.Uptime, m.Load, m.MemFree, m.MemTotal)
	for _, fs := range m.Disks {
		log.WithFields(log.Fields{"module": "wan-metrics"}).Infof("Fs: %s -> %s  Free: %d/%d kB", fs.Device, fs.Mountpoint, fs.Free, fs.Size)
	}
	for _, iface := range m.Ifaces {
		log.WithFields(log.Fields{"module": "wan-metrics"}).Infof("Net: %s  TX: %d  RX: %d kB", iface.Name, iface.TxBytes, iface.RxBytes)
	}
}

func (m *Metric) Data() ([]byte, error) {
	defer m.mtx.Unlock()
	m.mtx.Lock()

	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return data, nil
}
