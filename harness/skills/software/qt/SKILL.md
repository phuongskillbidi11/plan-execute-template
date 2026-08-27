---
name: qt
domain: software
level: technology
description: QObject ownership, signals/slots, the event loop, and thread affinity — the Qt-specific facts layered on top of general C++ methodology.
tags: [qt, qobject, qwidget, qthread, qml]
triggers: [qt, qobject, qwidget, "signal/slot", "signals and slots", qthread, qml, automoc, autouic, autorcc]
version: "1.0.0"
requires: [software/cpp]
recommends: []
capabilities: []
conflicts: []
when_to_use: Reading, writing, or debugging Qt (Widgets or QML) application code.
when_not_to_use: Plain C++ with no Qt involved — use software/cpp alone.
---

# Skill: qt

## Purpose

What's specific to Qt, on top of the general C++ ownership/lifetime pitfalls in `software/cpp`
(a hard dependency of this skill — always loaded alongside it).

## Method

1. **Parent-child ownership is Qt's real memory-management model for `QObject`s** — a `QObject`
   constructed with a parent is deleted automatically when that parent is deleted; giving a
   heap-allocated `QObject` a parent and then also wrapping it in a `unique_ptr`/`delete`-ing it
   manually causes a double-free. Widgets in particular are almost always parented (to their
   containing layout/window) — treat an un-parented, manually-managed widget as the exception
   that needs a documented reason, not the default.
2. **Signals/slots decouple the sender from needing to know who's listening**, but a connection
   made with `Qt::AutoConnection` (the default) resolves to either a direct call or a queued
   event depending on whether sender and receiver live on the same thread *at connection time*
   — this can change behavior silently if an object is moved to a different thread later. Use an
   explicit `Qt::ConnectionType` when the cross-thread behavior actually matters.
3. **The event loop (`QCoreApplication::exec()`) is what makes signals, timers, and queued
   connections actually run** — code that blocks the GUI thread (a long computation, a
   synchronous network call, a tight loop) freezes the whole UI, since no queued event can be
   processed until control returns to the event loop. Move genuinely long work to a `QThread` or
   `QtConcurrent`, don't just hope it's fast enough.
4. **`QThread` "thread affinity" is about which thread a `QObject` *lives on* for
   event/slot delivery, not which thread executes a given function call.** Moving a worker
   `QObject` to a `QThread` via `moveToThread()` makes its slots run on that thread when
   invoked via a queued connection — calling a method on it directly (not through a signal) still
   runs on the caller's thread. Don't create a `QObject` inside a `QThread::run()` override and
   expect the usual moveToThread-based affinity rules to apply the same way.
5. **Qt Widgets' model/view separation (`QAbstractItemModel` + a view) exists so large
   datasets don't require materializing every row as a widget** — reaching for a widget-per-row
   approach for anything beyond a small, fixed list is usually the wrong default; implement a
   model instead once the data is dynamic or large.
6. **Qt Network's classes are asynchronous by default** (`QNetworkAccessManager::get()` returns
   immediately; the response arrives via the `finished` signal) — code written as if a network
   call blocks and returns a value directly is a common source of "why is this always empty"
   bugs.
7. **AUTOMOC/AUTOUIC/AUTORCC (CMake) generate code from `Q_OBJECT` macros, `.ui` files, and
   `.qrc` files respectively at build time** — a "undefined reference to vtable" or "unknown
   type" build error after adding a new `Q_OBJECT` class or `.ui` file is very often a missing
   re-run of these generators (usually just needs a clean re-configure), not an actual code bug.

## Anti-patterns

- Manually deleting a parented `QObject` (or wrapping it in a smart pointer) in addition to
  its parent-based ownership, causing a double-free.
- Running long/blocking work directly on the GUI thread and wondering why the UI is frozen.
- Assuming `moveToThread()` changes which thread a direct (non-signal) method call executes on.
