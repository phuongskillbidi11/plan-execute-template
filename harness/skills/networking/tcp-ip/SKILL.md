---
name: tcp-ip
domain: networking
level: technology
description: TCP/IP fundamentals that explain most "intermittent" network bugs — connection state, timeouts, and the difference between a closed port and an unreachable host.
tags: [tcp, ip, networking, socket, timeout]
triggers: [tcp, "tcp/ip", socket, timeout, "connection refused", "connection reset"]
version: "1.0.0"
requires: []
recommends: []
capabilities: []
conflicts: []
when_to_use: Debugging or designing anything that opens a TCP connection — a service, a device integration, an API client.
when_not_to_use: A purely UDP-based or link-layer-only problem — TCP's specific guarantees (ordering, retransmission, connection state) don't apply.
---

# Skill: tcp-ip

## Purpose

The handful of TCP/IP facts that explain most bugs reported as "sometimes it just doesn't
connect."

## Facts worth checking

1. **"Connection refused" vs "timeout" mean different things.** Refused means a host
   responded and nothing is listening on that port (or a firewall actively rejected it,
   depending on config) — the network path works. A timeout with no response at all usually
   means the host is unreachable, or a firewall is silently dropping packets, not that the
   remote service is merely slow.
2. **TCP is a stream, not a message protocol.** One `write()` is not guaranteed to arrive as
   one `read()` — application-level framing (a length prefix, a delimiter, a fixed record
   size) is the caller's responsibility, not TCP's.
3. **Keep-alive is not the same as an application-level heartbeat.** OS-level TCP keep-alive
   can take minutes to detect a dead peer by default; if a protocol needs to detect a stale
   connection quickly, it needs its own heartbeat/timeout at the application layer.
4. **A half-open connection can look alive** — if one side crashes without sending a FIN
   (e.g. power loss on an embedded device), the other side may not notice until it tries to
   write and gets a reset, or until a read timeout fires.

## Anti-patterns

- Retrying a "connection refused" in a tight loop assuming it's transient, without checking
  whether the target service is actually running.
- Assuming a single `write()` call corresponds to a single `read()` on the other end.
