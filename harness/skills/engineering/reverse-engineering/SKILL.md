---
name: reverse-engineering
domain: engineering
level: engineering
description: Method for working with decompiled or reverse-engineered code and protocols — preserve verified behavior, form evidence-based hypotheses, and make small reversible changes. For legitimate interoperability, protocol analysis, and maintenance — not a malware-analysis framework.
tags: [decompiled, "reverse engineering", disassembly]
triggers: [decompiled, "reverse engineering", ghidra, disassembly, "reversed protocol"]
version: "1.0.0"
requires: []
recommends: []
capabilities: []
conflicts: []
when_to_use: Working with decompiled source, a disassembled binary, or a protocol whose only
  documentation is prior reverse-engineering work — maintaining, extending, or interoperating
  with something whose original source or spec isn't available.
when_not_to_use: Malware analysis, exploit development, or any use aimed at bypassing licensing/
  DRM rather than legitimate interoperability, maintenance, or protocol analysis of something
  you have a right to interoperate with.
---

# Skill: reverse-engineering

## Purpose

How to work safely and effectively with decompiled code or a reverse-engineered protocol —
the goal is legitimate interoperability, protocol analysis, debugging, and maintenance. This is
not a malware-analysis or exploitation framework.

## Method

1. **Known-good behavior is the source of truth, not the decompiled/disassembled code's
   apparent structure.** Decompiler output is a *reconstruction*, not the original source —
   variable names, control-flow shape, and even apparent logic can be decompiler artifacts
   rather than the original author's intent. When decompiled code and observed real-world
   behavior disagree, trust the observed behavior and treat the decompiled reading as the thing
   that needs re-checking.
2. **Do not "clean up" decompiled output as a first step.** Renaming variables, restructuring
   control flow, or "simplifying" logic you don't yet fully understand risks silently changing
   behavior that was subtly load-bearing (an odd-looking check that's actually a real edge-case
   handler, a magic constant that's actually a protocol requirement). Understand *why* something
   looks odd before changing how it's expressed — a change with zero behavioral difference is
   the only safe kind at this stage.
3. **Every claim about what reversed code does is a hypothesis until verified against real
   behavior.** Prefer testing a specific, falsifiable prediction (this input produces this
   output, this byte at this offset means X) against the real system over reasoning from the
   disassembly alone — a plausible-looking read of assembly/decompiled code is often wrong in
   a way only observed behavior reveals.
4. **Be explicit about the binary/source boundary.** State clearly which parts of your
   understanding come from the actual binary/protocol capture (verified) versus inference,
   convention, or a similar-looking known protocol (assumed) — conflating the two is how a
   plausible-but-wrong assumption quietly becomes treated as fact.
5. **Document uncertainty explicitly, not just conclusions.** A note like "bytes 4-7 appear to
   be a length field, unconfirmed against edge cases" is more valuable to whoever reads it next
   (including your own future self) than a confident-sounding but unverified claim stated as
   fact.
6. **Make small, reversible changes and re-verify against known-good behavior after each one.**
   When modifying or reimplementing reverse-engineered logic, change one specific, testable
   thing at a time and confirm the observable behavior still matches the known-good reference —
   a large rewrite based on your own understanding of decompiled code risks losing correctness
   you can't easily attribute to any one change.
7. **Capture and replay real traffic/behavior as your regression baseline**, the same principle
   `engineering/protocol-engineering` recommends for protocol work generally — a real captured
   sequence from known-good behavior is more trustworthy than a from-scratch reimplementation
   based purely on reading the reversed logic.

## Anti-patterns

- Restructuring or "simplifying" decompiled code before understanding why it looks the way it
  does, and silently changing behavior in the process.
- Treating a plausible reading of disassembly as confirmed fact without checking it against
  real observed behavior.
- Making a large rewrite of reverse-engineered logic in one step instead of small, individually
  re-verified changes.
