# seton

A CLI tool for managing notes embedded directly in source files and saved to a local database.

## Installation

### Download a binary

Download the latest release for your platform from the [releases page](https://github.com/hdirksor/seton/releases).

### Install with Go

```bash
go install github.com/hdirksor/seton@latest
```

### Build from source

```bash
git clone https://github.com/hdirksor/seton
cd seton
go build -o seton .
```

## Note Format

Wrap notes with `~!` and `!~` delimiters and tag them with `#hashtags`:

```
~! This needs refactoring #tech-debt !~
~! Ask team about this approach #question #auth !~
```

## Usage

### Jot a note

Open an interactive form to write and tag a note, saved to `~/.seton/notes.db`:

```bash
seton jot
```

### Search tags interactively

Browse tags interactively and display matching notes:

```bash
seton search
```

### Query notes

List notes from the database. With no arguments, returns all notes. With tags, returns only notes matching every tag (AND logic):

```bash
seton query                  # all notes
seton query todo             # notes tagged #todo
seton query auth bug         # notes tagged both #auth and #bug
```

### Export notes

Query notes by tags and write results to a markdown file in `~/.seton/exports/`:

```bash
seton export todo
seton export auth bug
seton export auth --dir ./notes    # write to a custom directory
```

### Import notes

Review and save notes from a file containing `~!` `!~` blocks:

```bash
seton import <file>
```

## Configuration

Seton reads `~/.seton/config.toml` if present. Defaults:

```toml
[delimiters]
open  = "~!"
close = "!~"

[paths]
root = "~/seton"
```

## License

MIT
