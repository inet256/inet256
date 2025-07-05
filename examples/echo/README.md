# Echo Example

A config has already been provided with `inet256 create-config --api_endpoint=tcp://127.0.0.1:2560 > config.yml`.
Generate your own key with `inet256 keygen > private_key.inet256`

Now run the daemon, let that sit in one terminal.
```
inet256 daemon --config=config.yml
```

In another terminal run the echo server.
This will connect to the daemon through the API and the daemon will manage the node.
The echo command does not run it's own node.
```shell
$ INET256_API=tcp://127.0.0.1:2560 inet256 echo
```

The first log line is the address of the echo server. We will need that in a minute

In another terminal (3rd one, last one) run the nc command.  Use the echo servers address as the first argument.
This will also connect to the daemon.
This process does not run it's own node.

```shell
$ INET256_API=tcp://127.0.0.1:2560 inet256 nc <echo server address>
```

Now try typing something, everything should be echoed back to you.
e.g.
```
INFO[0000] uMd_IBiw5_XD2hcE4r3u4MDrB-SNB2VgrejfkffoL-c
hello
hello
hello world
hello world
hello world my name is foo
hello world my name is foo
```

And you should see some log lines in the echo server's output

```
INFO[0000] X2jlb0m9oFiWmgGBZZsBwkzWGewMvCQswlFQgIu1IPY
INFO[0044] echoed 5 bytes from uMd_IBiw5_XD2hcE4r3u4MDrB-SNB2VgrejfkffoL-c
INFO[0050] echoed 11 bytes from uMd_IBiw5_XD2hcE4r3u4MDrB-SNB2VgrejfkffoL-c
INFO[0062] echoed 26 bytes from uMd_IBiw5_XD2hcE4r3u4MDrB-SNB2VgrejfkffoL-c
```
