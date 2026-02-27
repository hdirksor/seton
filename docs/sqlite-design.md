# SQLite Design Notes

## Core concerns

The current `Open()` runs `CREATE TABLE IF NOT EXISTS` inline. That works for initial creation but gives no path to evolve the schema — adding columns, changing constraints, or renaming things requires manual intervention.

## Migration strategies

### Option 1: Versioned migrations in code

Keep a slice of SQL strings, track the current version in a `schema_version` table, and apply only unapplied migrations on startup.

```go
var migrations = []string{
    `CREATE TABLE notes (id INTEGER PRIMARY KEY AUTOINCREMENT, text TEXT NOT NULL, tags TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
    `ALTER TABLE notes ADD COLUMN source_file TEXT`, // future
}
```

Pros: no external tools, zero deps, full control, easy to test.
Cons: you write the SQL yourself; no rollback story.

This is the right choice for a small CLI tool. The complexity ceiling is low enough that a hand-rolled migrator stays readable.

### Option 2: Embed SQL migration files

Same idea, but migrations live in `store/migrations/001_init.sql`, `002_add_source.sql`, etc., embedded with `//go:embed`. The Go code iterates the files in order.

Pros: SQL in `.sql` files (editor highlighting, cleaner diffs), same logic as option 1.
Cons: slightly more setup. Marginal benefit for a small schema.

### Option 3: Migration library (goose, golang-migrate)

Drop in `pressly/goose` or `golang-migrate/migrate`. They handle versioning, up/down, and CLI tooling.

Pros: battle-tested, rollback support, good for teams.
Cons: overkill here — adds a dependency and ceremony for what will likely be 3-4 migrations total.

## Tag querying

Tag querying with space-separated strings in a single column (`"#todo #refactor"`) works fine for simple cases but gets awkward:

- `WHERE tags LIKE '%#todo%'` works but can false-match (`#todo-later`)
- Multi-tag AND requires chaining LIKE clauses
- No index benefit

### Option A: Keep space-separated, use exact token matching

```sql
WHERE ' ' || tags || ' ' LIKE '% #todo %'
```

Cheap, no schema change, acceptable for a personal tool with hundreds of notes.

### Option B: Normalize tags into a separate table

```sql
notes(id, text, created_at)
note_tags(note_id, tag)
```

Proper relational model, indexed, clean multi-tag queries. Requires a migration if adopted later.

## Recommendation

Given the scale (personal CLI, likely hundreds of notes):

1. **Option 1** for migrations — hand-rolled versioned slice in `store.go`, `schema_version` table, runs on every `Open()`. Simple, no deps, testable.
2. **Option A** for tags now — space-separated is fine to ship. Add a comment in the code noting option B if querying becomes painful.

The migration pattern gives an escape hatch to add the tags table later as a migration without breaking existing data.
