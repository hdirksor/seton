# Testing Strategy

## Layers

| Layer | Location | What it covers |
|---|---|---|
| Unit | `commands/*_test.go`, `config/*_test.go`, `store/*_test.go` | Pure functions, bubbletea model logic, DB helpers |
| Integration | `integration/integration_test.go` | Full cobra command execution against a real SQLite DB |
| Smoke / E2E | (not yet implemented) | Binary compiles and basic commands exit cleanly |
| TUI | Not automated — use VHS for demos | `jot`, `search`, `import` wizard |

---

## Integration tests (Option A: in-process cobra)

`InitRootCmd()` is called directly in test code. stdout/stderr are redirected via cobra's `SetOut`/`SetErr`. The DB and config are isolated per test with:

```go
t.Setenv("HOME", t.TempDir())
```

This causes `store.Open()` and `config.Load()` to use a fresh, throwaway directory every test run.

### Running locally

```bash
go test ./integration/...
```

### Running in CI (GitHub Actions)

Integration tests use no external dependencies beyond Go and the SQLite driver already in `go.mod`, so they run in the standard `go test` step:

```yaml
- run: go test ./...
```

No extra setup required.

### What is covered

| Test | Command | Scenario |
|---|---|---|
| `TestQueryEmptyDB` | `query` | Returns "No notes found." with empty DB |
| `TestQueryAllNotes` | `query` | Returns all notes when no tags given |
| `TestQueryByTag` | `query <tag>` | Filters notes to matching tag |
| `TestQueryByTagNoMatch` | `query <tag>` | Returns "No notes found." when no match |
| `TestQueryNormalizeTag` | `query todo` | Accepts tags without leading `#` |
| `TestExportWritesFile` | `export --dir <dir> <tag>` | Creates a markdown file in the given dir |
| `TestExportNoNotes` | `export <tag>` | Returns an error when no notes match |

### What is not covered

- `jot`, `search`, `import` — these launch bubbletea programs and take over stdin/stdout. They require a pseudo-terminal (pty) to drive and are not tested here.
- `extract` — operates on source files with `~!`/`!~` delimiters; covered by parser unit tests.

---

## Other options considered

### Option B: subprocess tests

Build the binary in `TestMain`, shell out with `exec.Command` per test.

- Pros: black-box, tests the real binary users get
- Cons: slower (build step per CI run), no coverage attribution, TUI still untestable

### Option C: testscript (rogpeppe/go-internal)

Write declarative `.txtar` scripts. Used in the Go stdlib for CLI testing.

- Pros: readable scripts, good for documenting expected behavior
- Cons: extra dependency, learning curve for `.txtar` format

### Option D: VHS (charmbracelet/vhs)

Write `.tape` scripts that simulate keypresses against the real binary.

- Pros: drives actual TUI keystrokes, doubles as demo documentation
- Cons: requires VHS installed, assertions are limited, slow in CI, primarily useful for demos
