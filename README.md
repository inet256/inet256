# INET256

A unified 256 bit address space for peer-to-peer hosts.

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

INET256 aims to provide tooling in the form of NATs, DHCPv6, TUN devices, VPNs, secure transports, and trackers, which can be leveraged for all compatible networks.

## Spec
Addresses are determined by the `SHA3-256` of the `PKIX ASN.1 DER` encoding of the public key.

More about that design decision in `docs/`

## Building Blocks (Help From Below)

This project encourages use of primitives from the message based p2p networking library [go-p2p](https://github.com/brendoncarroll/go-p2p)

Its `Swarm` abstraction provides transports for INET256 networks.
The library also provides discovery services, and methods for serializing and creating INET256 addresses.
Notably, the discovey services and NAT management provide a way for users to link all their devices to INET256 without touching a VPS or DNS entry, or forwarding a port.
It models friend-to-friend communication, and unreliable messages well, making it better suited for overlay networks than, for example: libp2p, which is more suited for public networks, with reliable communication.

## Tooling (Help From Above)
This project will provide tools for using INET256 networks, some of which is not yet implemented

- [x] INET256 to IPv6 mapping. Inspired by Yggdrasil
- [ ] A TUN device using the mapping.
- [ ] NAT Table using the mapping. No port mappings Layer 3 only.
- [ ] DHCPv6 server which gives out addresses corresponding to virtual nodes.
- [ ] IPv4 VPN, declarative mappings from INET256 -> IPv4. similar to wireguard.
