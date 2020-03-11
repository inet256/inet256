# INET256

A proposed standard for network identity and address allocation.

## Introduction
Surveying peer-to-peer applications written in recent years, a certain trend has become obvious.
Applications are built on top of networks whose address space is the hash of some signing key.
The libp2p stack does this.
CJDNS does this. Yggdrasil does this. Tor does this.
Just to name a few popular peer-to-peer projects.

The pattern is actually very obvious once you sit down to write an application in this domain.

Projects are converging on the same design, without any coordination, or central planning.
When this happens it seems to indicate something deeply true about the solution space.

## What’s in an Address?
Address allocation has to be about identities.
The path through the network between identities is secondary, and may not even be unique.
There may be many paths, each with different properties: latency, financial cost, and bandwidth.
This is a new idea to many people, especially if your only experience with networks is IPv4/IPv6, both of which conflate identity and location.

All of the applications mentioned earlier use slightly different methods to derive an address.
We may benefit from some sort of standardization.

## A Standard
Hopefully by exploring the design space, and tradeoffs between parameters herein, developers of new projects can be guided to the same point in the space, rather than all ending up a stone’s throw from one another.

Most standards are doomed to fail, but this is usually due to there not actually being one simplest way to do something.
One group wants to include functionality for their own use case and exclude other functionality for “simplicity”.
And so there are competing standards with a lot of overlap, but with no simple mapping between them.
Neither standard is a superset of the other.
A standard with a superset of the functionality of both would be more complicated than both, and therefore less likely to be adopted.

*But this time it’s different*

## Design
In this case there are really only 2 parameters in the design space.
```
The hash function. f(bytes) → hash_sized_output
The key serializing function. f(key) → bytes
```

And there are already widely adopted standards for both of these things. Let’s go through them.

### Hash Function
SHA - Secure Hash Algorithm.
Every once in a while the NIST in the U.S. holds a contest to certify one algorithm as SHA*X*.  These algorithms are the most widely used cryptographic hash functions.
This is the most widely used standard, so we should use it.
Sorry BLAKE.

> Aside: If you are unfamiliar with the BLAKE functions you should take a look at BLAKE3.  It's very fast.

The latest SHA*X* is SHA3, so let’s use that.
Although the SHA3 functions are slower than the SHA2 family of functions, the SHA2 functions are vulnerable to length extension attacks.
This isn’t really an issue, many secure systems work around it.
But it means that a developer knowing nothing about how hash functions work internally, and just assuming they approximate a random oracle reasonably well, could design a broken system if they reached for SHA2, but would be fine reaching for SHA3.
From a software engineering perspective, SHA3 is a categorically better primitive for this reason.

Next is the length.
A 256 bit hash function can provide 256 bit security against second preimage attacks, and 128 bit security against collisions.
A 512 bit hash function can provide double for both.
There is a tradeoff between security and address size here.
Smaller addresses are better, more security is better.
Many people consider 256 bit hashes to be sufficient and 512 to be overkill.
At the time of writing, that seems reasonable.
The rationale at least is clear: if we need the extra bits then we have to pick 512, if we don’t then we don’t.
Smaller is better, and 256 is enough, so 256 bits it is.

### Key Serialization
Now let’s look at the key serializing function.

The key marshaling function needs to support multiple key types, since public key cryptography is changing rapidly, and we want to enable a switch to other asymmetric algorithms, especially post-quantum algorithms, down the road.
Many of these networks just use one key algorithm, we want to make it easy for them to support more.

It turns out that this is done all the time without too much fuss in TLS.  The sheer number of TLS connections active at any point in time make x509 the most popular standard for serializing keys.

I know many people cringe at TLS because it is so complicated. It’s particularly looked down on among peer-to-peer application developers because it’s so complicated to just secure a connection between two peers who already know each others identity. The IPFS project even made their own secure transport secio out of frustration.

But let me explain, we only need the public key serialization part which doesn’t include any of the certificate stuff that scares people away from p2p development with TLS.

The serialization function is actually just composed of already standardized components.
ASN.1 is just a way of representing structured data.
DER is just a deterministic way of serializing ASN.1 structures.
PKIX composes these along with a number from somewhere in the sky to identify the public key algorithm.
X509 PKIX seems like the best candidate for our key serializing function.

If you follow and agree with this reasoning then we have chosen `SHA3-256` as the hash function.  And `X509 PKIX` as the key marshaling function.

This particular method of allocating addresses, and determining what key is at an address is the **INET256 Address Scheme**

## What does this get us?

### Network Bridges
If an address means the same thing to everyone, someone could bridge a network like CJDNS with Tor.
Right now Tor uses the SHA3 of ed25519 public keys, and CJDNS uses the SHA512 of RSA public keys.
If they both used INET256 addresses then nodes in one could contact nodes in another.

### Single Target For Applications
Single target for application developers.
People can start hosting services and connecting to one another based on stable identifiers, while projects like Yggdrasil and CJDNS innovate beneath them.
No NAT traversal, no VPNs, no certificates.
Just a list of addresses will be the norm for configuration.

### Applications As Hosts
When addresses are plentiful, and allocated cryptographically, there is nothing stopping each application from having its own address.
A user might have a node running on their system that manages peering with others, and then several application nodes peering with the system node.
This allows each application node to use the network in a way optimized for the application.
If the underlying network protocol supports QoS for example, then the node could leverage certain QoS settings to benefit the application.

### Mapping to IPv6
The Yggdrasil project has a great way of mapping IPv6 addresses to their internal network addresses.
They use a compression scheme to remove similar leading bits, and the internal networking space is mapped into a deprecated IPv6 subnet.
Every network should be able to leverage this scheme.
This mapping enables IPv6 applications to communicate with the network.

### Next Steps
The way to gain adoption is not to go around trying to change existing networks.
Developers are not interested in changing their software just to adhere to a standard without proven value.
The way towards adoption is to develop applications that expect INET256 beneath them, or that create INET256 networks internally.
Still, network developers want users, and if INET256 helps them get users they may be interested.

The goal is to create a growing ecosystem of networked applications sharing the same address space.
Eventually the functionality in that ecosystem will become hard to ignore; tooling, routing algorithms, and other investment will manifest.
Right now it’s bring your own network, and run it in-process, but eventually INET256 could become something you expect to have as a primitive.

Take a look at [blobcache](https://github.com/brendoncarroll/blobcache).  It creates an overlay network similar to CJDNS as part of running the application.
It uses INET256 addresses in its overlay network.
It also uses a different hash function (BLAKE3) for most of the application, that’s fine, and proof that this standard is minimally invasive.

The library [go-p2p](https://github.com/brendoncarroll/go-p2p) enables the creation of inet256 networks.
The `PeerID` type is equivalent to an INET256 address.
Functions are included for marshaling and parsing public keys, and constructing PeerIDs.  Consider using these types in a p2p application, and you will have joined the INET256 ecosystem.
