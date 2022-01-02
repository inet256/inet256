# Utilities

This project ships an `inet256` binary which includes a reference implementation of an INET256 service, implementing the [INET256 Spec](./10_Spec.md).
It also includes several utilities which are INET256 applications, and consume the INET256 service as clients.
These utilities are not coupled to the reference implementation, and could work with any implementation of INET256.

## `inet256 echo`
Generate a random key and begin listening for messages, and echoing them back to the sender.

## `inet256 nc <dst>`
Reads messages from stdin line buffered and sends them to an address

## `ip6-portal --private-key <path>`
Runs the [IP6 Portal](./31_IP6_Portal.md)

## `ip6-addr [--private-key <path>] [addr]`
Prints the IPv6 address that would be assigned in the portal to a given INET256 address, or private key.
