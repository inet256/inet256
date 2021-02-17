# LAN Discovery

The lan discovery protocol works by multicasting beacons which contain information about the transport addresses that a node is listening on.

In order to prevent tracking (real life location tracking), messages can only be read by someone who knows the PeerID of the sender.
This is accomplished by encrypting the message with the PeerID of the sender, and attaching the Hash of the PeerID concatenated with a nonce.

When the amount of nodes on the network is small, it's not that difficult to try the ID of every peer on the network, but when it is large it's more difficult.
Decryption is an O(n) operation where n is the number of peers you are looking for.
Anyone can author a message as any peer, but authentication is handled by INET256, so they couldn't trick anyone into connecting to them.

Better solutions are welcome.  You can always turn it off.
