# GÜAN

A [high performance](https://wiki.fd.io/view/VPP) remotely manageable and monitorizable home router based on [commodity hardware](https://www.amazon.es/Partaker-Firewall-Appliance-Mikrotik-Industrial/dp/B073FBDJYF/).

## Router Design

Router software has the following custom services enabled:
  - **wan-agent:** Manages the connnection with the controllers and configuration updates.
  - **wan-metrics:** Sends iface/cpu/memory stats to the controllers.
  - **wan-dhcp:** DHCP server for LAN-side hosts.

Besides that, some other software is in use on the router:
   - [**pihole:**](https://pi-hole.net) In order to offer add-free local DNS server

### Activation Process

`wan-agent` looks for `routerconfig.json` configuration file every 10 seconds. If a pendrive with this file is inserted, it is copied to the internal location where `wan-agent` expects to locate such file.

Once it is found, it applies that configuration, and sends hello message to the controllers. They answer with new encryption keys. `wan-agent` applies them and restart itself.

### VPP Configuration file:

```yaml
unix {
  nodaemon
  log /var/log/vpp/vpp.log
  full-coredump
  cli-listen /run/vpp/cli.sock
  gid vpp
  startup-config /etc/vpp/initial_setup.conf
  poll-sleep-usec 100
}

api-trace {
  on
}

api-segment {
  gid vpp
}

socksvr {
  default
}

cpu {
     # main-core 1
     # corelist-workers 2-3,18-19
     # skip-cores 4
     # workers 2
     # scheduler-policy fifo
     # scheduler-priority 50
}

dpdk {
  dev 0000:01:00.0 {
        name port1
  }
  dev 0000:02:00.0 {
        name port2
  }
  dev 0000:03:00.0 {
        name port3
  }
  dev 0000:04:00.0 {
        name port4
  }
  dev 0000:05:00.0 {
        name port5
  }
  dev 0000:06:00.0 {
        name port6
  }
}

plugins {
  plugin default { disable }
  plugin dpdk_plugin.so { enable }
  plugin acl_plugin.so { enable }
  plugin dhcp_plugin.so { enable }
  plugin dns_plugin.so { enable }
  plugin nat_plugin.so { enable }
}
```

### VPP Configuration steps for a common router on top of telco router:

```yaml
comment { configure wan port }
set int state port1 up
comment { set int mac address port1 7D:3F:69:54:b8:4C }
set dhcp client intfc port1 hostname vpprouter

comment { create IRB loopback interface }
loopback create
set int l2 bridge loop0 1 bvi
set int ip address loop0 192.168.2.1/24
set int state loop0 up

comment { add more ports to the IRB bridge group }
set int l2 bridge port2 1
set int state port2 up

set int l2 bridge port3 1
set int state port3 up

set int l2 bridge port4 1
set int state port4 up

set int l2 bridge port5 1
set int state port5 up

set int l2 bridge port6 1
set int state port6 up

comment { create tap iface for dhcp server and host-stack access }
create tap host-if-name lstack host-ip4-addr 192.168.2.2/24
set int l2 bridge tap0 1
set int state tap0 up

comment { configure nat }
nat44 add interface address port1
set interface nat44 in loop0 out port1

nat44 add static mapping local 192.168.2.2 22 external port1 22 tcp

add default linux route via 192.168.2.1
```

## Configuration upgrade process and rollback.

Every hour `wan-agent` asks for the configuration to the `wan-controller`. Controllers can also push new configuration to the routers at will.

When this happens, `wan-agent` executes the following steps:
  1. Save old configuration as backup.
  2. Applies new configuration.
  3. Waits 10 seconds for the configuration to be applied.
  4. During 50 seconds it monitors connection to the controllers as well as to other "health endpoints".
  5. If connection is lost, it performs a rollback. If it is not, it deletes backup configuration and accepts new one as the good one.

## Configuration file

Json/yaml configuration file:

```yaml
---
name: vppRouter
description: My Home Router
uuid:  f61ba3a5-012c-46d9-9f92-c95d02ddb5c0 
dns:
- 8.8.8.8
- 8.8.4.4
network:
  name: HomeNetwork
  description: My Home Network
  uuid:  5bfb30c8-e1e7-40b6-af43-a043f2eb3b20 
  addr: 192.168.2.0
  mask: 255.255.255.0
  gateway: 192.168.2.1
  uplink:
    name: port1
    address: 0.0.0.0
    dhcp_enabled: true
  ports:
  - port2
  - port3
  - port4
  - port5
encryption:
  cert: 'J7WBXT9AZHD2RPM3YLN732XO1WNLMZEHUMZJOZPEVD7YVUTYM493IKY9Z924XHRUD3870FTEKEQA'
  key: '2178VI4CAPBIZZ23R939E0N8VKMEC57ZOZTVE5Q8KLQBZ9PV316Y1GWB2NPVZT5ZITG0OJ5XEF69LMZEHUMZJOZPEV'
controllers:
- 192.168.0.76:6633
```

## Controller Design

Controller software is designed to be installed on a docker/kubernetes cluster. Controller pieces are basically three:
  - **wan-controller:** This controller offers an API in order to configure and get information about the connected routers. It also offers a prometheus endpoint to extract information about the routers.
  - **prometheus:** : It is used to store health information about routers connected to the controller
  - **grafana:** : It is used to graphically show information about the routers.

### API Definition

```
[GET/PUT] router
[GET/PUT] router/{ID}
[GET/PUT] router/{ID}/encryption
[GET/PUT] router/{ID}/controllers
[GET/PUT] router/{ID}/networks
```

## TODO

- [ ] Include custom NAT rules.
- [ ] Include [IPSEC](https://wiki.fd.io/view/VPP/IPSec_and_IKEv2)/[wireguard](https://www.wireguard.com) support for encrypted L3 traffic.
- [ ] Include pihole monitorization / configuration from the controller
- [ ] Include support for [PPPoE](https://docs.fd.io/vpp/17.10/clicmd_src_plugins_pppoe.html) uplink.
- [ ] Include support for multiple networks.
- [ ] Include support for dual uplink configuration.
- [ ] Add [IPFIX](https://wiki.fd.io/view/VPP/IPFIX) flow stats collection
- [ ] Add the possibility to remotely start, stop and configure containers

## How to install it

 1. Install CentOS
 2. Install VPP

# References

[How to develop Go gRPC microservice](https://medium.com/@amsokol.com/tutorial-how-to-develop-go-grpc-microservice-with-http-rest-endpoint-middleware-kubernetes-daebb36a97e9)

[GoVPP](https://github.com/FDio/govpp)

[Advance Command Execution in go](https://blog.kowalczyk.info/article/wOYk/advanced-command-execution-in-go-with-osexec.html)

[Simple netlink library for go](https://github.com/vishvananda/netlink)

