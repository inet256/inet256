# Forrest Network
This routing protocol is inspired by Yggdrasil, but instead of one spanning tree, it uses a forrest of spanning trees.

## Wire Format
The message is divided into 3 sections:
- Type
- Metadata Length
- Body

```
| 1 byte type | 3 byte metadata length | message body |
```

The metadata length is the amount of the body which is metadata.
For example data messages will have the application data in the data section, and src, dst and other routing information in the metadata section.
