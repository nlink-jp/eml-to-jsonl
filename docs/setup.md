# Setup Guide

## Prerequisites

- Go 1.22 or later

No LLM API is required — eml-to-jsonl is a pure parser with no network calls.

## Installation

```sh
git clone https://github.com/nlink-jp/eml-to-jsonl.git
cd eml-to-jsonl
make build
# Add dist/ to PATH or copy dist/eml-to-jsonl to a directory on PATH
```

## Git hooks

```sh
make setup
```

Installs `pre-commit` (vet + lint) and `pre-push` (full check) hooks.

## Quick start

```sh
# Parse a single EML file
eml-to-jsonl message.eml

# Parse all EML files in a directory
eml-to-jsonl ~/Downloads/exported-mail/

# Pretty-print for inspection
eml-to-jsonl --pretty message.eml | head -40

# Pipe into lite-llm
eml-to-jsonl inbox/ | lite-llm -p "List the sender and subject of each email."
```
