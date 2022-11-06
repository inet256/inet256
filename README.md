# INET256

![Matrix](https://img.shields.io/matrix/inet256:matrix.org?label=%23inet256%3Amatrix.org&logo=matrix)
[![GoDoc](https://godoc.org/github.com/inet256/inet256?status.svg)](http://godoc.org/github.com/inet256/inet256)
[<img src="https://discord.com/assets/cb48d2a8d4991281d7a6a95d2f58195e.svg" width="80">](https://discord.gg/TWy6aVWJ7f)

A 256 bit address space for peer-to-peer hosts/applications.

> **The value proposition**:
>
> All you have to know to send messages to another process is its address, which will never change.
>
> All you have to do to recieve messages is generate a private, public key pair and connect to the INET256 daemon

The INET256 API and Address Spec: [Spec](./doc/10_Spec.md)

The architecture of the reference implementation: [Architecture](./ARCHITECTURE.md)

Documentation for the daemon's config file: [Daemon Config](./doc/22_Daemon_Config.md)

## Features
- Stable addresses derived from public keys
- Secure communication to other nodes in the network
- Best-effort delivery like IP or UDP. At-most-once delivery, unlike IP and UDP.
- Messages are never corrupted. If it gets there, it's correct.
- Easy to add/remove/change routing algorithms.
- Addresses are plentiful. Spawn a new node for each process. Every process gets its own address, no need for ports.
- Daemon can run without root or `NET_ADMIN` capability.
- IPv6 Portal for IPv6 over INET256. Exposed as a TUN device. (requires `NET_ADMIN`).
- Autopeering and transport address discovery help make peering easy.

## Mesh256
The reference implementation of an INET256 Service, which lives in this repository, is called `mesh256`.
It uses a distributed routing algorithm to allow nodes on the network to discover paths between one another.

### Network Routing Protocols
The autoconfiguring, distributed routing algorithms of the sort required are under active research, and we don't want to couple mesh256 to any one algorithm as the state of the art could change rapidly.

Users are ultimately in control of which networks they participate in.
Networks can be selected in the configuration file.

We are eager to add other protocols.
Check out `networks/beaconnet` for an example of simple routing protocol. It's a good place to start.

## Utilities/Applications 
This project provides tools for using INET256 networks, some of which are not yet implemented

- [x] IPv6 Portal (TUN Device). Similar to CJDNS and Yggdrasil.
- [ ] IPv4 VPN, declarative mappings from INET256 -> IPv4. similar to WireGuard.
- [x] netcat.  Send newline separated messages to other nodes: `inet256 nc`.
- [x] echo. A server to echo messages back to the sender: `inet256 echo`.
- [x] Central Discovery Server to keep track of roaming peers. `inet256 central-discovery`

## Code Tour
- `pkg/inet256` API definitions.  Mostly things required by the spec.

- `pkg/mesh256` The reference implementation of an INET256 Service. 

- `pkg/inet256d` The daemon that manages setting up transports, autopeering, discovery, the actual INET256 service, and the gRPC API.

- `pkg/inet256ipv6` Logic for bridging INET256 to IPv6. Includes the IPv6 portal.

- `pkg/inet256test` A test suite for Network implementations.

- `pkg/autopeering` Autopeering interface definition and implementations

- `pkg/discovery` discovery interface definition and implementations

- `networks/` Network implementations, routing logic is in these.

- `client/` Client implementations, these connect to the daemon.

## License
Code in this repository is by default licensed under the GPL as defined in `LICENSE`.
Some of the sub-directories contain their own `LICENSE` files for the LGPL, or MPL as defined therein.
That license applies to the sub-directory.

In summary: you should be able to import an INET256 client in a language of your choice and do whatever you want.
But other than clients, the implementation is strongly copyleft.
