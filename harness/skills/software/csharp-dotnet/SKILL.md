---
name: csharp-dotnet
domain: software
level: technology
description: .NET Framework vs. modern .NET awareness, target-framework inspection, P/Invoke and managed/unmanaged boundaries, x86/x64 compatibility — the platform facts that cause most confusing C# bugs.
tags: [csharp, dotnet, "dotnet framework", nuget, csproj]
triggers: [csharp, "c#", dotnet, ".net framework", "target framework", csproj, nuget, pinvoke, "p/invoke"]
version: "1.0.0"
requires: []
recommends: []
capabilities: []
conflicts: []
when_to_use: Any C#/.NET project — desktop, service, console, or library — before assuming behavior that's actually specific to one .NET flavor applies universally.
when_not_to_use: A task with no .NET involved at all, or one that's purely about a specific UI framework (see software/winforms) rather than the language/runtime itself.
---

# Skill: csharp-dotnet

## Purpose

The .NET-platform facts that cause confusion when a codebase mixes .NET Framework and modern
.NET assumptions, or crosses a native/managed boundary — before any UI-framework-specific
guidance applies.

## Method

1. **.NET Framework and modern .NET (5/6/7/8+) are not the same runtime.** They share the C#
   language but differ in available APIs, deployment model (GAC vs. self-contained/
   framework-dependent), and default project SDK style. Check the `.csproj`'s
   `<TargetFramework>`/`<TargetFrameworks>` (`net48` vs. `net8.0-windows`, etc.) before assuming
   a NuGet package, API, or build behavior is available — a package targeting `netstandard2.0`
   works on both; one requiring `net6.0+` does not run on .NET Framework at all.
2. **SDK-style vs. legacy `.csproj` changes what "add a reference" even means.** A legacy
   `.csproj` (still common in older WinForms projects) lists every source file and reference
   explicitly; an SDK-style project (`<Project Sdk="Microsoft.NET.Sdk">`) globs source files and
   resolves references via `PackageReference`. Mixing assumptions from one style while editing
   the other silently breaks the build or misses files.
3. **x86 vs. x64 vs. AnyCPU matters most at the P/Invoke boundary.** A managed assembly built
   `AnyCPU` runs as whichever bitness the host process is, but a `DllImport` to a native DLL
   only resolves if that native DLL's own bitness matches the running process — "works on my
   machine" P/Invoke failures are very often a bitness mismatch, not a missing DLL.
4. **P/Invoke marshaling is a real source of silent corruption, not just crashes.** A struct
   layout, string marshaling (`CharSet`, ANSI vs. Unicode), or calling-convention mismatch
   between the `DllImport` signature and the native function's real signature can produce wrong
   values or heap corruption without an immediate exception — verify the native header/docs
   directly rather than trusting a plausible-looking existing signature.
5. **The managed/unmanaged boundary owns its own lifetime rules.** A native handle or pointer
   returned across P/Invoke isn't tracked by the .NET garbage collector — it must be explicitly
   freed via whatever the native API's own cleanup function is (or wrapped in a `SafeHandle`),
   not left to `Dispose`/finalization assumptions that only apply to managed objects.
6. **Testing a legacy desktop app often means testing around the framework, not through it** —
   .NET Framework-era codebases frequently have little to no dependency injection, making
   direct unit tests hard; look for a seam (extract the logic being changed into a plain class)
   rather than trying to unit-test a form or a static-heavy entry point directly.

## Anti-patterns

- Assuming a NuGet package or API available in modern .NET is available on .NET Framework (or
  vice versa) without checking the actual `TargetFramework`.
- Copying a `DllImport` signature from an online example without verifying it against the
  actual native header for struct layout, string marshaling, and calling convention.
- Building `AnyCPU` and assuming P/Invoke will "just work" regardless of the native DLL's own
  bitness.
