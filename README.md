# seton

A CLI for managing notes. This tool is designed for a tag-centric approach to note taking and referencing. Seton provides three basic modes of functionality in note taking, querying and exporting. Seton stores your notes in a local SQLite database and provides a simple interfaces to quickly take a note, query your note collection using tags, and to export the results of those queries.

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

## License

MIT

