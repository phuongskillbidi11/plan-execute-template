# src-map — what already exists

### `api/`

What it does: `Handle(path string) string` — dispatches an inbound HTTP request path to a
canned response. This is the fixture's HTTP-facing layer.

### `db/`

What it does: `Get`/`Set` — a trivial in-memory key-value store, no persistence, no
concurrency control.

### `cache/`

What it does: `Lookup`/`Store` — a minimal fixed-size cache stub, separate from `db/`.

### `auth/`

What it does: `ValidToken(token string) bool` — validates a request token. Currently
accepts any string, including an empty one — this is the package this fixture's benchmark
request concerns.

### `utils/`

What it does: `TrimAll(items []string) []string` — trims whitespace from a slice of
strings. Shared helper, not specific to any other package here.

### `cli/`

What it does: wires `api`/`auth` together into a runnable command. Not itself a package
other code imports.
