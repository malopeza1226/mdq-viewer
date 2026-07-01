package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	italic = "\033[3m"
	uline  = "\033[4m"

	fgGreen   = "\033[32m"
	fgBlue    = "\033[34m"
	fgMagenta = "\033[35m"
	fgBrBlack = "\033[90m"
)

func visibleLen(s string) int {
	n := 0
	for i := 0; i < len(s); {
		if s[i] == '\033' {
			j := i + 1
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				i = j + 1
				continue
			}
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		n += runeWidth(r)
		i += size
	}
	return n
}

func runeWidth(r rune) int {
	switch {
	case r >= 0x1100 && r <= 0x115F,
		r >= 0x2329 && r <= 0x232A,
		r >= 0x2600 && r <= 0x27BF,
		r >= 0x2E80 && r <= 0x303E,
		r >= 0x3041 && r <= 0x33FF,
		r >= 0x3400 && r <= 0x4DBF,
		r >= 0x4E00 && r <= 0xA4CF,
		r >= 0xA960 && r <= 0xA97C,
		r >= 0xAC00 && r <= 0xD7AF,
		r >= 0xF900 && r <= 0xFAFF,
		r >= 0xFE10 && r <= 0xFE19,
		r >= 0xFE30 && r <= 0xFE6F,
		r >= 0xFF01 && r <= 0xFF60,
		r >= 0xFFE0 && r <= 0xFFE6,
		r >= 0x1B000 && r <= 0x1B0FF,
		r >= 0x1B100 && r <= 0x1B12F,
		r >= 0x1F004 && r <= 0x1F004,
		r >= 0x1F0CF && r <= 0x1F0CF,
		r >= 0x1F18E && r <= 0x1F18E,
		r >= 0x1F191 && r <= 0x1F19A,
		r >= 0x1F200 && r <= 0x1F202,
		r >= 0x1F210 && r <= 0x1F23B,
		r >= 0x1F240 && r <= 0x1F248,
		r >= 0x1F250 && r <= 0x1F251,
		r >= 0x1F260 && r <= 0x1F265,
		r >= 0x1F300 && r <= 0x1F9FF,
		r >= 0x1FA00 && r <= 0x1FAFF,
		r >= 0x20000 && r <= 0x3FFFF:
		return 2
	}
	return 1
}

type tableData struct {
	rows      [][]string
	aligns    []int
	numCols   int
	colWidths []int
}

func buildTableData(tbl Node, cellFn func(Node) string) *tableData {
	if len(tbl.Children) == 0 {
		return nil
	}
	var rows [][]string
	var aligns []int
	if tbl.Alignments != nil {
		aligns = tbl.Alignments
	}
	for _, row := range tbl.Children {
		if row.Type != BlockTableRow {
			continue
		}
		var cells []string
		for _, cell := range row.Children {
			if cell.Type == BlockTableCell {
				cells = append(cells, cellFn(cell))
			}
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 {
		return nil
	}
	numCols := 0
	for _, r := range rows {
		if len(r) > numCols {
			numCols = len(r)
		}
	}
	if numCols == 0 {
		return nil
	}
	if len(aligns) < numCols {
		naligns := make([]int, numCols)
		copy(naligns, aligns)
		for i := len(aligns); i < numCols; i++ {
			naligns[i] = AlignNone
		}
		aligns = naligns
	}
	colWidths := make([]int, numCols)
	for _, r := range rows {
		for ci, cell := range r {
			w := visibleLen(cell)
			if w > colWidths[ci] {
				colWidths[ci] = w
			}
		}
	}
	return &tableData{
		rows:      rows,
		aligns:    aligns,
		numCols:   numCols,
		colWidths: colWidths,
	}
}

func padCell(s string, w int, a int) string {
	rw := visibleLen(s)
	switch a {
	case AlignRight:
		left := w - rw
		if left < 0 {
			left = 0
		}
		return strings.Repeat(" ", left) + s + " "
	case AlignCenter:
		left := (w - rw) / 2
		right := w - rw - left
		if left < 0 {
			left = 0
		}
		if right < 0 {
			right = 0
		}
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right) + " "
	default:
		right := w - rw
		if right < 0 {
			right = 0
		}
		return " " + s + strings.Repeat(" ", right) + " "
	}
}

func formatResult(v interface{}, format string) (string, error) {
	switch format {
	case "json":
		return formatJSON(v)
	case "text":
		return formatText(v), nil
	case "markdown", "plain":
		nodes, ok := v.([]Node)
		if !ok {
			return "", fmt.Errorf("expected nodes for markdown output")
		}
		return renderMarkdown(nodes, 0), nil
	case "terminal":
		nodes, ok := v.([]Node)
		if !ok {
			return "", fmt.Errorf("expected nodes for terminal output")
		}
		return renderTerminal(nodes, 0), nil
	default:
		return "", fmt.Errorf("unknown format: %s", format)
	}
}

func formatJSON(v interface{}) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}

func formatText(v interface{}) string {
	switch val := v.(type) {
	case []Node:
		return formatNodesText(val)
	case CountResult:
		return formatCountText(val)
	case Stats:
		return formatStatsText(val)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatNodesText(nodes []Node) string {
	var b strings.Builder
	for i, n := range nodes {
		if i > 0 {
			b.WriteString("\n")
		}
		switch n.Type {
		case BlockHeading:
			prefix := strings.Repeat("#", n.Level)
			fmt.Fprintf(&b, "%s %s", prefix, n.Content)
		case BlockCodeBlock:
			if n.Lang != "" {
				fmt.Fprintf(&b, "```%s\n", n.Lang)
			} else {
				b.WriteString("```\n")
			}
			b.WriteString(n.Content)
			if !strings.HasSuffix(n.Content, "\n") {
				b.WriteString("\n")
			}
			b.WriteString("```")
		case BlockLink:
			if n.Content != "" {
				fmt.Fprintf(&b, "[%s](%s)", n.Content, n.URL)
			} else {
				b.WriteString(n.URL)
			}
		case BlockImage:
			fmt.Fprintf(&b, "![%s](%s)", n.Content, n.URL)
		case BlockParagraph:
			b.WriteString(n.Content)
		case BlockBlockquote:
			b.WriteString("> " + n.Content)
		case BlockThematicBreak:
			b.WriteString("---")
		case BlockListItem:
			fmt.Fprintf(&b, "- %s", n.Content)
		case BlockTable:
			b.WriteString(renderPlainTable(n))
		case BlockList:
			for _, child := range n.Children {
				if child.Type == BlockListItem {
					prefix := "- "
					if n.Ordered {
						prefix = fmt.Sprintf("%d. ", child.Start)
					}
					fmt.Fprintf(&b, "%s%s\n", prefix, child.Content)
				}
			}
		case BlockText:
			b.WriteString(n.Content)
		default:
			b.WriteString(n.Content)
		}
	}
	return b.String()
}

func formatCountText(counts CountResult) string {
	type entry struct {
		key   string
		value int
	}
	var entries []entry
	for k, v := range counts {
		entries = append(entries, entry{k, v})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].value > entries[j].value
	})

	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "  %-20s %d\n", e.key+":", e.value)
	}
	return b.String()
}

func formatStatsText(s Stats) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  %-20s %d\n", "total_blocks:", s.TotalBlocks)
	fmt.Fprintf(&b, "  %-20s %d\n", "headings:", s.Headings)
	fmt.Fprintf(&b, "  %-20s %d\n", "paragraphs:", s.Paragraphs)
	fmt.Fprintf(&b, "  %-20s %d\n", "code_blocks:", s.CodeBlocks)
	fmt.Fprintf(&b, "  %-20s %d\n", "links:", s.Links)
	fmt.Fprintf(&b, "  %-20s %d\n", "images:", s.Images)
	fmt.Fprintf(&b, "  %-20s %d\n", "lists:", s.Lists)
	fmt.Fprintf(&b, "  %-20s %d\n", "blockquotes:", s.Blockquotes)
	fmt.Fprintf(&b, "  %-20s %d\n", "texts:", s.Texts)
	fmt.Fprintf(&b, "  %-20s %d\n", "thematic_breaks:", s.ThematicBreaks)
	return b.String()
}

func renderMarkdown(nodes []Node, depth int) string {
	var b strings.Builder
	for _, n := range nodes {
		switch n.Type {
		case BlockHeading:
			prefix := strings.Repeat("#", n.Level)
			fmt.Fprintf(&b, "\n%s %s\n\n", prefix, inlineOrContent(n))
		case BlockParagraph:
			if depth > 0 {
				b.WriteString(strings.Repeat("  ", depth))
			}
			b.WriteString(inlineOrContent(n))
			b.WriteString("\n\n")
		case BlockCodeBlock:
			if n.Lang != "" {
				fmt.Fprintf(&b, "```%s\n", n.Lang)
			} else {
				b.WriteString("```\n")
			}
			b.WriteString(n.Content)
			if !strings.HasSuffix(n.Content, "\n") {
				b.WriteString("\n")
			}
			b.WriteString("```\n\n")
		case BlockList:
			for i, child := range n.Children {
				if child.Type == BlockListItem {
					prefix := "- "
					if n.Ordered {
						prefix = fmt.Sprintf("%d. ", n.Start+i)
					}
					indent := strings.Repeat("  ", depth)
					itemContent := renderMarkdown(child.Children, depth+1)
					if strings.TrimSpace(itemContent) == "" {
						itemContent = child.Content
					}
					itemContent = strings.TrimRight(itemContent, "\n")
					b.WriteString(indent + prefix + itemContent + "\n")
				}
			}
			b.WriteString("\n")
		case BlockBlockquote:
			content := renderMarkdown(n.Children, 0)
			if strings.TrimSpace(content) == "" {
				content = n.Content
			}
			lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
			for _, line := range lines {
				b.WriteString("> " + line + "\n")
			}
			b.WriteString("\n")
		case BlockListItem:
			content := renderMarkdown(n.Children, depth)
			if strings.TrimSpace(content) == "" {
				content = n.Content
			}
			b.WriteString(content)
		case BlockTable:
			b.WriteString(renderPlainTable(n))
			b.WriteString("\n")
		case BlockThematicBreak:
			b.WriteString("---\n\n")
		case BlockLink:
			b.WriteString(renderLink(false, n))
		case BlockImage:
			fmt.Fprintf(&b, "![%s](%s)", n.Content, n.URL)
		case BlockEmphasis:
			b.WriteString("*" + inlineOrContent(n) + "*")
		case BlockStrong:
			b.WriteString("**" + inlineOrContent(n) + "**")
		case BlockCodeSpan:
			b.WriteString("`" + n.Content + "`")
		case BlockHTMLBlock:
			b.WriteString(n.Content + "\n\n")
		case BlockInlineHTML:
			b.WriteString(n.Content)
		case BlockText, "":
			b.WriteString(n.Content)
		}
	}
	return b.String()
}

func renderTerminal(nodes []Node, depth int) string {
	var b strings.Builder
	for _, n := range nodes {
		switch n.Type {
		case BlockHeading:
			headingColor := "\033[97m"
			switch n.Level {
			case 1:
				headingColor = "\033[93m"
			case 2:
				headingColor = "\033[96m"
			}
			b.WriteString("\n")
			b.WriteString(bold + headingColor)
			b.WriteString(renderInline(true, n.Children))
			b.WriteString(reset)
			b.WriteString("\n\n")
		case BlockParagraph:
			content := renderInline(true, n.Children)
			b.WriteString(content)
			b.WriteString("\n\n")
		case BlockCodeBlock:
			lines := strings.Split(strings.TrimRight(n.Content, "\n"), "\n")
			numw := len(strconv.Itoa(len(lines)))
			if numw < 2 {
				numw = 2
			}
			b.WriteString("\n")
			b.WriteString(fgGreen)
			if n.Lang != "" {
				b.WriteString(dim + fgBrBlack + "─╴" + n.Lang + reset + "\n")
			}
			for i, line := range lines {
				lineno := fmt.Sprintf("%*d", numw, i+1)
				b.WriteString(dim + fgBrBlack + lineno + " │" + reset + " ")
				b.WriteString(fgGreen + line + reset + "\n")
			}
			b.WriteString(dim + fgBrBlack + strings.Repeat("─", numw+2) + "┘" + reset)
			b.WriteString("\n\n")
		case BlockTable:
			b.WriteString(renderTerminalTable(n))
			b.WriteString("\n")
		case BlockList:
			for i, child := range n.Children {
				if child.Type == BlockListItem {
					prefix := bold + "- " + reset
					if n.Ordered {
						prefix = bold + fmt.Sprintf("%d. ", n.Start+i) + reset
					}
					indent := strings.Repeat("  ", depth)
					itemContent := renderTerminal(child.Children, depth+1)
					if strings.TrimSpace(itemContent) == "" {
						itemContent = child.Content
					}
					itemContent = strings.TrimRight(itemContent, "\n")
					b.WriteString(indent + prefix + itemContent + "\n")
				}
			}
			b.WriteString("\n")
		case BlockBlockquote:
			content := renderTerminal(n.Children, 0)
			if strings.TrimSpace(content) == "" {
				content = n.Content
			}
			lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
			for _, line := range lines {
				b.WriteString(dim)
				b.WriteString("│ " + reset)
				b.WriteString(line + "\n")
			}
			b.WriteString("\n")
		case BlockListItem:
			content := renderTerminal(n.Children, depth)
			if strings.TrimSpace(content) == "" {
				content = n.Content
			}
			b.WriteString(content)
		case BlockThematicBreak:
			b.WriteString(dim + strings.Repeat("─", 48) + reset + "\n\n")
		case BlockLink:
			b.WriteString(renderLink(true, n))
		case BlockImage:
			b.WriteString(dim + fgMagenta + "[" + n.Content + "](" + n.URL + ")" + reset)
		case BlockEmphasis:
			b.WriteString(italic + renderInline(true, n.Children) + reset)
		case BlockStrong:
			b.WriteString(bold + renderInline(true, n.Children) + reset)
		case BlockCodeSpan:
			b.WriteString(fgGreen + n.Content + reset)
		case BlockHTMLBlock:
			b.WriteString(n.Content + "\n\n")
		case BlockInlineHTML:
			b.WriteString(n.Content)
		case BlockText, "":
			b.WriteString(n.Content)
		}
	}
	return b.String()
}

func terminalWidth() int {
	w := os.Getenv("COLUMNS")
	if w != "" {
		if n, err := strconv.Atoi(w); err == nil && n > 10 {
			return n
		}
	}
	return 80
}

func renderTerminalTable(tbl Node) string {
	data := buildTableData(tbl, func(n Node) string {
		return renderInline(true, n.Children)
	})
	if data == nil {
		return ""
	}

	// save minimum column widths (content-based) to prevent
	// clamping from shrinking below what cells can display
	minWidths := make([]int, data.numCols)
	copy(minWidths, data.colWidths)

	total := data.numCols + 1
	for _, w := range data.colWidths {
		total += w + 2
	}
	maxW := terminalWidth() - 2
	if total > maxW {
		excess := total - maxW
		for excess > 0 {
			biggest := 0
			bi := 0
			for i, w := range data.colWidths {
				if w > minWidths[i] && w > biggest {
					biggest = w
					bi = i
				}
			}
			if biggest == 0 {
				break
			}
			shrink := data.colWidths[bi] - minWidths[bi]
			if shrink > excess {
				shrink = excess
			}
			data.colWidths[bi] -= shrink
			excess -= shrink
		}
	}

	hline := func(left, mid, right, horz string) string {
		var bld strings.Builder
		bld.WriteString(left)
		for ci := 0; ci < data.numCols; ci++ {
			w := data.colWidths[ci] + 2
			bld.WriteString(strings.Repeat(horz, w))
			if ci < data.numCols-1 {
				bld.WriteString(mid)
			}
		}
		bld.WriteString(right)
		return bld.String()
	}

	isHeader := true
	top := hline("┌", "┬", "┐", "─")
	sepLine := hline("├", "┼", "┤", "─")
	bot := hline("└", "┴", "┘", "─")

	var out strings.Builder
	out.WriteString("\n")
	out.WriteString(fgBrBlack + top + reset + "\n")

	for ri, row := range data.rows {
		out.WriteString(fgBrBlack + "│" + reset)
		for ci := 0; ci < data.numCols; ci++ {
			content := ""
			if ci < len(row) {
				content = row[ci]
			}
			a := AlignNone
			if ci < len(data.aligns) {
				a = data.aligns[ci]
			}
			if isHeader {
				out.WriteString(bold + padCell(content, data.colWidths[ci], a) + reset)
			} else {
				out.WriteString(padCell(content, data.colWidths[ci], a))
			}
			if ci < data.numCols-1 {
				out.WriteString(fgBrBlack + "│" + reset)
			}
		}
		out.WriteString(fgBrBlack + "│" + reset + "\n")
		if isHeader {
			out.WriteString(fgBrBlack + sepLine + reset + "\n")
			isHeader = false
		} else if ri < len(data.rows)-1 {
			out.WriteString(fgBrBlack + sepLine + reset + "\n")
		}
	}

	out.WriteString(fgBrBlack + bot + reset + "\n")
	return out.String()
}

func renderPlainTable(tbl Node) string {
	data := buildTableData(tbl, func(n Node) string {
		return renderInline(false, n.Children)
	})
	if data == nil {
		return ""
	}

	sep := func() string {
		var bld strings.Builder
		bld.WriteString("|")
		for ci := 0; ci < data.numCols; ci++ {
			bld.WriteString(strings.Repeat("-", data.colWidths[ci]+2))
			bld.WriteString("|")
		}
		return bld.String()
	}()

	var out strings.Builder
	for ri, row := range data.rows {
		out.WriteString("|")
		for ci := 0; ci < data.numCols; ci++ {
			content := ""
			if ci < len(row) {
				content = row[ci]
			}
			a := AlignNone
			if ci < len(data.aligns) {
				a = data.aligns[ci]
			}
			out.WriteString(padCell(content, data.colWidths[ci], a))
			out.WriteString("|")
		}
		out.WriteString("\n")
		if ri == 0 {
			out.WriteString(sep + "\n")
		}
	}
	return out.String()
}

func renderInline(term bool, nodes []Node) string {
	if len(nodes) == 0 {
		return ""
	}
	var b strings.Builder
	for _, n := range nodes {
		switch n.Type {
		case BlockText:
			b.WriteString(n.Content)
		case BlockCodeSpan:
			if term {
				b.WriteString(fgGreen + n.Content + reset)
			} else {
				b.WriteString("`" + n.Content + "`")
			}
		case BlockLink:
			b.WriteString(renderLink(term, n))
		case BlockImage:
			if term {
				b.WriteString(dim + fgMagenta + "[" + n.Content + "](" + n.URL + ")" + reset)
			} else {
				fmt.Fprintf(&b, "![%s](%s)", n.Content, n.URL)
			}
		case BlockEmphasis:
			if term {
				b.WriteString(italic + renderInline(true, n.Children) + reset)
			} else {
				b.WriteString("*" + inlineOrContent(n) + "*")
			}
		case BlockStrong:
			if term {
				b.WriteString(bold + renderInline(true, n.Children) + reset)
			} else {
				b.WriteString("**" + inlineOrContent(n) + "**")
			}
		case BlockInlineHTML:
			b.WriteString(n.Content)
		default:
			b.WriteString(n.Content)
		}
	}
	return b.String()
}

func renderLink(term bool, n Node) string {
	label := n.Content
	if len(n.Children) > 0 {
		label = renderInline(term, n.Children)
	}
	if term {
		var b strings.Builder
		b.WriteString(uline + fgBlue + label + reset)
		if n.URL != "" {
			b.WriteString(" ")
			b.WriteString(dim + fgBrBlack + "(" + n.URL + ")" + reset)
		}
		return b.String()
	}
	if label != "" {
		return fmt.Sprintf("[%s](%s)", label, n.URL)
	}
	return n.URL
}

func inlineOrContent(n Node) string {
	if len(n.Children) > 0 {
		return renderInline(false, n.Children)
	}
	return n.Content
}
