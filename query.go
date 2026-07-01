package main

import (
	"fmt"
	"strings"
)

func filterByType(nodes []Node, typeName string) []Node {
	blockType := typeNameToBlockType(typeName)
	if blockType == "" {
		return nil
	}
	var result []Node
	collectByType(nodes, blockType, &result)
	return result
}

func typeNameToBlockType(name string) BlockType {
	switch name {
	case "headings":
		return BlockHeading
	case "paragraphs":
		return BlockParagraph
	case "code_blocks", "code":
		return BlockCodeBlock
	case "links":
		return BlockLink
	case "images":
		return BlockImage
	case "lists":
		return BlockList
	case "list_items":
		return BlockListItem
	case "blockquotes":
		return BlockBlockquote
	case "text":
		return BlockText
	case "tables":
		return BlockTable
	case "table_rows":
		return BlockTableRow
	case "table_cells":
		return BlockTableCell
	case "html":
		return BlockHTMLBlock
	case "breaks", "thematic_breaks":
		return BlockThematicBreak
	}
	return ""
}

func collectByType(nodes []Node, blockType BlockType, result *[]Node) {
	for _, n := range nodes {
		if n.Type == blockType {
			*result = append(*result, n)
		}
		if len(n.Children) > 0 {
			collectByType(n.Children, blockType, result)
		}
	}
}

func filterByLevel(nodes []Node, level int) []Node {
	var result []Node
	for _, n := range nodes {
		if n.Type == BlockHeading && n.Level == level {
			result = append(result, n)
		}
		if len(n.Children) > 0 {
			result = append(result, filterByLevel(n.Children, level)...)
		}
	}
	return result
}

func filterByAttr(nodes []Node, expr string) []Node {
	blockType := ""
	attrKey := ""
	attrValue := ""

	if bkt := strings.IndexByte(expr, '['); bkt >= 0 {
		blockType = strings.TrimSpace(expr[:bkt])
		rest := strings.TrimSuffix(expr[bkt+1:], "]")
		if eq := strings.IndexByte(rest, '='); eq >= 0 {
			attrKey = strings.TrimSpace(rest[:eq])
			attrValue = strings.Trim(strings.TrimSpace(rest[eq+1:]), "\"'")
		}
	}

	if blockType == "" {
		return nil
	}

	targetType := typeNameToBlockType(blockType)
	if targetType == "" {
		if len(blockType) == 2 && blockType[0] == 'h' && blockType[1] >= '1' && blockType[1] <= '6' {
			return filterByLevel(nodes, int(blockType[1]-'0'))
		}
		return nil
	}

	candidates := filterByType(nodes, blockType)

	var result []Node
	for _, n := range candidates {
		switch attrKey {
		case "lang":
			if n.Lang == attrValue {
				result = append(result, n)
			}
		case "level":
			level := 0
			fmt.Sscanf(attrValue, "%d", &level)
			if n.Level == level {
				result = append(result, n)
			}
		case "url":
			if n.URL == attrValue {
				result = append(result, n)
			}
		}
	}
	return result
}

type CountResult map[string]int

func countByType(nodes []Node) CountResult {
	counts := make(CountResult)
	countRecursive(nodes, counts)
	return counts
}

func countRecursive(nodes []Node, counts map[string]int) {
	for _, n := range nodes {
		key := string(n.Type)
		counts[key]++
		if len(n.Children) > 0 {
			countRecursive(n.Children, counts)
		}
	}
}

type Stats struct {
	TotalBlocks int `json:"total_blocks"`
	Headings    int `json:"headings"`
	Paragraphs  int `json:"paragraphs"`
	CodeBlocks  int `json:"code_blocks"`
	Links       int `json:"links"`
	Images      int `json:"images"`
	Lists       int `json:"lists"`
	Blockquotes int `json:"blockquotes"`
	Texts       int `json:"texts"`
	ThematicBreaks int `json:"thematic_breaks"`
}

func calculateStats(nodes []Node) Stats {
	var s Stats
	s.TotalBlocks = countAll(nodes)
	s.Headings = len(filterByType(nodes, "headings"))
	s.Paragraphs = len(filterByType(nodes, "paragraphs"))
	s.CodeBlocks = len(filterByType(nodes, "code_blocks"))
	s.Links = len(filterByType(nodes, "links"))
	s.Images = len(filterByType(nodes, "images"))
	s.Lists = len(filterByType(nodes, "lists"))
	s.Blockquotes = len(filterByType(nodes, "blockquotes"))
	s.Texts = len(filterByType(nodes, "text"))
	s.ThematicBreaks = len(filterByType(nodes, "breaks"))
	return s
}

func countAll(nodes []Node) int {
	total := len(nodes)
	for _, n := range nodes {
		if len(n.Children) > 0 {
			total += countAll(n.Children)
		}
	}
	return total
}
