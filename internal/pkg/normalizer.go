package pkg

import (
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

type (
	MarkDownNormalizer interface {
		Normalize(input string) string
		normalizeBlock(input string) string
		normalizeHeading(line string) string
		normalizeBullet(line string) string
		normalizeTable(line string) string
		normalizeSymbols(input string) string
	}
	markdownNormalizer struct {
		hyphenBreaks    *regexp.Regexp
		multipleBlank   *regexp.Regexp
		trailingSpaces  *regexp.Regexp
		multipleSpaces  *regexp.Regexp
		headingFix      *regexp.Regexp
		emptyLinks      *regexp.Regexp
		invisibleChars  *strings.Replacer
		isolatedSymbols *regexp.Regexp
	}
)

func NewMarkdownNormalizer() MarkDownNormalizer {
	return &markdownNormalizer{
		hyphenBreaks: regexp.MustCompile(
			`(\p{L})-\n(\p{L})`,
		),

		multipleBlank: regexp.MustCompile(
			`\n{3,}`,
		),

		trailingSpaces: regexp.MustCompile(
			`[ \t]+$`,
		),

		multipleSpaces: regexp.MustCompile(
			`[ \t]{2,}`,
		),

		headingFix: regexp.MustCompile(
			`^(#{1,6})([^ #])`,
		),

		emptyLinks: regexp.MustCompile(
			`\[\s*\]\([^)]*\)`,
		),

		invisibleChars: strings.NewReplacer(
			"\u200B", "", // zero-width space
			"\u200C", "", // zero-width non-joiner
			"\u200D", "", // zero-width joiner
			"\uFEFF", "", // BOM / zero-width no-break space
		),
		isolatedSymbols: regexp.MustCompile(
			`(?m)^[^\p{L}\p{N}#\-*_]+$`,
		),
	}
}

func (n *markdownNormalizer) Normalize(input string) string {
	// 1. Unicode normalization
	input = norm.NFKC.String(input)
	// 2. Remove invisible characters
	input = n.invisibleChars.Replace(input)
	// 3. Normalize line endings
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")
	// 4. Remove empty markdown links
	input = n.emptyLinks.ReplaceAllString(input, "")
	// 4.5 Remove isolated symbol
	input = n.isolatedSymbols.ReplaceAllString(
		input,
		"",
	)
	// 5. Normalize Symbol
	input = n.normalizeSymbols(input)
	// 6. Split markdown blocks
	blocks := splitMarkdownBlocks(input)
	for i, block := range blocks {
		// never modify code blocks
		if block.Type == BlockCode {
			continue
		}
		block.Content = n.normalizeBlock(block.Content)
		blocks[i] = block
	}
	// 7. Join blocks
	result := joinMarkdownBlocks(blocks)
	// 8. Final cleanup
	result = n.multipleBlank.ReplaceAllString(
		result,
		"\n\n",
	)

	return strings.TrimSpace(result)
}

func (n *markdownNormalizer) normalizeBlock(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	for _, line := range lines {
		// remove trailing spaces
		line = n.trailingSpaces.ReplaceAllString(line, "")
		// fix:
		// embed-
		// ding
		line = n.hyphenBreaks.ReplaceAllString(line, "$1$2")
		// heading normalization
		line = n.normalizeHeading(line)
		// bullet normalization
		line = n.normalizeBullet(line)
		// table normalization
		line = n.normalizeTable(line)
		// normal paragraph spacing
		if !isMarkdownSyntax(line) {
			line = n.multipleSpaces.ReplaceAllString(
				line,
				" ",
			)
		}
		result = append(result, line)
	}
	// merge broken PDF paragraphs
	result = mergeParagraphLines(result)
	return strings.Join(result, "\n")
}

func (n *markdownNormalizer) normalizeHeading(line string) string {
	return n.headingFix.ReplaceAllString(
		line,
		"$1 $2",
	)
}

func (n *markdownNormalizer) normalizeBullet(line string) string {
	trim := strings.TrimSpace(line)
	// PDF bullet character
	if strings.HasPrefix(trim, "•") {
		return "- " +
			strings.TrimSpace(
				strings.TrimPrefix(trim, "•"),
			)
	}
	// normalize:
	// -       item
	if strings.HasPrefix(trim, "-") {
		return "- " +
			strings.TrimSpace(
				strings.TrimPrefix(trim, "-"),
			)
	}
	return line
}

func (n *markdownNormalizer) normalizeTable(line string) string {
	if !strings.Contains(line, "|") {
		return line
	}
	parts := strings.Split(line, "|")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return "| " + strings.Join(parts, " | ") + " |"
}
func (n *markdownNormalizer) normalizeSymbols(input string) string {
	replacer := strings.NewReplacer(
		"¶", "",
		"⁋", "",
		"•", "-",
		"▪", "-",
		"■", "",
	)
	return replacer.Replace(input)
}

func mergeParagraphLines(lines []string) []string {
	var result []string
	for _, line := range lines {
		if len(result) == 0 {
			result = append(result, line)
			continue
		}
		prev := result[len(result)-1]
		// don't merge markdown structures
		if shouldMerge(prev, line) {
			result[len(result)-1] =
				prev + " " + strings.TrimSpace(line)
			continue
		}
		result = append(result, line)
	}
	return result
}

func shouldMerge(previous, current string) bool {
	if previous == "" ||
		current == "" {
		return false
	}
	if isMarkdownSyntax(previous) ||
		isMarkdownSyntax(current) {
		return false
	}
	return true
}

func isMarkdownSyntax(line string) bool {
	trim := strings.TrimSpace(line)
	if trim == "" {
		return true
	}
	prefixes := []string{"#", "-", "*", "+", ">", "|", "```", "~~~"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(trim, prefix) {
			return true
		}
	}
	return false
}

type MarkdownBlockType int

const (
	BlockText MarkdownBlockType = iota
	BlockCode
)

type MarkdownBlock struct {
	Type    MarkdownBlockType
	Content string
}

func splitMarkdownBlocks(input string) []MarkdownBlock {
	lines := strings.Split(input, "\n")
	var blocks []MarkdownBlock
	var current strings.Builder
	inCodeBlock := false
	codeFence := ""

	flush := func(blockType MarkdownBlockType) {
		if current.Len() == 0 {
			return
		}

		blocks = append(blocks, MarkdownBlock{
			Type:    blockType,
			Content: current.String(),
		})
		current.Reset()
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Start or end fenced code block
		if isCodeFence(trimmed) {
			if !inCodeBlock {
				// flush previous text
				flush(BlockText)
				inCodeBlock = true
				codeFence = trimmed[:3]
				current.WriteString(line)
				current.WriteString("\n")
				continue
			}

			// closing same fence
			if strings.HasPrefix(trimmed, codeFence) {
				current.WriteString(line)
				current.WriteString("\n")
				flush(BlockCode)
				inCodeBlock = false
				codeFence = ""
				continue
			}
		}
		current.WriteString(line)
		current.WriteString("\n")
	}

	// remaining content
	if current.Len() > 0 {
		if inCodeBlock {
			flush(BlockCode)
		} else {
			flush(BlockText)
		}
	}

	return blocks
}

func isCodeFence(line string) bool {
	return strings.HasPrefix(line, "```") ||
		strings.HasPrefix(line, "~~~")
}

func joinMarkdownBlocks(blocks []MarkdownBlock) string {
	var result []string
	for _, block := range blocks {
		result = append(
			result,
			block.Content,
		)
	}

	return strings.Join(result, "\n\n")
}
