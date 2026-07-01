# mdq — a jq-like tool for markdown

`mdq` lets you format (in terminal), filter and format markdown documents from the command line. Think `jq`, but for markdown.

## Features

- **Query** — extract headings, code blocks, links, images, tables, lists, blockquotes
- **Filter** — `h1`–`h6` for heading levels, `code[lang=…]` for language-specific code blocks
- **Format** — terminal ANSI rendering (default), plain markdown, JSON, text
- **Pipe-friendly** — works with stdin/stdout, chains with `|`
- **Zero config** — single binary, no dependencies

## Installation

```bash
# Download the latest release binary
curl -Lo mdq https://github.com/malopeza1226/mdq/releases/latest/download/mdq-linux-amd64
chmod +x mdq
sudo mv mdq /usr/local/bin/

# Or build from source
git clone https://github.com/malopeza1226/mdq.git
cd mdq
go build -o mdq .           # requires Go 1.21+
```

## Usage

```
mdq [query] [file]
cat file.md | mdq [query]
```

### Queries

| Query | Description |
|-------|-------------|
| `.` | Render full document with terminal formatting |
| `.headings` | Extract all headings |
| `.paragraphs` | Extract all paragraphs |
| `.code_blocks` | Extract all code blocks |
| `.links` | Extract all links |
| `.images` | Extract all images |
| `.lists` | Extract all lists |
| `.blockquotes` | Extract all blockquotes |
| `.tables` | Extract all tables |
| `h1` … `h6` | Filter headings by level |
| `code[lang=go]` | Filter code blocks by language |
| `count` | Count elements by type |
| `stats` | Show document statistics |

### Options

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--plain` | Output plain markdown (no ANSI) |
| `-h`, `--help` | Show help |

### Examples

```bash
# Render a document with terminal formatting
mdq README.md

# Extract all headings (plain text)
mdq .headings README.md

# Extract all links as JSON
cat docs/article.md | mdq --json .links

# Count elements in a document
mdq count CHANGELOG.md

# Show document statistics
mdq stats DOCUMENTATION.md

# Filter heading level 2
mdq h2 README.md

# Filter code blocks by language
mdq 'code[lang=python]' docs/examples.md

# Output full document as plain markdown
mdq --plain README.md

# Query chaining with stdin
cat notes.md | mdq .headings

# Pipe: extract tables → count
mdq .tables report.md | mdq count

# Single-argument mode: file path or query
mdq README.md              # render file
mdq .links README.md       # query on file
cat README.md | mdq .links # stdin mode
```

## Terminal Output

`mdq` renders markdown to the terminal with:

- **Headings** — bold + color, no `#` noise
- **Inline formatting** — **bold**, *italic*, `code`, underlined links with dim URL
- **Tables** — box-drawing characters (┌─┬┐), column alignment (left/center/right), responsive width
- **Code blocks** — line numbers, language tag header, green theme
- **Blockquotes** — │ prefix, preserved inline formatting
- **Horizontal rules** — dim line across terminal width
- **Hard/soft line breaks** — preserved as in source

## JSON Output

```json
[
  {
    "Type": "heading",
    "Depth": 1,
    "Level": 1,
    "Content": "Heading 1",
    "Children": null
  },
  {
    "Type": "table",
    "Depth": 0,
    "Content": "",
    "Children": [
      {
        "Type": "table_header",
        "Children": [
          {
            "Type": "table_cell",
            "Content": "Left",
            "Attributes": {"align": "left"},
            "Children": null
          },
          ...
        ]
      }
    ]
  }
]
```

## Why mdq?

- **jq for markdown** — familiar query syntax for document exploration
- **Single binary** — no Node, Python, or Ruby runtime needed
- **CI-friendly** — pipe output to other tools, parse JSON for automation
- **Terminal-native** — beautiful ANSI rendering for everyday use

## License

MIT
