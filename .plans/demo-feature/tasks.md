# Tasks: Dark Mode Toggle

Each task must be completed and its test (see `tests.md`) must pass before
moving to the next. Mark `[x]` when done.

---

## Task 1 — Add color-mode initialization script to `<head>`

- [ ] **1.1** Open `public/index.html`. Locate the `<head>` section. Find the
  line that contains `<meta name="viewport"` (line ~5).

  Insert the following `<script>` block **immediately after** that line
  (before any `<link>` tags, so it runs before Bootstrap loads):

  ```html
  <script>
    (() => {
      'use strict'
      const stored = localStorage.getItem('theme')
      const theme = stored === 'light' || stored === 'dark' ? stored
        : window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
      document.documentElement.setAttribute('data-bs-theme', theme)
    })()
  </script>
  ```

- [ ] **1.2** Verify the script is placed before `<link rel="stylesheet"
  href="bootstrap.min.css">` — it must execute before Bootstrap parses
  `data-bs-theme`.

---

## Task 2 — Add theme dropdown to the navbar

- [ ] **2.1** Open `public/index.html`. Locate the closing `</nav>` tag.
  Immediately before it, insert the following dropdown:

  ```html
  <div class="dropdown ms-auto">
    <button class="btn btn-sm btn-outline-secondary dropdown-toggle"
            id="theme-toggle"
            data-bs-toggle="dropdown"
            aria-expanded="false"
            title="Toggle theme">
      <span id="theme-icon">◑</span>
    </button>
    <ul class="dropdown-menu dropdown-menu-end" aria-labelledby="theme-toggle">
      <li><button class="dropdown-item" data-theme="light">☀ Light</button></li>
      <li><button class="dropdown-item" data-theme="dark">☾ Dark</button></li>
      <li><button class="dropdown-item" data-theme="auto">◑ Auto</button></li>
    </ul>
  </div>
  ```

---

## Task 3 — Add theme-switching JavaScript

- [ ] **3.1** Open `public/index.html`. Locate the closing `</body>` tag.
  Immediately before it (after `<script src="bootstrap.bundle.min.js">`),
  insert the following inline script:

  ```html
  <script>
    (function () {
      const ICONS = { light: '☀', dark: '☾', auto: '◑' }

      function applyTheme(choice) {
        const resolved = choice === 'auto'
          ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
          : choice
        document.documentElement.setAttribute('data-bs-theme', resolved)
        document.getElementById('theme-icon').textContent = ICONS[choice] || ICONS.auto
        localStorage.setItem('theme', choice)
      }

      // Set icon to match stored preference on load
      const stored = localStorage.getItem('theme') || 'auto'
      document.getElementById('theme-icon').textContent = ICONS[stored] || ICONS.auto

      // Wire up dropdown buttons
      document.querySelectorAll('[data-theme]').forEach(btn => {
        btn.addEventListener('click', () => applyTheme(btn.dataset.theme))
      })

      // React to OS theme changes when "auto" is selected
      window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        if ((localStorage.getItem('theme') || 'auto') === 'auto') {
          applyTheme('auto')
        }
      })
    })()
  </script>
  ```

---

## Task 4 — Verify the HTML is valid

- [ ] **4.1** Run the HTML validator to confirm no syntax errors were
  introduced by the insertions:

  ```bash
  npx html-validate public/index.html
  ```

  If `html-validate` is not installed, use:

  ```bash
  [BUILD_COMMAND]
  ```

  and confirm the build exits with code 0.
