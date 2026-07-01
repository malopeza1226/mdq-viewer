package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	outputJSON := flag.Bool("json", false, "output as JSON")
	outputFormat := flag.String("format", "", "output format: text, json, markdown, plain (default: auto)")
	showHelp := flag.Bool("help", false, "show help")
	flag.BoolVar(showHelp, "h", false, "show help")
	outputPlain := flag.Bool("plain", false, "output plain markdown (no ANSI)")
	flag.Parse()

	if *showHelp {
		printHelp()
		return
	}

	args := flag.Args()

	var query string
	var input []byte
	var err error

	switch len(args) {
	case 0:
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			printHelp()
			os.Exit(1)
		}
		query = "."
		input, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case 1:
		if fi, statErr := os.Stat(args[0]); statErr == nil && fi.Mode().IsRegular() {
			query = "."
			input, err = os.ReadFile(args[0])
		} else {
			query = args[0]
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) != 0 {
				fmt.Fprintln(os.Stderr, "error: stdin required when query is given without a file")
				os.Exit(1)
			}
			input, err = io.ReadAll(os.Stdin)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		query = args[0]
		input, err = os.ReadFile(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	doc, err := parseMarkdown(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	result, err := executeQuery(doc, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmtLen := *outputFormat
	if fmtLen == "" {
		if *outputJSON {
			fmtLen = "json"
		} else if *outputPlain {
			fmtLen = "plain"
		} else {
			fmtLen = detectFormat(query)
		}
	}

	output, err := formatResult(result, fmtLen)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(output)
}

func detectFormat(query string) string {
	switch {
	case query == ".":
		return "terminal"
	case query == "count" || query == "stats":
		return "text"
	default:
		return "text"
	}
}

func executeQuery(doc *Document, query string) (interface{}, error) {
	parts := strings.Split(query, "|")
	var current interface{} = doc.Nodes

	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch {
		case part == ".":
		case part == "count":
			counts := countByType(doc.Nodes)
			return counts, nil
		case part == "stats":
			s := calculateStats(doc.Nodes)
			return s, nil
		case len(part) == 2 && part[0] == 'h' && part[1] >= '1' && part[1] <= '6':
			level := int(part[1] - '0')
			current = filterByLevel(asNodes(current), level)
		case strings.HasPrefix(part, "."):
			key := part[1:]
			current = filterByType(asNodes(current), key)
		case strings.Contains(part, "["):
			current = filterByAttr(asNodes(current), part)
		default:
			return nil, errors.New("unknown query: " + part)
		}
	}
	return current, nil
}

func asNodes(v interface{}) []Node {
	if nodes, ok := v.([]Node); ok {
		return nodes
	}
	return nil
}

func printHelp() {
	fmt.Println(`mdq — a jq-like tool for markdown

Usage:
  mdq [query] [file]
  cat file.md | mdq [query]

Queries:
  .                    render document with ANSI terminal formatting
  .headings            extract all headings
  .paragraphs          extract all paragraphs
  .code_blocks         extract all code blocks
  .links               extract all links
  .images              extract all images
  .lists               extract all lists
  .blockquotes         extract all blockquotes
  .tables              extract all tables
  h1, h2, ..., h6      filter headings by level
  code[lang=go]        filter code blocks by language
  count                count elements by type
  stats                show document statistics

Options:
  --json               output as JSON
  --plain              output plain markdown (no ANSI)
  -h, --help           show this help

Examples:
  mdq README.md                          # terminal formatted
  mdq --plain README.md                  # plain markdown
  mdq .headings README.md
  cat article.md | mdq .links
  mdq --json . README.md
  mdq h2 README.md
  mdq 'code[lang=python]' README.md`)
}
