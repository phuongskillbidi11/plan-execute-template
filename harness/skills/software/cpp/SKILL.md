---
name: cpp
domain: software
level: technology
description: C++ pitfalls worth checking before trusting a change — ownership, lifetime, and undefined behavior over syntax preference.
tags: [cpp, "c++", memory, raii]
triggers: [cpp, "c++", raii, "undefined behavior", segfault, "memory leak"]
version: "1.0.0"
requires: []
recommends: []
capabilities: []
conflicts: []
when_to_use: Reviewing or writing C++ code, especially anything touching raw pointers, manual memory management, or object lifetime.
when_not_to_use: Pure build-system or packaging questions with no C++ source involved.
---

# Skill: cpp

## Purpose

The C++-specific checks that matter more than style: who owns this memory, how long does it
live, and does anything here invoke undefined behavior.

## Checklist

1. **Ownership** — for every `new`/raw pointer, who deletes it, and can that path be skipped
   by an early return or an exception? Prefer RAII (`unique_ptr`, `shared_ptr`, containers)
   over manual `delete`.
2. **Lifetime** — does a reference or pointer outlive the object it points to (a common bug
   with lambda captures and container reallocation)?
3. **Undefined behavior** — signed overflow, use-after-move, uninitialized reads, and
   out-of-bounds access don't reliably crash; they corrupt silently. Treat compiler
   warnings (`-Wall -Wextra`) about these as blocking, not cosmetic.
4. **Const-correctness** communicates intent — a non-const reference parameter that's never
   mutated is a readability bug waiting to become a real one.

## Anti-patterns

- Catching `...` and swallowing the exception without knowing what it was.
- Storing a reference to a `std::vector` element across a push_back that could reallocate.
- Using `reinterpret_cast` where `static_cast` (or no cast at all) would do.
