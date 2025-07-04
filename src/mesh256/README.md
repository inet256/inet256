# Mesh256
Mesh256 is the reference implementation of an `inet256.Service`.
Mesh256 is built as a mesh network using a distributed routing algorithm.

Any suitable routing algorithm can be used to route packets and drive the network.
The interface to implement is `mesh256.Network`, defined in `network.go`.
There is a package including tests for `mesh256.Network` implementations in `mesh256test`.

Mesh256 is built around `Swarms` from the `go-p2p` library.

Networks (routing algorithms) are given a private key, a `Swarm` and a `PeerSet`.

The `PeerSet` is just a list of peers that are accessible through the swarm.
These are the one-hop peers.
It is the job the Network to figure out how to route messages to more peers than just those in this set.

## Stack
An mesh256 Node's stack looks like this, implemented as layered `Swarms`

```
                Application Data :-)
|-----------------------------------------------|
|             Encryption Layer (P2PKE)          |
|-----------------------------------------------|
|           Network Routing Algorithm           |
|-----------------------------------------------|
|        Packet Fragmentation & Assembly        |
|-----------------------------------------------|
|                 Multihoming                   |
|-----------------------------------------------|
| Transports:                                   |
|     (QUIC)   |    (QUIC)      |               |
|      UDP     |    ETHERNET    |  MEMORY       |
|-----------------------------------------------|                   ...
                        NODE 1                                      NODE 2
                        |                                           |
                        |___________________________________________|


```

The life of a message through the stack starting as application data:

1. All traffic from client applications is encrypted immeditately, and only readable at its intended destination.
2. The applicaiton ciphertext is passed to the network routing algorithm, which creates a network message.
3. The network message may need to be broken up and reassembled if it is larger than the transport MTU.
4. The best transport for a peer is selected.
5. The network message is encrypted if it has to leave the process.
6. The network message ciphertext, is sent out using one of the transport swarms.

All data leaving the node is encrypted until the next hop.  All data traveling through a network is encrypted until it reaches it's destination.

There is no fragmentation at the network layer, only at the one hop layer if the MTU is less than 2^16-1.
If a message is fragmented, it is reassembled at the next hop, before the network algorithms see it.
Application data is never fragmented, and then passed to the networks.
