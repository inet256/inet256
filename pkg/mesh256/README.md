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
|             Encryption Layer (QUIC)           |
|-----------------------------------------------|
|              Routing Algorithms               |
|-----------------------------------------------|
|      Multiplexing & Packet Fragmentation      |
|-----------------------------------------------|
|             Encryption Layer (QUIC)           |
|-----------------------------------------------|
| Transports:                                   |
|       UDP     |   ETHERNET    |  MEMORY       |
|-----------------------------------------------|                   ...
                        NODE 1                                      NODE 2
                        |                                           |
                        |___________________________________________|


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

There is no fragmentation at the network layer, only at the one hop layer if the MTU is less than 2^16-1.
If a message is fragmented, it is reassembled at the next hop, before the network algorithms see it.
Application data is never fragmented, and then passed to the networks.
