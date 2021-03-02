# Kademlia + Source Routing
This network uses a routing algorithm similar to CJDNS

## Architecture
There are several messages types which are mostly paired into requests/responses for a few services.
- RequestPeerInfo/PeerInfo
- Ping/Pong
- QueryRoutes/RouteList
- Data

The request/responses are handled by services, which manage access to underlying stores.

Driving the whole operation is the crawler, which is constantly looking for new routes to peers close in key space.

Messages are defined and serialized using protocol buffers.
It's not a great choice for a production network protocol, but it makes iteration very fast.
If our biggest problem was serialization performance and we had to switch, that would be awesome.

### Message Types

#### Ping/Pong
Ping is for checking if a path works.
Pings should never have their path overridden, that would defeat their purpose.
They intentionally contain very little information, and do not contain any public keys.
A pong is a response to a ping. Unsolicited pongs should be ignored.
When a route can no longer be pinged, it should be removed from the table.

#### PeerInfoRequest PeerInfo
This is basically an exchange of keys.
Requests should be responded to with a PeerInfo.
These messages contain enough information to confirm the address is real and validate the message signature.
These messages should add the public key to a cache so that subsequent messages can be validated.

#### QueryRoutes/RouteList
This is how nodes grow their route table.
They ask other nodes for paths to nodes they haven't seen.

#### Data
Data messages are data from the layer above that must be routed.
They are subject to path improvement, and do not have to contain a complete path to the destination.
