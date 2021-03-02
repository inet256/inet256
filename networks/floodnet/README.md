# Flood Network
This is a naive routing algorithm which floods the network with all messages.
Messages have a max hop count to prevent infinite cycles.
Maybe a uuid, for dropping already seen messages is a better way.

This really only exists to test that the Node and Server in the `inet256` package work with a trivially correct (albeit inefficient) routing algorithm.
