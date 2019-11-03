package config

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

type Config struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	UUID        string        `json:"uuid"`
	DNSs        []string      `json:"dns"`
	Network     Network       `json:"network"`
	Encryption  EncryptConfig `json:"encryption"`
	Controllers []string      `json:"controllers"`
}

type Network struct {
	Name        string   `json:"name"`
	Description string   `json:"descr"`
	UUID        string   `json:"uuid"`
	Address     string   `json:"addr"`
	Mask        string   `json:"mask"`
	Gateway     string   `json:"gateway"`
	Uplink      Uplink   `json:"uplink"`
	Ports       []string `json:"ports"`
}

type Uplink struct {
	Name    string `json:"name"`
	Address string `json:"addr"`
	DHCP    bool   `json:"dhcp_enabled"`
}

func (c *Config) WriteDNS() error {
	f, err := os.OpenFile("/etc/resolv.conf", os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	_, err = fmt.Fprintf(w, "#Modified by WAN-AGENT\n")
	if err != nil {
		return err
	}
	for dns, _ := range c.DNSs {
		_, err := fmt.Fprintf(w, "nameserver %s\n", dns)
		if err != nil {
			return err
		}
	}
	w.Flush()
	return nil
}

func (c *Config) WriteHostname() error {
	f, err := os.OpenFile("/etc/hostname", os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	_, err = fmt.Fprintf(w, "%s\n", c.Name)
	if err != nil {
		return err
	}
	w.Flush()
	return nil
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

func (c *Config) Restore(filepath string) error {
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
	if len(c.Controllers) == 1 {
		return c.Controllers[0]
	}
	rand.Seed(time.Now().UnixNano())
	num := rand.Intn(len(c.Controllers) - 1)
	return c.Controllers[num]
}
