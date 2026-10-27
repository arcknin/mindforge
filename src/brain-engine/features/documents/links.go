package documents

import (
	"sort"
	"strings"
	"unicode/utf8"
)

// ValidID accepts the canonical lowercase, hyphenated UUID representation.
// Version and variant bits are intentionally not constrained: the document
// contract only requires a valid UUID structure and stable lowercase form.
func ValidID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i := 0; i < len(id); i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if id[i] != '-' {
				return false
			}
			continue
		}
		if !isLowerHex(id[i]) {
			return false
		}
	}
	return true
}

func isLowerHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
}

// ParseLinks extracts the supported canonical document-reference subset from
// Markdown. It deliberately does not resolve title wikilinks, interpret
// CommonMark in full, or rewrite the source body. Invalid UUID tokens remain
// ordinary text. Fenced and inline code, plus escaped tokens, are ignored.
func ParseLinks(markdown string) ([]string, error) {
	if !utf8.ValidString(markdown) || strings.ContainsRune(markdown, '\x00') {
		return nil, ErrInvalidDocument
	}
	targets := make([]string, 0)
	seen := make(map[string]struct{})
	backticks := indexBacktickRuns(markdown)
	for i := 0; i < len(markdown); {
		if marker, width, ok := fenceAt(markdown, i); ok {
			if end := skipFence(markdown, i, marker, width); end > i {
				i = end
				continue
			}
		}

		if markdown[i] == '`' && !isEscaped(markdown, i) {
			width := runLength(markdown, i, '`')
			if end := skipInlineCodeIndexed(backticks, i+width, width); end >= 0 {
				i = end
				continue
			}
		}

		const prefix = "[[doc:"
		if strings.HasPrefix(markdown[i:], prefix) && !isEscaped(markdown, i) {
			candidateStart := i + len(prefix)
			candidateEnd := candidateStart + 36
			if candidateEnd+2 <= len(markdown) && markdown[candidateEnd] == ']' && markdown[candidateEnd+1] == ']' {
				candidate := markdown[candidateStart:candidateEnd]
				if ValidID(candidate) {
					if _, exists := seen[candidate]; !exists {
						seen[candidate] = struct{}{}
						targets = append(targets, candidate)
					}
					i = candidateEnd + 2
					continue
				}
			}
		}
		i++
	}
	return targets, nil
}

func runLength(text string, start int, marker byte) int {
	width := 0
	for start+width < len(text) && text[start+width] == marker {
		width++
	}
	return width
}

func fenceAt(text string, start int) (byte, int, bool) {
	if start > 0 && text[start-1] != '\n' {
		return 0, 0, false
	}
	i := start
	spaces := 0
	for i < len(text) && text[i] == ' ' && spaces < 4 {
		i++
		spaces++
	}
	if spaces > 3 || i >= len(text) || (text[i] != '`' && text[i] != '~') {
		return 0, 0, false
	}
	marker := text[i]
	width := runLength(text, i, marker)
	if width < 3 {
		return 0, 0, false
	}
	return marker, width, true
}

func skipFence(text string, start int, marker byte, width int) int {
	lineEnd := strings.IndexByte(text[start:], '\n')
	if lineEnd < 0 {
		return len(text)
	}
	lineEnd += start
	for lineStart := lineEnd + 1; lineStart < len(text); {
		if closingFenceLine(text, lineStart, marker, width) {
			end := strings.IndexByte(text[lineStart:], '\n')
			if end < 0 {
				return len(text)
			}
			return lineStart + end + 1
		}
		next := strings.IndexByte(text[lineStart:], '\n')
		if next < 0 {
			return len(text)
		}
		lineStart += next + 1
	}
	return len(text)
}

func closingFenceLine(text string, start int, marker byte, width int) bool {
	i := start
	spaces := 0
	for i < len(text) && text[i] == ' ' && spaces < 4 {
		i++
		spaces++
	}
	if spaces > 3 || i >= len(text) || text[i] != marker {
		return false
	}
	if runLength(text, i, marker) < width {
		return false
	}
	runEnd := i + runLength(text, i, marker)
	lineEnd := strings.IndexByte(text[runEnd:], '\n')
	if lineEnd < 0 {
		lineEnd = len(text) - runEnd
	}
	return strings.TrimSpace(text[runEnd:runEnd+lineEnd]) == ""
}

func skipInlineCode(text string, start, width int) int {
	return skipInlineCodeIndexed(indexBacktickRuns(text), start, width)
}

func isEscaped(text string, index int) bool {
	backslashes := 0
	for i := index - 1; i >= 0 && text[i] == '\\'; i-- {
		backslashes++
	}
	return backslashes%2 == 1
}

const backtickMarker byte = 96

func indexBacktickRuns(text string) map[int][]int {
	index := make(map[int][]int)
	for i := 0; i < len(text); {
		if text[i] != backtickMarker || isEscaped(text, i) {
			i++
			continue
		}
		width := runLength(text, i, backtickMarker)
		index[width] = append(index[width], i)
		i += width
	}
	return index
}

func skipInlineCodeIndexed(index map[int][]int, start, width int) int {
	positions := index[width]
	next := sort.Search(len(positions), func(index int) bool {
		return positions[index] >= start
	})
	if next == len(positions) {
		return -1
	}
	return positions[next] + width
}
