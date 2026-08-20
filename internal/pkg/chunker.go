package pkg

import (
	"log/slog"
	"strings"

	"github.com/tmc/langchaingo/textsplitter"
)

type (
	Chunk struct {
		Content  string
		Index    int
		Metadata map[string]any
	}

	MarkdownChunker interface {
		Split(markdown string) []Chunk
	}
	markdownChunker struct {
		splitter *textsplitter.MarkdownTextSplitter
		MaxChars int
		Overlap  int
	}
)

const (
	DefaultChunkSize   = 2000
	DefaultChunkOverlap = 200
)

func NewMarkdownChunker() MarkdownChunker {
	return &markdownChunker{
		splitter: textsplitter.NewMarkdownTextSplitter(
			textsplitter.WithChunkSize(DefaultChunkSize),
			textsplitter.WithChunkOverlap(DefaultChunkOverlap),
			textsplitter.WithCodeBlocks(true),
			textsplitter.WithJoinTableRows(true),
			textsplitter.WithHeadingHierarchy(true),
			textsplitter.WithKeepSeparator(true),
		),
		MaxChars: DefaultChunkSize,
		Overlap:  DefaultChunkOverlap,
	}
}

func (c *markdownChunker) Split(markdown string) []Chunk {
	if strings.TrimSpace(markdown) == "" {
		return nil
	}

	sections, err := c.splitter.SplitText(markdown)
	if err != nil {
		slog.Error("markdown split failed",
			"error", err,
		)
		return nil
	}

	chunks := make([]Chunk, 0, len(sections))
	for i, section := range sections {
		content := strings.TrimSpace(section)
		if content == "" {
			continue
		}
		chunks = append(chunks, Chunk{
			Content: content,
			Index:   i,
		})
	}
	return chunks
}
