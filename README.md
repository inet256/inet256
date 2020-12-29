# INET256

A unified 256 bit address space for peer-to-peer hosts/applications.

> **The value proposition**:
>
> All you have to do to send messages to another process is know its address (hash of public key).
>
> All you have to do to recieve messages is generate a private, public key pair and connect to the inet256 daemon

## Background
There are now a few projects/networks which use the hash of a public key, as a mechanism for assigning addresses.

- [CJDNS](https://github.com/cjdelisle/cjdns)
SHA512 of RSA Key
- [Yggdrasil](https://github.com/yggdrasil-network/yggdrasil-go)
SHA512 of Ed25519
- [Tor](https://www.torproject.org/)
Raw Ed25519 (among other schemes)
- [IPFS](https://github.com/ipfs/go-ipfs)
Multihash of Ed25519

They are all using different hash functions, and key serialization functions.
The space would benefit from some standardization, in particular if users who cared less about the network/routing and more about services had one daemon to install that gave them access to the whole address space that would be good for all the networks.

Creating a single target for application and service developers will hopefully inspire more development, and nudge these networks to assimilate their address schemes.

INET256 aims to provide tooling in the form of NATs, DHCPv6, TUN devices, VPNs, secure transports, and discovery services, which can be leveraged for all compatible networks.

## Spec
Addresses are determined by the `SHA3-256` of the `PKIX ASN.1 DER` encoding of the public key.

More about that design decision in `docs/`

## Building Blocks (Help From Below)

This project encourages use of primitives from the message based p2p networking library [go-p2p](https://github.com/brendoncarroll/go-p2p)

Its `Swarm` abstraction provides transports for INET256 networks.
The library also provides discovery services, and methods for serializing and creating INET256 addresses.
Notably, the discovery services and NAT management provide a way for users to link all their devices to INET256 without touching a VPS or DNS entry, or forwarding a port.
It models friend-to-friend communication, and unreliable messages well, making it better suited for overlay networks than, for example: libp2p, which is more suited for public networks, with reliable communication.

## Tooling (Help From Above)
This project will provide tools for using INET256 networks, some of which are not yet implemented

- [x] INET256 to IPv6 mapping. Inspired by Yggdrasil
- [ ] A TUN device using the mapping.
- [ ] NAT Table using the mapping. No port mappings Layer 3 only.
- [ ] DHCPv6 server which gives out addresses corresponding to virtual nodes.
- [ ] IPv4 VPN, declarative mappings from INET256 -> IPv4. similar to wireguard.

## Use Cases

#### I Have a Network Protocol. How do I start a Node?
Assuming you have a package `mynetwork` which contains an INET256 network factory, the entrypoint would look something like this:

```go
// cmd/mynetwork/main.go

package main

import (
    "log"

    "github.com/inet256/inet256/pkg/inet256cmd"
    "mynetwork.org/mynetwork"
)

func main() {
    inet256cmd.Register("mynetwork", mynetwork.Factory)
    if err := inet256cmd.Execute(); err != nil {
        log.Fatal(err)
    }
}
```

## License
Code in this repository is by default licensed under the GPL as defined in `LICENSE`.
Some of the sub-directories contain their own `LICENSE` files for the LGPL as defined therein.
That license applies to the sub-tree.

In summary: you should be able to import an inet256 client in a language of your choice and do whatever you want.
But other than clients, the implementation is strongly copyleft.
