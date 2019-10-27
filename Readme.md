# GÜAN

wan-controller is me trying to create a remote network controller from scratch for linux based devices. It is intended ti allow you to:

- Apply ACL rules
- Apply port-forwarding
- Manage BGP
- Create VXLAN tunnels
- Configure existing Interfaces
- Get information about CPU/Memory/Interface usage

## API Definition

```
[GET/PUT] Configuration
[GET/PUT] Configuration/Encryption
[GET/PUT] Configuration/Controllers
[GET/PUT] Configuration/Interfaces
[GET/PUT] Configuration/Routes
[GET/PUT] Configuration/Iptables
[GET/PUT] Configuration/NTPServers
[GET/PUT] Configuration/Services
```

## Activation Process

Router looks for `routerconfig.json` configuration file. If it is not present, it looks for `activation-config.json` each 10 seconds.

Once it is found, It applies configuration, stores it as `routerconfig.json` and sends hello message to the controllers. They answer with new encryption keys. Router applies them and restart.

## Configuration upgrade process and rollback.

Every hour router asks for the configuration to the controllers.
Controllers can also push new configuration to the routers at will.

When this happens, Router saves old configuration, applies new one and during 5 minutes it monitors connection to the controllers. If connection is lost, it performs a rollback.

## Monitorization

Every 5 min, router sends metrics to the controllers with memory, CPU, interface counters and disk metrics.

## VPP Configuration file:

```
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
  # poll-sleep 10
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

```
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
## References

[How to develop Go gRPC microservice](https://medium.com/@amsokol.com/tutorial-how-to-develop-go-grpc-microservice-with-http-rest-endpoint-middleware-kubernetes-daebb36a97e9)

[GoVPP](https://github.com/FDio/govpp)
