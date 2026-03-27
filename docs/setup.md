# Setup Guide

## Prerequisites

- Go 1.22 or later

No LLM API is required — lite-eml is a pure parser with no network calls.

## Installation

```sh
git clone https://github.com/nlink-jp/lite-eml.git
cd lite-eml
make build
# Add bin/ to PATH or copy bin/lite-eml to a directory on PATH
```

## Git hooks

```sh
make setup
```

Installs `pre-commit` (vet + lint) and `pre-push` (full check) hooks.

## Quick start

```sh
# Parse a single EML file
lite-eml message.eml

# Parse all EML files in a directory
lite-eml ~/Downloads/exported-mail/

# Pretty-print for inspection
lite-eml --pretty message.eml | head -40

# Pipe into lite-llm
lite-eml inbox/ | lite-llm -p "List the sender and subject of each email."
```
