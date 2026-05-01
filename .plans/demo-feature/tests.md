# Tests: Dark Mode Toggle

> **Testing note:** Tests follow the Karpathy goal-driven principle — each test
> verifies user-visible behavior with an exact Pass/Fail condition. "It compiles"
> is not a passing test. Every test must be independently runnable and
> unambiguously verifiable.

Run each test after completing the corresponding task. Stop and report on first
failure. T1–T2 are static/build checks. T3–T7 require a running browser.

---

## T1 — Script placement (after Task 1)

```bash
grep -n "data-bs-theme\|localStorage" public/index.html | head -20
```

**Pass:** At least two lines appear — one inside `<head>` (line number < the
`<link rel="stylesheet">` line) and one inside `<body>` (the switcher script).

**Fail:** No matches, or the `<head>` match appears *after* the Bootstrap
`<link>` — the initialization script is in the wrong place.

---

## T2 — Build passes (after Task 4)

```bash
[BUILD_COMMAND]
```

**Pass:** Build exits with code 0 and no `error:` lines in output.

**Fail:** Copy the full error output and report it. Do not attempt to fix it —
report and stop.

---

## T3 — Page loads without flash-of-wrong-theme (manual)

1. Open the browser DevTools → Application → Local Storage.
2. Set `theme` to `"dark"`.
3. Hard-reload the page (Ctrl+Shift+R).

**Pass:** The page background is dark immediately on load — no white flash
before Bootstrap styles apply.

**Fail:** A white background appears for >100 ms before switching to dark.
Screenshot the Network waterfall and report which resource loads last.

---

## T4 — Dropdown renders correctly (manual)

Open `[RUN_COMMAND]` and navigate to the page.

**Pass (all must be true):**
- [ ] Theme toggle button is visible in the top-right of the navbar
- [ ] Clicking the button opens a dropdown with three options: Light, Dark, Auto
- [ ] Current icon matches the active theme (☀ for Light, ☾ for Dark, ◑ for Auto)

**Fail:** Button not visible, dropdown doesn't open, or icons are wrong.
Note which criterion failed.

---

## T5 — Theme switching works (manual)

With the page open:

1. Click the dropdown → select **Dark**.
2. Verify: page background switches to dark Bootstrap theme.
3. Click the dropdown → select **Light**.
4. Verify: page switches to light theme.
5. Click the dropdown → select **Auto**.
6. Verify: theme matches OS preference.

**Pass:** Each selection immediately changes `data-bs-theme` on `<html>` (verify
in DevTools → Elements) and the visual theme updates.

**Fail:** Note which step failed. Check DevTools → Console for JS errors.

---

## T6 — Preference persists across reload (manual)

1. Select **Dark** from the dropdown.
2. Reload the page (Ctrl+R, not hard-reload).

**Pass:** Page loads in dark mode immediately (no flash). The dropdown icon shows ☾.

**Fail:** Page loads in light/auto mode, ignoring the stored preference.
Check DevTools → Application → Local Storage → key `theme` value.

---

## T7 — Auto mode reacts to OS change (manual)

> Requires a system that can toggle dark/light OS preference easily
> (macOS System Preferences, Windows Settings, or Chrome DevTools device emulation).

1. Select **Auto** from the dropdown.
2. Switch your OS to **dark** mode.

**Pass:** Page theme switches to dark without reload.

3. Switch your OS to **light** mode.

**Pass:** Page theme switches to light without reload.

**Fail:** Theme does not update until page reload.
Note which OS/browser combination was tested.
