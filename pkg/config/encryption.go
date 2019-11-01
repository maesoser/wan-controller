package config

import (
	"bytes"
	"os"
	"encoding/json"
)

type EncryptConfig struct {
	Certificate string `json:"cert"`
	Key         string `json:"key"`
}

func (c *EncryptConfig) SaveToFile(filepath string) (int, error) {

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
	n, err := file.Write([]byte(c.Certificate))
	if err != nil {
		return 0, err
	}
	return n, nil
}
