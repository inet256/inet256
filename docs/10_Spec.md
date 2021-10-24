# INET256 Specification

INET256 is a standardized networking API, and cryptographic address scheme.
It does *not* prescribe protocols, wire formats, or routing algorithms.
It does define an address scheme.
It does define an upward facing API.

## Addresses
INET256 addresses are 256 bits dervied from serializing, then hashing a public signing key.
The signing algorithms supported are RSA and Ed25519.
Keys are serialized using the PKIX serialization format.
Keys are hashed by taking the serialized format and feeding it to SHAKE256 XOF then reading 256 bits of output.

New signing algorithms can be added or removed over time.

## API Methods
The INET256 API is defined in terms of methods which can be implemented by libraries, RPCs, system calls, etc.
The reference implementation uses gRPC.

## `connect(privateKey) -> Node`
The implementation must provide a way to create new nodes in the network with a caller-provided private signing key.
The implementation is trusted to keep the key safe, and to manage running the node, and communicating with others.

## `disconnect(privateKey)`
Should remove the node corresponding to private key from the network.
After disconnecting the privateKey must not be retained by the implementation.

## `Node.mtu(address) -> mtu`
The implementation must provide a way to determine the maximum message size.
The implementation must provide a way to cancel or abort this operation if it takes longer than a certain amount of time.
This call may not error, it should instead default to some minimum MTU if there is a timeout.

## `Node.send(address, message)`
The implementation must provide a way to send messages with sizes up to and including the MTU to an address.
If the message exceeds the MTU send must error.
Delivery is best effort, but must be at most once per message.

The implementation must provide a way to cancel or abort this operation if it takes longer than a certain amount of time.

## `Node.receive() -> (address, message)`
The implementation must provide a way to recieve messages up to the MTU, and for the caller to know which address they came from.
Delivery is best effort, but must be at most once per message.

The method signature is written to show the flow of data, not to imply that the implementation must allocate memory and return it.
Messages can be delivered through shared memory, or callbacks.

## `Node.findAddr(prefix) -> (address)`
The implementation must provide a way to find an address known to the network, which has the specified prefix.
The implementation must provide a way to cancel or abort this operation if it takes longer than a certain amount of time.
If an address with prefix cannot be found, an error should be returned.
If this call errors it should be assumed that `Send` will also error, and that the network does not contain any address with the prefix.

## `Node.lookupPublicKey(address) -> publicKey`
The implementation must provide a way to discover the public key which corresponds to an address.
The implementation must provide a way to cancel or abort this operation if it takes longer than a certain amount of time.
If a key cannot be found an error should be returned.

## `Node.publicKey() -> publicKey`
The implementation must provide a way to derive a public key from the private key used to create the node.

## `Node.localAddr() -> addr`
This should return the local address of the node.

## Security
A message sent from address A to B must only be recieved by a node with the private key corresponding to the address B.
A message recieved at address A from B must have only been sent by a node with the private key corresponding to the address B.

Messages must be delivered unaltered or not at all.
Messages must be delivered at most once.

There must be no way for a given node A to prove to a third party that another node B authored a particular message.

All messages must be confidential.
No parties other than the sender and reciever are able to read messages sent between them.

## Anonymity
INET256 implementations are not required to provide anonymity, although they can.
There is no guarentee that if either the sender or reciever is anonymous, then both are.
This will be implementation dependent.
It is very likely that traffic anonymous to non-anonymous and vice versa will be the norm.
