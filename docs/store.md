# store package

The `store` package is responsible for all database access: opening the database,
keeping its schema up to date, saving notes, and querying them.

The database lives at `~/.seton/notes.db` and is created automatically on first run.

---

## Schema

Two tables hold all data.

**`notes`** — one row per note.

| column       | type     | description                          |
|--------------|----------|--------------------------------------|
| `id`         | INTEGER  | auto-incrementing primary key        |
| `text`       | TEXT     | the body of the note                 |
| `created_at` | DATETIME | set automatically by SQLite on insert|

**`note_tags`** — one row per tag per note.

| column    | type    | description                              |
|-----------|---------|------------------------------------------|
| `note_id` | INTEGER | foreign key referencing `notes.id`       |
| `tag`     | TEXT    | a single tag, e.g. `#todo`               |

A `UNIQUE(note_id, tag)` constraint prevents the same tag appearing twice on one note.

This is a deliberate split from storing tags as a space-separated string in a single
column. Separate rows make querying exact tag matches clean and reliable (no LIKE tricks).

---

## Migrations

The schema has changed over the lifetime of the project. The migration system ensures
that any existing database is brought up to date automatically whenever `Open()` is
called — whether that database was created yesterday or six months ago.

### How it works

A third table, `schema_version`, records which migrations have already run:

```
schema_version
--------------
version INTEGER
```

Each migration is numbered starting at 1. On startup, `applyMigrations` reads the
highest version recorded in `schema_version`. Any migration with a higher number
is then applied in order.

Each migration runs inside its own **transaction**. If anything fails, the transaction
is rolled back and the error is returned — the database is left exactly as it was.
If the migration succeeds, the version number is written to `schema_version` inside
the same transaction before it commits. This means a migration is never recorded as
done unless it actually succeeded.

### The four migrations

**Migration 1** — creates the `notes` table with a `tags TEXT` column (the original
schema, where tags were stored as a space-separated string like `"#todo #refactor"`).

**Migration 2** — creates the `note_tags` table (the current schema for tags).

**Migration 3** — a Go function that reads every existing note's `tags` string, splits
it on whitespace, and inserts each token as a row in `note_tags`. This moves legacy
data from the old format into the new one. It runs inside the same transaction as
migration 2's version record.

**Migration 4** — drops the `tags` column from `notes` now that the data has been
moved. After this migration the column no longer exists.

### Adding a future migration

Append a new entry to the `migrations` slice in `store.go`. Give it either a SQL
statement, a Go function (`fn`), or both. The version number is determined by position
in the slice — you never set it manually.

```go
var migrations = []migration{
    // existing entries...
    {sql: `ALTER TABLE notes ADD COLUMN source_file TEXT`}, // migration 5
}
```

On the next run, any database not yet at version 5 will have this applied.

---

## Saving a note

`SaveNote(db, text, userTags)` does three things atomically:

1. Extracts any `#tag` patterns found in the note body using a regular expression.
2. Merges those with the tags the user explicitly typed in the form, deduplicating
   and adding a `#` prefix to any tag missing one.
3. Inserts the note text into `notes`, then inserts each tag into `note_tags`.

All three steps happen inside a single transaction, so the database never ends up
with a note that has no tags or tags that point to a non-existent note.

---

## Querying notes

`QueryNotes(db, tags)` returns notes that match **all** of the provided tags (AND logic).
With no tags, all notes are returned.

The SQL uses **INTERSECT** to enforce AND logic:

```sql
-- for tags ["#auth", "#bug"]
SELECT n.id, n.text, n.created_at, GROUP_CONCAT(nt.tag, ' ')
FROM notes n
LEFT JOIN note_tags nt ON nt.note_id = n.id
WHERE n.id IN (
    SELECT note_id FROM note_tags WHERE tag = '#auth'
    INTERSECT
    SELECT note_id FROM note_tags WHERE tag = '#bug'
)
GROUP BY n.id
ORDER BY n.created_at DESC
```

INTERSECT returns only the note IDs that appear in **both** subquery results, which
is equivalent to requiring every tag to be present. One subquery is added per tag.

`GROUP_CONCAT` collects all tags for each note back into a single string so they
can be returned alongside the note text. The Go code then splits that string back
into a `[]string` to populate `Note.Tags`.
