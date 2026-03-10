# Partial import sessions

## The problem

`seton import` saves notes to the database one at a time as the user steps through blocks. If the user aborts mid-session (ctrl+c), any notes already saved remain in the database. The source file is left untouched on disk.

This creates a re-import hazard: running `seton import` on the same file a second time will show already-saved blocks again, and pressing ctrl+s will create duplicates.

## Options

### 1. Convention — user skips already-saved blocks

No code changes. On re-import the user presses ctrl+k to skip blocks they already saved.

Breaks down quickly if the user doesn't remember what was saved, or if there are many blocks.

### 2. Remove saved blocks from the source file in-place

After each ctrl+s, delete that block (including its delimiters) from the source file. On abort, the file reflects exactly what is left to import. Re-importing is safe because saved blocks no longer exist in the file.

Requires in-place file mutation while the program is running. Must track byte offsets or re-parse after each save, and handle partial writes safely. More I/O surface area for error.

### 3. Content-hash deduplication

Store a hash of each note's text in the database. On import, skip or warn when a block matches an existing note.

Requires a schema change, hash storage, and UI for the "already seen" case. Does not handle intentional duplicates (the same text saved twice on purpose) gracefully.

### 4. Session rollback on abort

Track the database IDs of notes saved during the current session. If the user aborts, delete those notes. The database is restored to its state before the import started.

Requires `SaveNote` to return the inserted ID, a `savedIDs []int64` field on the model, and a `DeleteNotes` call in the abort path. Changes are localized and no file I/O or schema changes are needed.

Does not protect against hard kills (process killed, power loss). Notes saved before a crash are not rolled back. If that matters, a session table in the database would be needed, which adds significant complexity.

## Recommendation

Option 4 is the right default. It is the simplest to implement correctly, handles the common abort case, and leaves no observable side effects for the user. The hard-kill edge case is acceptable — it is unlikely in normal use and the workaround (re-import, skip duplicates) is straightforward.

Option 2 could be layered on top of option 4 later if users want the source file to reflect import progress visually.
