# Spec: Dark Mode Toggle

> **Planning note:** This spec was written following the Karpathy principles —
> think before coding, simplest viable approach, explicit out-of-scope, and
> goal-driven tests. No implementation code was written until this spec was
> reviewed.

## Goal

Add a light/dark/auto theme toggle to the app's top navigation bar. Users have
requested the ability to switch themes without changing OS settings. The toggle
should persist across page reloads using `localStorage`.

## Key discoveries

- Bootstrap 5.3+ supports `data-bs-theme="dark|light"` on `<html>` natively —
  no extra CSS needed.
- The project already loads Bootstrap 5.3 from `public/bootstrap.min.css`.
- There is no existing theme logic anywhere in the codebase.
- `localStorage` is available in all target browsers (Chrome 80+, Firefox 78+,
  Safari 14+).

## Design

### Approach

Inline a small `<script>` in `<head>` (before Bootstrap loads) that reads
`localStorage.get('theme')` and sets `data-bs-theme` on `<html>` immediately.
This prevents flash-of-wrong-theme (FOWT).

A dropdown in the navbar offers three options: Light, Dark, Auto (follows OS).
Selecting an option writes to `localStorage` and updates `data-bs-theme`.

### Component layout

```
<nav>
  ...existing nav items...
  <div class="dropdown ms-auto">
    <button>☀ / ☾ / ◑</button>      ← icon changes to match current theme
    <ul class="dropdown-menu">
      <li>Light</li>
      <li>Dark</li>
      <li>Auto</li>
    </ul>
  </div>
</nav>
```

### localStorage key

`theme` → `"light"` | `"dark"` | `"auto"` (default: `"auto"`)

## Files changed

| File | Change |
|---|---|
| `public/index.html` | Add color-mode script to `<head>` and theme dropdown to `<nav>` |

## Out of scope

- Per-user server-side theme preference (no backend changes)
- Custom color palettes beyond Bootstrap's built-in dark/light
- Animation on theme switch (out of scope for v1)
- IE11 support
