# INET256

![Matrix](https://img.shields.io/matrix/inet256:matrix.org?label=%23inet256%3Amatrix.org&logo=matrix)
[![GoDoc](https://godoc.org/github.com/inet256/inet256?status.svg)](http://godoc.org/github.com/inet256/inet256)

A 256 bit address space for peer-to-peer hosts/applications.

> **The value proposition**:
>
> All you have to do to send messages to another process is know its address, which will never change.
>
> All you have to do to recieve messages is generate a private, public key pair and connect to the inet256 daemon

[Architecture](./ARCHITECTURE.md)

[Spec](./docs/10_Spec.md)

## Features
- Stable addresses derived from public keys
- Secure communication to other nodes in the network
- Best-effort delivery like IP or UDP. At-most-once delivery, unlike IP and UDP.
- Messages are never corrupted. If it gets there, it's correct.
- Easy to add/remove/change routing algorithms.
- Addresses are plentiful. Spawn a new node for each process. Every process gets its own address, no need for ports.
- IPv6 Portal for IPv6 over INET256. Exposed as a TUN device.
- Autopeering and transport address discovery help make peering easy.

## Network Routing Protocols
This project separates a modern communication API from the routing algorithm that powers it.
The autoconfiguring, distributed routing algorithms of the sort required are very much under active research, and we don't want to couple INET256 to any one algorithm as the state of the art could change rapidly.

Users are ultimately in control of which networks they participate in.
Networks can be selected in the configuration file.

We are eager to add other protocols.
Check out `networks/floodnet` for an example of a naive flooding protocol.
It's a good place to start.

## What is Provided
### Building Blocks (Help From Below)

This project encourages use of primitives from the message based p2p networking library [go-p2p](https://github.com/brendoncarroll/go-p2p)

Its `Swarm` abstraction provides transports for INET256 networks.
The library also provides discovery services, and methods for serializing and creating INET256 addresses.
Notably, the discovery services and NAT management provide a way for users to link all their devices to INET256 without touching a VPS or DNS entry, or forwarding a port.
It models friend-to-friend communication, and unreliable messages well, making it better suited for overlay networks than, for example: libp2p, which is more suited for public networks, with reliable communication.

### Utilities/Applications (Help From Above)
This project will provide tools for using INET256 networks, some of which are not yet implemented

- [x] IPv6 Portal (TUN Device). Similar to CJDNS and Yggdrasil.
- [ ] NAT Table from IPv6 to INET256. No port mappings, layer 3 only.
- [ ] DHCPv6 server which gives out addresses corresponding to virtual nodes.
- [ ] IPv4 VPN, declarative mappings from INET256 -> IPv4. similar to WireGuard.


## Code Tour
- `pkg/inet256` API definitions.  Mostly things required by the spec.

- `pkg/inet256srv` The reference implementation of an INET256 Service. 

- `pkg/inet256d` The daemon that manages setting up transports, autopeering, discovery, the actual INET256 service, and the gRPC API.

- `pkg/inet256ipv6` Logic for bridging INET256 to IPv6. Includes the IPv6 portal.

- `pkg/inet256test` A test suite for Network implementations.

- `networks/` Network implementations, routing logic is in these.

- `client/` Client implementations, these connect to the daemon.

## License
Code in this repository is by default licensed under the GPL as defined in `LICENSE`.
Some of the sub-directories contain their own `LICENSE` files for the LGPL, or MPL as defined therein.
That license applies to the sub-tree.

In summary: you should be able to import an inet256 client in a language of your choice and do whatever you want.
But other than clients, the implementation is strongly copyleft.
