all: dependencies metrics agent

metrics:
	mkdir -p bin
	go vet cmd/wan-metrics/main.go
	go build -o bin/wan-metrics cmd/wan-metrics/main.go

dhcp:
	mkdir -p bin
	go vet cmd/wan-dhcp/main.go
	go build -o bin/wan-dhcp cmd/wan-dhcp/main.go

agent:
	binapi-generator --input-file=/usr/share/vpp/api/vpe.api.json --output-dir=binapi
	binapi-generator --input-file=/usr/share/vpp/api/interface.api.json --output-dir=binapi
	binapi-generator --input-file=/usr/share/vpp/api/l2.api.json --output-dir=binapi
	binapi-generator --input-file=/usr/share/vpp/api/dhcp.api.json --output-dir=binapi
	binapi-generator --input-file=/usr/share/vpp/api/tapv2.api.json --output-dir=binapi
	binapi-generator --input-file=/usr/share/vpp/api/nat.api.json --output-dir=binapi

	go vet cmd/wan-agent/main.go
	go build -o bin/wan-agent cmd/wan-agent/main.go

dependencies:
	go get github.com/shirou/gopsutil
	go get github.com/bennyscetbun/jsongo
	go get github.com/ftrvxmtrx/fd
	go get git.fd.io/govpp.git
	go get github.com/insomniacslk/dhcp
	go get github.com/vishvananda/netlink
	go install git.fd.io/govpp.git/cmd/binapi-generator

.PHONY: clean

clean:
	rm -f bin/wan-metrics
	rm -f bin/wan-agent
	rm -f bin/wan-dhcp
	rm -fr binapi/*
