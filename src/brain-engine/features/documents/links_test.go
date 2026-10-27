package documents

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

const (
	docA = "00000000-0000-4000-8000-000000000001"
	docB = "00000000-0000-4000-8000-000000000002"
)

func TestParseLinksRecognizesCanonicalTargetsOnce(t *testing.T) {
	got, err := ParseLinks("[[Título]] [[doc:" + docA + "]] [[doc:" + docA + "]] [[doc:" + docB + "]] [[doc:not-a-uuid]]")
	if err != nil {
		t.Fatalf("ParseLinks() error = %v", err)
	}
	want := []string{docA, docB}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseLinks() = %#v, want %#v", got, want)
	}
}

func TestParseLinksSkipsFencedInlineAndEscapedCode(t *testing.T) {
	markdown := "before [[doc:" + docA + "]]\n" +
		"```md\n[[doc:" + docB + "]]\n```\n" +
		"inline `[[doc:" + docB + "]]`\n" +
		"escaped \\[[doc:" + docB + "]]\n" +
		"after [[doc:" + docB + "]]"
	got, err := ParseLinks(markdown)
	if err != nil {
		t.Fatalf("ParseLinks() error = %v", err)
	}
	want := []string{docA, docB}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseLinks() = %#v, want %#v", got, want)
	}
}

func TestParseLinksTreatsUnclosedBacktickAsLiteral(t *testing.T) {
	got, err := ParseLinks("unclosed ` marker [[doc:" + docA + "]]")
	if err != nil {
		t.Fatalf("ParseLinks() error = %v", err)
	}
	if !reflect.DeepEqual(got, []string{docA}) {
		t.Fatalf("ParseLinks() = %#v, want %q", got, docA)
	}
}

func TestParseLinksRejectsInvalidMarkdownBytes(t *testing.T) {
	for _, markdown := range []string{"bad\x00body", string([]byte{0xff})} {
		if _, err := ParseLinks(markdown); !errors.Is(err, ErrInvalidDocument) {
			t.Fatalf("ParseLinks(%q) error = %v, want ErrInvalidDocument", markdown, err)
		}
	}
}

func TestValidIDRequiresCanonicalLowercaseUUID(t *testing.T) {
	if !ValidID(docA) {
		t.Fatal("ValidID() rejected canonical UUID")
	}
	if ValidID("00000000-0000-4000-8000-00000000000A") {
		t.Fatal("ValidID() accepted uppercase UUID")
	}
	if ValidID("00000000000040008000000000000001") {
		t.Fatal("ValidID() accepted UUID without canonical separators")
	}
}

func TestParseLinksHandlesMalformedPrefixesAtDocumentLimit(t *testing.T) {
	const maxMarkdown = 1 << 20
	const malformed = "[[doc:xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	valid := "[[doc:" + docA + "]]"
	var markdown strings.Builder
	markdown.Grow(maxMarkdown)
	for markdown.Len()+len(malformed)+len(valid) <= maxMarkdown {
		markdown.WriteString(malformed)
	}
	markdown.WriteString(valid)

	got, err := ParseLinks(markdown.String())
	if err != nil {
		t.Fatalf("ParseLinks() error = %v", err)
	}
	if !reflect.DeepEqual(got, []string{docA}) {
		t.Fatalf("ParseLinks() = %#v, want only final valid target %s", got, docA)
	}
}
