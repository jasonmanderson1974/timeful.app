package routes

import (
	"strings"
	"testing"
)

func TestSanitizeCommentText(t *testing.T) {
	if _, ok := sanitizeCommentText("   "); ok {
		t.Error("whitespace-only comment should be rejected")
	}
	if _, ok := sanitizeCommentText(""); ok {
		t.Error("empty comment should be rejected")
	}
	if got, ok := sanitizeCommentText("  hi there  "); !ok || got != "hi there" {
		t.Errorf("trim: got %q ok=%v, want \"hi there\" true", got, ok)
	}
	long := strings.Repeat("a", maxCommentLength+50)
	got, ok := sanitizeCommentText(long)
	if !ok || len(got) != maxCommentLength {
		t.Errorf("over-long: len=%d ok=%v, want %d true", len(got), ok, maxCommentLength)
	}
}
