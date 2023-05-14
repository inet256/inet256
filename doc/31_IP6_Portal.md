# IP6 Portal

The IP6 Portal is a compatibility bridge to IPv6.
It provides access to an IPv6 subnet, where IPv6 addresses are assigned deterministically based on INET256 addresses.
This is the primary interface exposed by networks like Yggdrasil and CJDNS.
INET256 doesn't play favorites with network traffic.  All traffic is encrypted before it even makes it to the routing layer.
IPv6 traffic is *just* application data, and the IPv6 Portal is *just* another application.

The portal works by creating a TUN device and tells the OS to route all traffic with IP6 prefix `0200::/7` to the portal's device.

The portal process reads and writes from the TUN device in a loop.
The [code](../pkg/inet256ipv6/portal.go) is very simple if you are curious.
When a packet comes in from the TUN device, the portal looks up an INET256 address which could match the IPv6 address and sends the packet to that address over INET256.
The whole IPv6 packet is sent over INET256; the headers are not unwrapped, and rewrapped.
When a message comes in from INET256, its source INET256 address will correspond to exactly one IPv6 address.
If the IPv6 source address in the packet's header matches what it is supposed to be, the whole message is copied to the TUN device.  If it doesn't match, it is dropped.
The operating system receives the packet as regular IPv6 traffic, so any application works on the internet will also work using the IP6 Portal.

> NOTE to application developers:
>
> If you are developing a new application which depends on INET256, it is highly recommended that you consume the INET256 API directly.
It is generally not safe to assume that IPv6 traffic is confidential, although it would be when using the portal.
If you mess up an address, or start serving traffic on a different IPv6 address, then you are at risk of communicating insecurely.
