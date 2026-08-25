---
name: modbus
domain: automation
level: technology
description: Modbus addressing and framing pitfalls — PDU addresses are not the same as the conventional 4xxxx/3xxxx register numbers, and RTU vs TCP framing differs.
tags: [modbus, "modbus tcp", "modbus rtu", register]
triggers: [modbus, "modbus tcp", "modbus rtu", "holding register", "40001"]
version: "1.0.0"
requires: []
recommends: [networking/tcp-ip]
capabilities: []
conflicts: []
when_to_use: Reading or writing Modbus registers/coils over either TCP or serial (RTU/ASCII).
when_not_to_use: A protocol that merely resembles Modbus in concept (e.g. a proprietary polling protocol) — don't assume Modbus-specific addressing applies.
---

# Skill: modbus

## Purpose

The addressing and framing details that cause almost every real Modbus bug.

## Method

1. **PDU address ≠ conventional register number.** "Read holding register 40001" in vendor
   documentation almost always means PDU address `0x0000` (the conventional numbering is
   1-indexed and offset by a function-code range: 40001 → PDU 0, 40002 → PDU 1, ...). Do not
   assume the literal number in a spec sheet is the value to send on the wire — check which
   convention the specific device/library uses.
2. **Function code determines the address space.** Coils, discrete inputs, input registers,
   and holding registers are four separate address spaces that can each start at 0 —
   "register 5" is ambiguous without knowing which function code (space) it's in.
3. **RTU vs TCP framing differs.** RTU adds a slave/unit ID and CRC over serial; TCP wraps
   the same PDU in an MBAP header with a transaction ID instead of a CRC (TCP already
   guarantees byte integrity). A library written for one doesn't automatically speak the
   other.
4. **Byte/word order (endianness) for multi-register values is not standardized** — a 32-bit
   value spanning two registers can be big-endian, little-endian, or word-swapped depending
   on the vendor; verify against the specific device's documentation rather than assuming.

## Anti-patterns

- Blindly assuming a spec's "40001" is the literal address to request.
- Assuming a Modbus TCP library will work unmodified against RTU hardware (or vice versa).
