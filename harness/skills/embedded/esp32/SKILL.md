---
name: esp32
domain: embedded
level: technology
description: ESP32-specific constraints — dual-core scheduling, watchdogs, flash wear, and peripheral pin conflicts — before assuming general embedded advice applies.
tags: [esp32, microcontroller]
triggers: [esp32, "esp-32", esp-idf, arduino]
version: "1.0.0"
requires: []
recommends: []
capabilities: []
conflicts: []
when_to_use: Any firmware task targeting an ESP32 (Arduino framework or ESP-IDF).
when_not_to_use: A different microcontroller family — this skill's specifics (dual-core, WiFi/BT coexistence, flash partitioning) don't transfer directly.
---

# Skill: esp32

## Purpose

The constraints specific to the ESP32 that generic "embedded systems" advice glosses over.

## Constraints to check

1. **Dual-core** — WiFi/BT stack runs pinned tasks on core 0 by default; blocking user code
   on the wrong core can starve the radio stack. Know which core a task runs on before
   diagnosing a "random" disconnect.
2. **Watchdog timers** — a long blocking loop (or a busy spin with no yield) trips the task
   watchdog and reboots the device; yield (`vTaskDelay`, `yield()`) inside any loop that
   might run long.
3. **Flash wear** — frequent writes to the same NVS/SPIFFS/LittleFS region wear out flash;
   batch writes, or use wear-leveling storage, for anything written more than occasionally.
4. **Pin conflicts** — several "general purpose" pins are strapping pins or shared with
   flash/PSRAM on many ESP32 variants; a pinout that looks free on paper can still be
   unusable on the actual board revision.

## Anti-patterns

- Assuming `delay()` inside a FreeRTOS task is harmless — it blocks that task, not the CPU,
  but a task blocked too long still trips its own watchdog.
- Writing to NVS every loop iteration instead of only on a real state change.
