package main

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

type BlockType string

const (
	BlockDocument      BlockType = "document"
	BlockHeading       BlockType = "heading"
	BlockParagraph     BlockType = "paragraph"
	BlockCodeBlock     BlockType = "code_block"
	BlockLink          BlockType = "link"
	BlockImage         BlockType = "image"
	BlockList          BlockType = "list"
	BlockListItem      BlockType = "list_item"
	BlockBlockquote    BlockType = "blockquote"
	BlockThematicBreak BlockType = "thematic_break"
	BlockText          BlockType = "text"
	BlockEmphasis      BlockType = "emphasis"
	BlockStrong        BlockType = "strong"
	BlockCodeSpan      BlockType = "code_span"
	BlockHTMLBlock     BlockType = "html_block"
	BlockInlineHTML    BlockType = "inline_html"
	BlockTable         BlockType = "table"
	BlockTableRow      BlockType = "table_row"
	BlockTableCell     BlockType = "table_cell"
)

const (
	AlignNone   = 0
	AlignLeft   = 1
	AlignRight  = 2
	AlignCenter = 3
)

type Node struct {
	Type       BlockType `json:"type"`
	Level      int       `json:"level,omitempty"`
	Content    string    `json:"content,omitempty"`
	Lang       string    `json:"lang,omitempty"`
	URL        string    `json:"url,omitempty"`
	Ordered    bool      `json:"ordered,omitempty"`
	Start      int       `json:"start,omitempty"`
	Depth      int       `json:"depth,omitempty"`
	Alignment  int       `json:"alignment,omitempty"`
	Alignments []int     `json:"alignments,omitempty"`
	Children   []Node    `json:"children,omitempty"`
}

type Document struct {
	Nodes []Node `json:"nodes"`
}

func parseMarkdown(source []byte) (*Document, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
		),
	)
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	root := Node{
		Type:     BlockDocument,
		Children: walkNodes(doc, source, 0),
	}
	return &Document{Nodes: root.Children}, nil
}

func walkNodes(n ast.Node, source []byte, depth int) []Node {
	var nodes []Node
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		node := convertNode(c, source, depth)
		if node != nil {
			nodes = append(nodes, *node)
		}
	}
	return nodes
}

func convertNode(n ast.Node, source []byte, depth int) *Node {
	switch v := n.(type) {
	case *ast.Heading:
		return &Node{
			Type:     BlockHeading,
			Level:    v.Level,
			Content:  extractText(v, source),
			Depth:    depth,
			Children: walkInlineChildren(v, source, depth),
		}
	case *ast.Paragraph:
		children := walkInlineChildren(v, source, depth)
		content := extractText(v, source)
		return &Node{
			Type:     BlockParagraph,
			Content:  content,
			Depth:    depth,
			Children: children,
		}
	case *ast.FencedCodeBlock:
		lang := string(v.Language(source))
		var buf bytes.Buffer
		lines := v.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(source))
		}
		return &Node{
			Type:    BlockCodeBlock,
			Lang:    lang,
			Content: buf.String(),
			Depth:   depth,
		}
	case *ast.CodeBlock:
		var buf bytes.Buffer
		lines := v.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(source))
		}
		return &Node{
			Type:    BlockCodeBlock,
			Content: buf.String(),
			Depth:   depth,
		}
	case *ast.List:
		ordered := v.IsOrdered()
		start := v.Start
		var children []Node
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if li, ok := c.(*ast.ListItem); ok {
				itemChildren := walkNodes(li, source, depth+1)
				itemContent := extractText(li, source)
				children = append(children, Node{
					Type:     BlockListItem,
					Content:  itemContent,
					Depth:    depth + 1,
					Children: itemChildren,
				})
			}
		}
		return &Node{
			Type:     BlockList,
			Ordered:  ordered,
			Start:    start,
			Depth:    depth,
			Children: children,
		}
	case *ast.Blockquote:
		children := walkNodes(v, source, depth+1)
		return &Node{
			Type:     BlockBlockquote,
			Depth:    depth,
			Children: children,
		}
	case *ast.ThematicBreak:
		return &Node{
			Type:  BlockThematicBreak,
			Depth: depth,
		}
	case *extast.Table:
		aligns := make([]int, len(v.Alignments))
		for i, a := range v.Alignments {
			aligns[i] = int(a)
		}
		var children []Node
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			switch c.(type) {
			case *extast.TableHeader:
				var cells []Node
				for cell := c.FirstChild(); cell != nil; cell = cell.NextSibling() {
					if tc, ok := cell.(*extast.TableCell); ok {
						cells = append(cells, Node{
							Type:      BlockTableCell,
							Alignment: int(tc.Alignment),
							Depth:     depth + 1,
							Children:  walkInlineChildren(tc, source, depth+2),
						})
					}
				}
				if len(cells) > 0 {
					children = append(children, Node{
						Type:     BlockTableRow,
						Depth:    depth + 1,
						Children: cells,
					})
				}
			case *extast.TableRow:
				tr := c.(*extast.TableRow)
				children = append(children, *convertTableRow(tr, source, depth+1))
			}
		}
		return &Node{
			Type:       BlockTable,
			Alignments: aligns,
			Depth:      depth,
			Children:   children,
		}
	case *ast.HTMLBlock:
		var buf bytes.Buffer
		lines := v.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(source))
		}
		return &Node{
			Type:    BlockHTMLBlock,
			Content: buf.String(),
			Depth:   depth,
		}
	}
	return nil
}

func convertTableRow(n *extast.TableRow, source []byte, depth int) *Node {
	var cells []Node
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if cell, ok := c.(*extast.TableCell); ok {
			cells = append(cells, Node{
				Type:      BlockTableCell,
				Alignment: int(cell.Alignment),
				Depth:     depth,
				Children:  walkInlineChildren(cell, source, depth+1),
			})
		}
	}
	return &Node{
		Type:     BlockTableRow,
		Depth:    depth,
		Children: cells,
	}
}

func walkInlineChildren(n ast.Node, source []byte, depth int) []Node {
	var nodes []Node
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		node := convertInlineNode(c, source, depth)
		if node != nil {
			nodes = append(nodes, *node)
		}
		if text, ok := c.(*ast.Text); ok {
			if text.HardLineBreak() {
				nodes = append(nodes, Node{
					Type:    BlockText,
					Content: "\n",
					Depth:   depth,
				})
			} else if text.SoftLineBreak() {
				nodes = append(nodes, Node{
					Type:    BlockText,
					Content: " ",
					Depth:   depth,
				})
			}
		}
	}
	return nodes
}

func convertInlineNode(n ast.Node, source []byte, depth int) *Node {
	switch v := n.(type) {
	case *ast.Link:
		children := walkInlineChildren(v, source, depth+1)
		return &Node{
			Type:     BlockLink,
			URL:      string(v.Destination),
			Content:  extractText(v, source),
			Depth:    depth,
			Children: children,
		}
	case *ast.Image:
		children := walkInlineChildren(v, source, depth+1)
		return &Node{
			Type:     BlockImage,
			URL:      string(v.Destination),
			Content:  extractText(v, source),
			Depth:    depth,
			Children: children,
		}
	case *ast.Emphasis:
		children := walkInlineChildren(v, source, depth+1)
		typ := BlockEmphasis
		if v.Level == 2 {
			typ = BlockStrong
		}
		return &Node{
			Type:     typ,
			Depth:    depth,
			Children: children,
		}
	case *ast.CodeSpan:
		return &Node{
			Type:    BlockCodeSpan,
			Content: extractText(v, source),
			Depth:   depth,
		}
	case *ast.Text:
		seg := v.Segment
		return &Node{
			Type:    BlockText,
			Content: string(seg.Value(source)),
			Depth:   depth,
		}
	case *ast.RawHTML:
		return &Node{
			Type:    BlockInlineHTML,
			Content: string(v.Segments.Value(source)),
			Depth:   depth,
		}
	}
	return nil
}

func extractText(n ast.Node, source []byte) string {
	var buf bytes.Buffer
	ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
	case *ast.Text:
		seg := node.Segment
		buf.Write(seg.Value(source))
		if node.HardLineBreak() {
			buf.WriteString("\n")
		} else if node.SoftLineBreak() {
			buf.WriteString(" ")
		}
	case *ast.String:
		buf.Write(node.Value)
	}
		return ast.WalkContinue, nil
	})
	return buf.String()
}
