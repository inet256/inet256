# INET256 Architecture

This document describes the architecture of the reference implementation of an INET256 service.
Most of what is disussed here are implementation details, specific to this implementation.
Also take a look at the [API Spec](./doc/10_Spec.md) to see what is required vs incidental.

## Mesh256
The reference implementation is provided as a library called [`mesh256`](./pkg/mesh256/README.md).
The library can be used to construct mesh networks on user-defined transports, with arbitrary routing algorithms.
This makes it very easy to turn a distributed routing algorithm into an INET256 network.
Or to run Mesh256 on top of a new transport.

## HTTP API
The API presented to clients of INET256 is based on the `Node` type in `pkg/inet256/`

Client implementations can be found in `client/`

Clients connect to the daemon in `pkg/inet256d` over the HTTP API in `pkg/inet256http`.
When a client connects, the daemon sets up a virtual node for the client.
This is a new node in the network with a distinct address derived from the clients public key. Every process that connects to INET256 gets its own stable address, which cryptographically represents the process's identity.

## Helper Services
There are services that run in the background to help with establishing connections to peers.

### Discovery
Discovery in INET256 refers to the process of finding addresses for a known peer that you want to connect to.
This could be over udp, ethernet, on the local network, or over the internet.
As long as there is a Swarm which can send to the address it can be used to contact the peer.

### Autopeering
Autopeering refers to finding nodes to peer with that the user did not explicitly name by ID.
This could mean getting a dynamic list from a trusted source, or it could mean connecting to random people on the internet.
This will always be off by default.

Autopeering works by adding and removing peers from the `PeerSet` that is passed to the networks.

Look at `pkg/autopeering` for the common interface implemented by autopeering services.
