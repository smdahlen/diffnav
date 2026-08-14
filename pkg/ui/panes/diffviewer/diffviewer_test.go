package diffviewer

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderPreamble_Empty(t *testing.T) {
	if got := renderPreamble(""); got != "" {
		t.Fatalf("expected empty string for empty preamble, got %q", got)
	}
	if got := renderPreamble("   \n  \n  "); got != "" {
		t.Fatalf("expected empty string for whitespace-only preamble, got %q", got)
	}
}

func TestRenderPreamble_GitShow(t *testing.T) {
	preamble := `commit abc123def456
Author: Jane Doe <jane@example.com>
Date:   Mon Jan 1 00:00:00 2026 +0000

    feat: add new feature

    This is the body of the commit message.`

	got := renderPreamble(preamble)
	plain := ansi.Strip(got)

	// All original content lines should be preserved in the output.
	for _, want := range []string{
		"commit abc123def456",
		"Author: Jane Doe <jane@example.com>",
		"Date:   Mon Jan 1 00:00:00 2026 +0000",
		"feat: add new feature",
		"This is the body of the commit message.",
	} {
		if !strings.Contains(plain, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, plain)
		}
	}
}

func TestRenderPreamble_MergeCommit(t *testing.T) {
	preamble := `commit abc123def456
Merge: aaa111 bbb222
Author: Jane Doe <jane@example.com>
Date:   Mon Jan 1 00:00:00 2026 +0000

    Merge branch 'feature' into main`

	got := renderPreamble(preamble)
	plain := ansi.Strip(got)

	for _, want := range []string{
		"Merge: aaa111 bbb222",
		"Merge branch 'feature' into main",
	} {
		if !strings.Contains(plain, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, plain)
		}
	}
}

func TestWrapWidthsBreaksOnSpaces(t *testing.T) {
	// "alpha beta " fills 11 cells exactly, so "gamma" moves to the next row
	// whole rather than being split.
	got := wrapWidths("alpha beta gamma", 11)
	want := []int{11, 5}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestWrapWidthsHardBreaksOverlongWord(t *testing.T) {
	got := wrapWidths("supercalifragilistic", 8)
	for _, w := range got {
		if w > 8 {
			t.Fatalf("expected every run to fit in 8, got %v", got)
		}
	}
	total := 0
	for _, w := range got {
		total += w
	}
	if total != 20 {
		t.Fatalf("expected widths to sum to 20, got %d (%v)", total, got)
	}
}

func TestWrapDiffLineIndentsUnderGutter(t *testing.T) {
	gutter := "  1 \u22ee  1 \u2502"
	line := gutter + "alpha beta gamma delta"

	got := wrapDiffLine(line, len([]rune(gutter))+11, true)
	if len(got) < 2 {
		t.Fatalf("expected the line to wrap, got %d rows", len(got))
	}

	pad := strings.Repeat(" ", len([]rune(gutter)))
	var content strings.Builder
	for i, row := range got {
		text := row.item.ContentNoAnsi()
		if i == 0 {
			if !strings.HasPrefix(text, gutter) {
				t.Fatalf("expected the first row to keep the gutter, got %q", text)
			}
			content.WriteString(strings.TrimPrefix(text, gutter))
			continue
		}
		if !strings.HasPrefix(text, pad) {
			t.Fatalf("expected row %d to be indented under the gutter, got %q", i, text)
		}
		content.WriteString(strings.TrimPrefix(text, pad))
	}

	if content.String() != "alpha beta gamma delta" {
		t.Fatalf("expected the wrapped rows to reconstruct the line, got %q", content.String())
	}
}

func TestWrapDiffLineLeavesShortLinesAlone(t *testing.T) {
	line := "  1 \u22ee  1 \u2502short"
	if got := wrapDiffLine(line, 200, true); len(got) != 1 {
		t.Fatalf("expected a short line to stay on one row, got %d", len(got))
	}
	if got := wrapDiffLine(line, 4, false); len(got) != 1 {
		t.Fatalf("expected no wrapping when wrapText is false, got %d", len(got))
	}
}

func TestWrapDiffLineKeepsBackgroundFillOnEveryRow(t *testing.T) {
	// delta ends a +/- line with reset, background, erase-to-end-of-line so the
	// colour runs to the pane edge. Every wrapped row needs its own copy.
	fill := "\x1b[0m\x1b[48;2;57;69;69m\x1b[0K\x1b[0m"
	gutter := "  1 \u22ee  1 \u2502"
	line := gutter + "\x1b[48;2;57;69;69malpha beta gamma delta epsilon" + fill

	got := wrapDiffLine(line, len([]rune(gutter))+11, true)
	if len(got) < 2 {
		t.Fatalf("expected the line to wrap, got %d rows", len(got))
	}
	// item.NewItem drops the erase-to-end-of-line; what survives, and what
	// actually makes the renderer pad in colour, is the trailing background.
	want := "\x1b[48;2;57;69;69m\x1b[0m"
	for i, row := range got {
		if !strings.HasSuffix(row.item.Content(), want) {
			t.Fatalf(
				"expected row %d to end with the background re-set, got %q",
				i,
				row.item.Content(),
			)
		}
	}
}

func TestWrapDiffLineWithoutFillAddsNone(t *testing.T) {
	gutter := "  1 \u22ee  1 \u2502"
	got := wrapDiffLine(gutter+"alpha beta gamma delta", len([]rune(gutter))+11, true)
	for i, row := range got {
		if strings.Contains(row.item.Content(), "\x1b[0K") {
			t.Fatalf("expected no fill on a context line, row %d: %q", i, row.item.Content())
		}
	}
}
