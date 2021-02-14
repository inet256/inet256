# INET256 Architecture

INET256 is built around `Swarms` from the `go-p2p` library.

Networks (routing algorithms) are given a private key, a `Swarm` and a `PeerSet`.

The `PeerSet` is just a list of peers that are accessible through the swarm.
These are the one-hop peers.
It is the job the Network to figure out how to route messages to more peers than just those in this set.

## Stack
An INET256 Node's stack looks like this, implemented as layered `Swarms`

```
                Application Data :-)
|-----------------------------------------------|
|              Encryption Layer (NPF)           |
|-----------------------------------------------|
| Network Selection                             |
|-----------------------------------------------|
| Routing Algorithms:                           |
|                |                |             |
|    Kad + SR    |    Network 2   |   Network 3 |
|                |                |             |
|-----------------------------------------------|
|         Multiplexing & Packet Aggregation     |
|-----------------------------------------------|
|              Encryption Layer (NPF)           |
|-----------------------------------------------|
| Transports:                                   |
|       UDP     |   ETHERNET    |  MEMORY       |
|-----------------------------------------------|                   ...
                        NODE 1                                      NODE 2
                        |                                           |
                        |___________________________________________|


(NPF) - Noise Protocol Framework
```

The life of a message through the stack starting as application data:

1. All traffic from client applications is encrypted immeditately, and only readable at its intended destination.
2. The destination address is searched for on each network in parallel. The result is temporarily cached.
3. Traffic is passed to the selected network which produces a message in its internal format and sends it where it should go.
4. The message is passed through layers which multiplex traffic from the multiple network algorithms.
This layer also ensures the MTU for one-hop traffic is a reasonable size.
5. The data is encrypted before leaving the node.
6. The data, now ciphertext, is sent out using one of the transport swarms.

All data leaving the node is encrypted until the next hop.  All data traveling through a network is encrypted until it reaches it's destination.

There is no fragmentation at the network layer, only at the one hop layer if the MTU is less than 2^16.
If a message is fragmented, it is reassembled at the next hop, before the network algorithms see it.
Application data is never fragmented, and then passed to the networks.

## API
The API presented to clients of INET256 is based on the `*Node` type in `pkg/inet256/`

Client implementations can be found in `client/`

Clients connect to the daemon, and the daemon manages setting up a virtual node, which runs all the network protocols just like any other node.  However, the virtual node's only peer is the main node.

Every process that connects to INET256 gets its own stable address, which cryptographically represents the process's identity.

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
