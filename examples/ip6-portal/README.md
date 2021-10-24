# IPv6 Portal

The IPv6 Portal is an INET256 application like any other.
It connects to the daemon, spawns a Node for itself, using its a private key, and then sends and receives messages.
It also creates a TUN device and routes traffic from a given IPv6 address to the corresponding INET256 address.

First create a private key.
```shell
$ inet256 keygen > ip6-example.pem
```

Print the IPv6 address for that private key.
This IPv6 address is what you would share with others so they can connect with you.
```shell
$ inet256 ip6-addr --private-key ip6-example.pem
207:dbc6:4305:e5d2:b1ed:e3ac:32c2:4875
```

Run the portal.
This will create a TUN device with routes setup for the subnet.
```shell
$ inet256 ip6-portal --private-key ip6-example.pem 
INFO[0000] Created TUN utun4
INFO[0000] Local INET256: Ht4yGC8ulY9vHWGWEkOvAm2Wky-uXHLWGnFGL3AIZtE
INFO[0000] Local IPv6: 207:dbc6:4305:e5d2:b1ed:e3ac:32c2:4875
INFO[0000] ifconfig output:
INFO[0000] device up
INFO[0000] mtu update
```
