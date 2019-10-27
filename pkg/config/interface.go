package config

type IfConfig struct {
	Name       string `json:"name"`
	Device     string `json:"device"`
	UUID       string `json:"uuid"`
	MAC        string `json:"macaddr"`
	EthtoolOps string `json:"ethops"`
	IPv6Metric uint32 `json:"ipv6metric"`
	IPv4Metric uint32 `json:"ipv4metric"`
	IPv6       bool   `json:"ipv6"`
	Type       string `json:"type"`
	Address    string `json:"address"`
	Mask       string `json:"mask"`
}
