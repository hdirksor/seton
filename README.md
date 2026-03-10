# seton

A CLI tool for managing code notes embedded directly in source files.

## Overview

Seton lets you embed tagged notes anywhere in your codebase using a simple delimiter syntax. Notes are extracted and organized into YAML files, making it easy to track tasks, questions, or any annotations alongside your code.

## Note Format

Wrap notes with `~!` and `!~` delimiters and tag them with `#hashtags`:

```
~! This needs refactoring #tech-debt !~
~! Ask team about this approach #question #auth !~
```

## Installation

```bash
go install github.com/hdickson/seton@latest
```

Or build from source:

```bash
git clone https://github.com/hdickson/seton
cd seton
go build -o seton .
```

## Usage

### Jot a note

Open an interactive terminal form to write a note and tag it, saved to a local SQLite database (`~/.seton/notes.db`):

```bash
seton jot
```

### Search tags interactively

Open an interactive tag browser to select tags, then display matching notes:

```bash
seton search
```

### Query notes

List notes from the database. With no arguments, returns all notes. With tags, returns only notes that match every tag (AND logic):

```bash
seton query                        # all notes
seton query todo                   # notes tagged #todo
seton query auth bug               # notes tagged both #auth and #bug
```

### Export notes

Query notes by tags and write the results to a markdown file in `~/.seton/exports/`:

```bash
seton export todo
seton export auth bug
seton export auth --dir ./notes    # write to a custom directory
```

### Extract notes

Walk a directory tree and extract all embedded notes into `.archive/` YAML files:

```bash
seton extract <directory>
```

For each source file containing notes, a corresponding `<filename>.yaml` is created in a `.archive/` directory alongside it.

## Output Format

Extracted notes are written as YAML:

```yaml
- rawtext: ~! Needs refactoring #tech-debt !~
  text: Needs refactoring
  tags:
    - '#tech-debt'
  file: main.go
```

## Development

```bash
go test ./...
```

## License

MIT
