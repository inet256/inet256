# Mesh256
Mesh256 is the reference implementation of an `inet256.Service`.
Mesh256 is built as a mesh network using a distributed routing algorithm.

Any suitable routing algorithm can be used to route packets and drive the network.
The interface to implement is `mesh256.Network`, defined in `network.go`.
There is a package including tests for `mesh256.Network` implementations in `mesh256test`.
