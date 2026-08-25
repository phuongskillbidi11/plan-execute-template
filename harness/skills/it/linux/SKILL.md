---
name: linux
domain: it
level: technology
description: Linux troubleshooting fundamentals — where to look for the actual error (logs, exit codes, permissions) before assuming a tool is broken.
tags: [linux, systemd, permissions, logs]
triggers: [linux, systemd, journalctl, permission, "exit code", syslog]
version: "1.0.0"
requires: []
recommends: []
capabilities: []
conflicts: []
when_to_use: Diagnosing a failure on a Linux host — a service that won't start, a permission error, an unexplained process exit.
when_not_to_use: A pure application-logic bug with no OS/system interaction involved.
---

# Skill: linux

## Purpose

Where the actual cause of a Linux-level failure is almost always visible, before guessing.

## Method

1. **Check the exit code and `journalctl`/`syslog` before anything else.** A service that
   "just doesn't start" almost always logged why; `systemctl status <unit>` and
   `journalctl -u <unit> -n 50` answer more questions than re-reading the config file does.
2. **Permission errors name the actual problem in the error text** — "Permission denied" vs
   "No such file or directory" vs "Operation not permitted" are different failure classes
   (file ownership/mode, a wrong path, or a capability/SELinux/AppArmor restriction
   respectively); read which one it actually is before changing permissions blindly.
3. **A process that dies with no log output** may have been OOM-killed — check `dmesg` /
   `journalctl -k` for an OOM-killer entry before assuming the application crashed on its
   own.
4. **`$PATH` and shell environment differ between an interactive login shell and a
   service's environment** — a command that works when typed manually can fail
   identically-named but differently-resolved under systemd/cron with a minimal
   environment.

## Anti-patterns

- Changing file permissions to `777` to "fix" a permission error without reading which
  specific permission was actually missing.
- Assuming a silently-dead process crashed in application code before checking for an
  OOM-kill.
