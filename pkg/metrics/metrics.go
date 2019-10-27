package metrics

import (
	"encoding/json"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
	mnet "github.com/shirou/gopsutil/net"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"sync"
	"time"
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
	UUID     string        `json:"uuid"`
	Load     []float64     `json:"load"`
	Uptime   time.Duration `json:"upt"`
	MemTotal uint64        `json:"memtotal"`
	MemFree  uint64        `json:"memfree"`
	MemBuff  uint64        `json:"membuff"`
	Disks    []Filesystem  `json:"disks"`
	Ifaces   []Iface       `json:"ifaces"`
	mtx      sync.Mutex
}

func (m *Metric) Update() {
	m.UpdateSystem()
	m.UpdateInterfaces()
	m.UpdateFilesystems()
}

const (
	socketAddress = "/etc/wan-data/wan-connector.sock"
	kB            = 1024
)

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

func (m *Metric) UpdateInterfaces() {
	ifaces, err := mnet.IOCounters(true)
	if err != nil {
		log.WithFields(log.Fields{"module": "wan-metrics"}).Error("Error getting system ifaces" + err.Error())
	} else {
		for _, iface := range ifaces {
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
}

func (m *Metric) UpdateFilesystems() {
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
