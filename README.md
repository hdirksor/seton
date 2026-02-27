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

### Extract notes

Walk a directory tree and extract all notes into `.archive/` YAML files:

```bash
seton extract <directory>
```

For each source file containing notes, a corresponding `<filename>.yaml` is created in a `.archive/` directory alongside it.

### Lint a file

Check a file for malformed note syntax:

```bash
seton lint <file>
```

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
