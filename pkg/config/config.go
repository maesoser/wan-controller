package config

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"math/rand"
	"os"
	"time"
)

type Config struct {
	Name        string           `json:"name"`
	Description string           `json:"descr"`
	UUID        string           `json:"uuid"`
	DNSServers  []string         `json:"dns"`
	IPv4Tables  []string         `json:"ipv4tables"`
	IPv6Tables  []string         `json:"ipv6tables"`
	IPv4Routes  []string         `json:"ipv4routes"`
	IPv6Routes  []string         `json:"ipv6routes"`
	Encryption  EncryptConfig    `json:"encryption"`
	Controllers []string         `json:"controllers"`
	Services    []ServicesConfig `json:"services"`
	LANPorts    []AccessPort     `json:"lan_ports"`
	WANPorts    []NetworkPort    `json:"wan_ports"`
}

func (c *Config) Update(newConfig Config) {
	if c.Checksum() == newConfig.Checksum() {
		log.WithFields(log.Fields{"module": "config-mgr"}).Warn("Configuration is the same, skipping")
		return
	}
}

func (c *Config) Load(filepath string) error {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, c)
	return err
}

func (c *Config) Save(filepath string) (int, error) {

	buffer := new(bytes.Buffer)
	encoder := json.NewEncoder(buffer)
	encoder.SetIndent("", "\t")

	err := encoder.Encode(c)
	if err != nil {
		return 0, err
	}
	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return 0, err
	}
	n, err := file.Write(buffer.Bytes())
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (c *Config) Backup(filepath string) (int, error) {
	return c.Save(filepath + ".bck")
}

func (c *Config) Restore(filepath string) (int, error) {
	return c.Load(filepath + ".bck")
}

func (c *Config) Checksum() [16]byte {
	data, err := json.Marshal(c)
	if err != nil {
		log.WithFields(log.Fields{"module": "config-mgr"}).Error("Starting wan-agent")
	}
	return md5.Sum(data)
}

func (c *Config) GetController() string {
	rand.Seed(time.Now().UnixNano())
	num := rand.Intn(len(c.Controllers) - 1)
	return c.Controllers[num]
}
