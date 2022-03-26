# Daemon Config
Documentation for the config file used by the reference implementation.

The daemon takes the path of the config file as a required flag
```
inet256 daemon --config=path/to/config/file.yaml
```

The configuration is provided in YAML.  It is deserialized into a structured defined in [pkg/inet256d/config.go](../pkg/inet256d/config.go).

# Fields

## `private_key_path`
The path to a file containing a private key.

e.g.
```yaml
private_key_path: path/to/private/key.pem
```

Paths by default are interpretted normally: relative to the current working directory.

As a special case paths beginning with a `./` are relative to the location of the config file.
So if the config is at `/home/user/inet256.yaml` then `./` would resolve to `/home/user`.

```yaml
private_key_path: ./path/relative/to/config.pem
```

## `api_endpoint`
The endpoint that the daemon will listen at, to serve the INET256 API.

e.g.
```yaml
api_endpoint: "127.0.0.1:2560"
```

## `transports`
This is a list of transports used to talk to one-hop peers.
There is no concept of connections or listening vs dialing.
If you want to communicate over a transport you need to have it enabled.

The example below will listen bind on all IPv4 addresses on a randomly chosen high number port.
The port will change every time the daemon is started, so this might be fine if you are using a discovery service, but if you are giving out a static address to your peers, this wouldn't work.
```yaml
transports:
- udp: "0.0.0.0:0"
```

This example would listen on port 9000.
If the node had a static IP, then you could give out the address with this port to your peers.
```yaml
transports:
- udp: "0.0.0.0:9000"
```

## `peers`
This is a list of peers or one-hop nodes to connect to.
They are presented as a set to the network algorithms.

Peers have an ID, which is their INET256 Address, and a list of transport addresses, which are used to contact them.

e.g.
```yaml
peers:
- id: Zxk5ZDOqk7fc74d5dzYQQl4zTEjfe6zvQ6ffLFXpHEU 
  addrs:
  - "quic+udp://Zxk5ZDOqk7fc74d5dzYQQl4zTEjfe6zvQ6ffLFXpHEU@12.34.56.78:9000"
- id: fK_KKhsijufvMAEoni0pkGddacrZAL5DCxhAukOc9jg 
  addrs:
  - "quic+udp://fK_KKhsijufvMAEoni0pkGddacrZAL5DCxhAukOc9jg@99.98.97.96:4500"
  - "quic+udp://fK_KKhsijufvMAEoni0pkGddacrZAL5DCxhAukOc9jg@100.99.40.30:4050"
```

The transport addresses also includes the protocol used to encrypt traffic over the transport.
This is setup automatically, at a layer above the transport configuration.
In order to see your node's transport addresses you can run `inet256 status` command.

## `network`
This is a specification for a network routing algorithm.

The example below will run the `beaconnet` network on all Nodes created by the service.
```yaml
network:
  beaconnet: {}
```
`beaconnet` exposes no additional parameters so it's specification is just the empty object.

Here is an example of 2 networks, using the same algorithm with different parameters.
`multi` is a network which nests other network specs, and multiplexes them onto the transport layer.
```yaml
network:
  multi:
    "n1":
        algo1:
            param1: 1
    "n2":
        algo1:
            param1: 2
```
- `n1` and `n2` are the network `codes` (used for multiplexing).
- `algo1` is the name of the network algorithm.
- `param1` is a parameter specific to `algo1`. The parameter `param1` is set to different values in each instantiation.

Network algorithms have to be compiled into the daemon to be used.
The `ls-networks` command in the CLI can be used to list the networks the daemon implements.
Different forks of the project may ship with different networks.

## `discovery`
A list of discovery service specs.
Discovery services allow the daemon to find transport addresses for a given INET256 peer.  This makes peering over the internet easier.

In the example below a central discovery server is configured to announce and find peers.
```yaml
discovery:
- central:
    endpoint: "123.234.132.231:8000"
```

Discovery services *do not* create connections to unknown peers.
A discovery service continuously attempts to find transport addresses for all of the peers specified in the configuration.
