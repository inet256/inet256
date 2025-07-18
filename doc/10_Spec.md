# INET256 Specification

INET256 is a standardized networking API, PKI, and cryptographic address scheme.

- It *does not* prescribe protocols, wire formats, or routing algorithms.
- It *does* define an address scheme.
- It *does* define API methods.

## 1 Addresses
INET256 addresses are 256 bits dervied from serializing, then hashing a public signing key.

**TLDR:**
```
    addr = SHAKE256( Serialize( public_key ) )
```

### 1.1 Public Signing Keys
The signing algorithms supported are:
- `Ed25519`

New signing algorithms can be added or removed over time.
If two peers do not support one another's signing algorithms, they will not be able to communicate.
The tradeoffs and migration process are the same as TLS.

Importantly, the hash-of-key design allows public keys to be much larger than a hash, as will be the case with post-quantum cryptography, while retaining fixed sized addresses.

### 1.2 Serialization
Keys are serialized using a simple serialization format, inspired by SSH.
Keys are prepended with a length-prefixed string, called the type tag, which indicates which algorithm the key is for.
In the SSH protocol this is 4 bytes, which means tags can be billions of bytes in length; not a legitimate use case.
In INET256 this is 2 bytes, which means that a tag can be at most 2^16-1 bytes.

```
| Type Tag Length (16 bits)              |
| Type Tag (variable length base on tag) |
| Key Data                               |
```

### 1.3 Hashing
Keys are hashed by taking the serialized format and feeding it to `SHAKE256 XOF` then reading 256 bits of output.

SHAKE256 only provides 256 bits of collision resistance when 512 bits of output are read.
For this reason, when signing an INET256 public key for another purpose, it may be necessary to read a full 512 bits from the XOF.  The XOF vs a hash function leaves that available to the application.

For the purpose of connecting to a trusted peer, 2nd-pre-image resistance is the important quality, which is why the addresses are only 256 bits long.

## 2 API Methods
The INET256 API is an upwards facing (towards the application, not the network link layer) API specification.

It is defined in terms of methods which can be implemented by libraries, RPCs, system calls, etc.
The reference implementation uses HTTP.

## 2.1 Service API
Conceptually the service API allows the creation and desctruction of Nodes which represent entities which can send and receive messages.

### 2.1.1 `open(privateKey, nodeOptions) -> Node`
The implementation must provide a way to create new nodes in the network with a caller-provided private signing key.
The implementation is trusted to keep the key safe, and to manage running the node, and communicating with others.

Node options is where standardized per node options will be set.
No options have been standardized at the present, but they may include things like QoS, anonymity, etc.

If the implemenation cannot create a node satisfying the configuration e.g. anonymous communication, it *must* return an error.

### 2.1.2 `drop(privateKey)`
Should remove the node corresponding to private key from the network.
After disconnecting the privateKey must not be retained by the implementation.
Any Nodes from previous calls to open must return errors for subsequent operations.

## 2.2 Node API
Once a node has been created through a service, the methods below allow it to be used for communication with other nodes in the network.

### 2.2.1 `Node.send(address, message)`
The implementation must provide a way to send messages with sizes up to and including the MTU to an address.
If the message exceeds the MTU send must error.
The MTU for INET256 services is 65535.
Delivery *must* be at-most-once per message.

The implementation must provide a way to cancel or abort this operation if it takes longer than a certain amount of time.

### 2.2.2 `Node.receive() -> (address, message)`
The implementation must provide a way to recieve messages up to the MTU, and for the caller to know which address they came from.
Delivery is best effort, but must be at most once per message.

The method signature is written to show the flow of data, not to imply that the implementation must allocate memory and return it.
Messages can be delivered through shared memory, or callbacks.

### 2.2.3 `Node.findAddr(prefix) -> (address)`
The implementation must provide a way to find an address known to the network, which has the specified prefix.
The implementation must provide a way to cancel or abort this operation if it takes longer than a certain amount of time.
If an address with prefix cannot be found, an error should be returned.
If this call errors it should be assumed that `Send` will also error, and that the network does not contain any address with the prefix.

### 2.2.4 `Node.lookupPublicKey(address) -> publicKey`
The implementation must provide a way to discover the public key which corresponds to an address.
The implementation must provide a way to cancel or abort this operation if it takes longer than a certain amount of time.
If a key cannot be found an error should be returned.

### 2.2.5 `Node.publicKey() -> publicKey`
The implementation must provide a way to derive a public key from the private key used to create the node.

### 2.2.6 `Node.localAddr() -> addr`
This should return the local address of the node.
The address will be derived from the Node's public key (accessible with `Node.localAddr`) as described in section 1.

## 3 Security
Formally, implementations must guarantee [IND-CCA2](https://en.wikipedia.org/wiki/Ciphertext_indistinguishability) security from any adversary with access to the network or link layers.

All messages must be confidential.
No parties other than the sender and reciever are able to read messages sent between them.

A message sent from address A to B must only be received by a node with the private key corresponding to the address B.

A message received at address A from B must have only been sent by a node with the private key corresponding to the address B.

Messages must be delivered unaltered or not at all.

There must be no way for a given node A to prove to a third party that another node B authored a particular message.
This prohibits authenticating messages directly with long lived signing keys.

## 4 Anonymity
INET256 implementations are not required to provide anonymity, although they can.
There is no guarentee that if either the sender or reciever is anonymous, then both are.
This will be implementation dependent.
It is very likely that traffic anonymous to non-anonymous and vice versa will be the norm.

Further revisions of this spec may formalize anonymity guarantees as `NodeOptions`.

## 5 Signatures
In INET256 each node has its own private signing key, managed by the client application which created the node.
INET256 implementations use this key to represent the node on the network and for authenticating the application traffic end-to-end.
Additionally, applications may use this key for their own purposes.

In order to facillitate all this signing with the same key, all signature production and verification using the key must be done using a signature scheme which does not produce colliding signatures across purposes.

The **INET256 Signature Scheme** solves this problem.

**TLDR**
```
sigCtx = CSHAKE256("", purpose)
sig = Sign( private_key, CSHAKE256( "", sigCtx, input ) )
```

CSHAKE is customized with a purpose string.
The input to CSHAKE is the bytes of the message to be signed.
Signatures are performed on 512 bits of output from CSHAKE.
The signature algorithm used is set by the public key.
Each type of public key in INET256 has exactly one signing algorithm associated with it and that algorithm accepts no additional parameters.

INET256 implementatations must perform all signing operations with a purpose beginning with `inet256/`.
Applications are free to use any purpose, so long as it does not begin with `inet256/`.
INET256 implementations *must* refuse to perform signing operations which use a purpose reserved for implementations.
