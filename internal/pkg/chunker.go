package pkg

import "strings"

type (
	Chunk struct {
		Content  string
		Index    int
		Metadata map[string]any
	}

	MarkdownChunker interface {
		Split(markdown string) []Chunk
		chunkSection(section string, start int) []Chunk
	}
	markdownChunker struct {
		MaxChars int
		Overlap  int
	}
)

func NewMarkdownChunker() MarkdownChunker {
	return &markdownChunker{
		MaxChars: 3000,
		Overlap:  400,
	}
}

func (c *markdownChunker) Split(markdown string) []Chunk {
	sections := splitSections(markdown)
	var chunks []Chunk
	index := 0
	for _, section := range sections {
		sectionChunks :=
			c.chunkSection(section, index)
		chunks = append(
			chunks,
			sectionChunks...,
		)
		index += len(sectionChunks)
	}
	return chunks
}

func (c *markdownChunker) chunkSection(section string, start int) []Chunk {
	paragraphs := cleanParagraphs(section)
	var chunks []Chunk
	var current strings.Builder
	index := start
	for _, paragraph := range paragraphs {
		if current.Len()+len(paragraph) > c.MaxChars {
			chunks = append(
				chunks,
				Chunk{
					Content: strings.TrimSpace(
						current.String(),
					),
					Index: index,
				},
			)
			index++
			// keep overlap from previous chunk
			overlap := lastNChars(
				current.String(),
				c.Overlap,
			)
			current.Reset()
			current.WriteString(
				overlap,
			)
			current.WriteString("\n\n")
		}
		current.WriteString(paragraph)
		current.WriteString("\n\n")
	}
	if current.Len() > 0 {
		chunks = append(
			chunks,
			Chunk{
				Content: strings.TrimSpace(
					current.String(),
				),
				Index: index,
			},
		)
	}
	return chunks
}

func splitSections(markdown string) []string {
	lines := strings.Split(markdown, "\n")
	var sections []string
	var current strings.Builder
	for _, line := range lines {
		if isSectionHeading(line) &&
			current.Len() > 0 {
			sections = append(
				sections,
				current.String(),
			)
			current.Reset()
		}
		current.WriteString(line)
		current.WriteString("\n")
	}
	if current.Len() > 0 {
		sections = append(
			sections,
			current.String(),
		)
	}
	return sections
}

func isSectionHeading(line string) bool {
	level := headingLevel(line)

	if level == 0 {
		return false
	}

	return level == 2
}

func lastNChars(text string, n int) string {
	if len(text) <= n {
		return text
	}
	return text[len(text)-n:]
}

func headingLevel(line string) int {
	line = strings.TrimSpace(line)
	count := 0
	for _, c := range line {
		if c == '#' {
			count++
		} else {
			break
		}
	}
	return count
}

func cleanParagraphs(section string) []string {
	lines := strings.Split(section, "\n")

	var paragraphs []string
	var current strings.Builder

	for _, line := range lines {

		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		current.WriteString(line)
		current.WriteString(" ")
	}

	if current.Len() > 0 {
		paragraphs = append(
			paragraphs,
			strings.TrimSpace(current.String()),
		)
	}

	return paragraphs
}
