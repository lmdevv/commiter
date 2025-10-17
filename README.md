# Commiter

A CLI tool that generates AI-powered commit messages based on your staged Git
changes stupidly fast.

## Features

- Uses OpenRouter API with Mistral AI model for intelligent commit message
  generation
- Supports both detailed messages with description options and simple one-liners
- Automatically copies generated messages to clipboard

## Installation

### Manual build

```bash
go build -o commiter
```

### Using Nix

```bash
nix build
./result/bin/commiter --help
```

or development shell:

```bash
nix develop
```

## Setup

1. Get an API key from [OpenRouter](https://openrouter.ai/)
2. Initialize the tool: `./commiter --init`
3. Enter your API key when prompted

## Usage

Stage your changes with `git add`, then run:

```bash
commiter  # Generates detailed commit message with options
commiter --simple  # Generates simple one-liner and auto-commits
```

The generated message will be printed and copied to your clipboard. In simple
mode, it will automatically commit with the generated message.
