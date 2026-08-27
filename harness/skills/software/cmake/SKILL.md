---
name: cmake
domain: software
level: technology
description: Targets, include/link propagation, toolchain files, and out-of-source builds — the CMake facts that cause most confusing build-configuration bugs.
tags: [cmake, "cmakelists.txt", "toolchain file"]
triggers: [cmake, "cmakelists.txt", "toolchain file", find_package, "out-of-source"]
version: "1.0.0"
requires: []
recommends: []
capabilities: []
conflicts: []
when_to_use: Configuring, debugging, or extending a CMake-based build.
when_not_to_use: A project whose build isn't CMake-based (Makefiles, MSBuild without CMake,
  Bazel, etc.) — the target/propagation model here is CMake-specific.
---

# Skill: cmake

## Purpose

The target-based mental model and the handful of mechanisms (propagation, toolchain files,
out-of-source builds) that cause most "it built differently on the other machine" bugs.

## Method

1. **Modern CMake is target-based, not variable-based.** `target_include_directories`,
   `target_link_libraries`, `target_compile_definitions` attach properties to a specific target,
   not the whole directory tree — prefer these over the older global `include_directories()`/
   `link_libraries()` forms, which apply to every target in and below the current
   `CMakeLists.txt` regardless of whether it needs them.
2. **`PRIVATE`/`PUBLIC`/`INTERFACE` control propagation, and picking the wrong one is a common
   silent-bloat or silent-break bug.** `PRIVATE` (needed to build this target, not exposed to
   its consumers), `PUBLIC` (needed by both this target and anything that links it), `INTERFACE`
   (needed only by consumers, not this target itself — e.g. a header-only library). Marking an
   implementation-only dependency `PUBLIC` leaks it into every downstream consumer's build;
   marking a genuinely-needed dependency `PRIVATE` on a library target breaks consumers that
   actually need it too.
3. **A toolchain file (`-DCMAKE_TOOLCHAIN_FILE=...`) is how cross-compilation is configured**,
   not a compiler flag passed directly — it sets `CMAKE_SYSTEM_NAME`, the compiler paths, and
   `CMAKE_SYSROOT` before the first `project()` call runs its compiler checks. Passing
   `CMAKE_C_COMPILER`/`CMAKE_CXX_COMPILER` alone without a toolchain file often still runs
   host-targeted compiler checks and picks up host libraries by accident.
4. **Build types (`CMAKE_BUILD_TYPE=Debug|Release|RelWithDebInfo|MinSizeRel`) control
   optimization/debug-info flags, but only for single-configuration generators** (Makefiles,
   Ninja) — multi-configuration generators (Visual Studio, Xcode) ignore
   `CMAKE_BUILD_TYPE` and instead select the configuration at build time
   (`cmake --build . --config Release`). Setting `CMAKE_BUILD_TYPE` and seeing no effect is
   usually this.
5. **`find_package()` has two real modes** — Config mode (finds a `<Package>Config.cmake` the
   dependency itself ships, the more reliable modern path) and Module mode (uses CMake's own
   bundled `Find<Package>.cmake`, which can lag behind a dependency's actual layout). A
   `find_package()` failure message naming "Config mode" vs. not finding a Module-mode script
   points at a different fix (set `<Package>_DIR` vs. install the dependency somewhere CMake's
   bundled finder expects).
6. **Out-of-source builds (a separate `build/` directory, never the source tree itself) aren't
   just a convention** — some generated files (a re-run `cmake` in the source tree, a stray
   `CMakeCache.txt`) can silently poison a later out-of-source build if the source tree was ever
   configured in-place even once; delete any in-source `CMakeFiles/`/`CMakeCache.txt` before
   trusting a fresh out-of-source configure.

## Anti-patterns

- Marking a dependency `PUBLIC` "to be safe," leaking implementation details into every
  consumer's include path and link line.
- Passing compiler paths directly instead of a toolchain file for cross-compilation, then being
  surprised when host headers/libraries get picked up.
- Setting `CMAKE_BUILD_TYPE` on a multi-configuration generator (Visual Studio/Xcode) and
  expecting it to have any effect.
