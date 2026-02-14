// Package diff provides unified diff generation.
package diff

import (
	"fmt"
	"strings"
)

// contextLines is the number of unchanged lines shown around each hunk.
const contextLines = 3

// Unified generates a unified diff between oldText and newText.
// Returns an empty string if the inputs are identical.
func Unified(filename, oldText, newText string) string {
	if oldText == newText {
		return ""
	}

	oldLines := splitLines(oldText)
	newLines := splitLines(newText)

	edits := myers(oldLines, newLines)
	hunks := buildHunks(edits)

	if len(hunks) == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n", filename)
	fmt.Fprintf(&b, "+++ b/%s\n", filename)

	for _, h := range hunks {
		h.writeTo(&b, oldLines, newLines)
	}

	return b.String()
}

// splitLines splits text into lines, preserving the trailing newline
// behavior. An empty string produces zero lines.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.SplitAfter(s, "\n")
	// SplitAfter leaves an empty trailing element when s ends with \n.
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// editKind represents a diff operation.
type editKind int

const (
	editEqual  editKind = iota
	editInsert          // line exists only in newText.
	editDelete          // line exists only in oldText.
)

// edit is a single diff operation.
type edit struct {
	kind   editKind
	oldIdx int // index in old (-1 for inserts).
	newIdx int // index in new (-1 for deletes).
}

// myers computes the shortest edit script using the Myers diff algorithm.
func myers(a, b []string) []edit {
	n := len(a)
	m := len(b)
	total := n + m
	if total == 0 {
		return nil
	}

	// v stores the farthest reaching path endpoints.
	// Indexed by k = x - y, offset by total to avoid negative indices.
	v := make([]int, 2*total+1)
	// trace stores a copy of v for each step d, used to reconstruct the path.
	trace := make([][]int, 0, total+1)

	for d := 0; d <= total; d++ {
		// Save v state for backtracking.
		vc := make([]int, len(v))
		copy(vc, v)
		trace = append(trace, vc)

		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[k-1+total] < v[k+1+total]) {
				x = v[k+1+total] // move down (insert).
			} else {
				x = v[k-1+total] + 1 // move right (delete).
			}
			y := x - k

			// Follow diagonal (equal lines).
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}

			v[k+total] = x

			if x >= n && y >= m {
				return backtrack(trace, a, b, d, total)
			}
		}
	}

	// Should not reach here for valid inputs.
	return nil
}

// backtrack reconstructs the edit script from the trace.
func backtrack(trace [][]int, a, b []string, d, total int) []edit {
	n := len(a)
	m := len(b)
	x, y := n, m

	var edits []edit

	for step := d; step > 0; step-- {
		v := trace[step]
		k := x - y

		var prevK int
		if k == -step || (k != step && v[k-1+total] < v[k+1+total]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}

		prevX := v[prevK+total]
		prevY := prevX - prevK

		// Diagonal (equal) lines.
		for x > prevX && y > prevY {
			x--
			y--
			edits = append(edits, edit{kind: editEqual, oldIdx: x, newIdx: y})
		}

		if k == -step || (k != step && v[k-1+total] < v[k+1+total]) {
			// Insert.
			y--
			edits = append(edits, edit{kind: editInsert, oldIdx: -1, newIdx: y})
		} else {
			// Delete.
			x--
			edits = append(edits, edit{kind: editDelete, oldIdx: x, newIdx: -1})
		}
	}

	// Remaining diagonal at d=0.
	for x > 0 && y > 0 {
		x--
		y--
		edits = append(edits, edit{kind: editEqual, oldIdx: x, newIdx: y})
	}

	// Reverse to get forward order.
	for i, j := 0, len(edits)-1; i < j; i, j = i+1, j-1 {
		edits[i], edits[j] = edits[j], edits[i]
	}

	return edits
}

// region represents a contiguous range of changed edits.
type region struct{ start, end int }

// hunk represents a unified diff hunk.
type hunk struct {
	oldStart int // 0-indexed start in old.
	oldCount int
	newStart int // 0-indexed start in new.
	newCount int
	edits    []edit
}

// buildHunks groups edits into hunks with context lines.
func buildHunks(edits []edit) []hunk {
	if len(edits) == 0 {
		return nil
	}

	regions := findChangeRegions(edits)
	merged := mergeRegions(regions)
	return regionsToHunks(merged, edits)
}

// findChangeRegions identifies contiguous ranges of non-equal edits.
func findChangeRegions(edits []edit) []region {
	var regions []region
	for i, e := range edits {
		if e.kind == editEqual {
			continue
		}
		if len(regions) == 0 || i > regions[len(regions)-1].end+1 {
			regions = append(regions, region{start: i, end: i})
		} else {
			regions[len(regions)-1].end = i
		}
	}
	return regions
}

// mergeRegions combines regions that are close enough that their contexts overlap.
func mergeRegions(regions []region) []region {
	var merged []region
	for _, r := range regions {
		if len(merged) > 0 && r.start-merged[len(merged)-1].end <= 2*contextLines {
			merged[len(merged)-1].end = r.end
			continue
		}
		merged = append(merged, r)
	}
	return merged
}

// regionsToHunks converts merged regions into hunks with context and line counts.
func regionsToHunks(regions []region, edits []edit) []hunk {
	hunks := make([]hunk, 0, len(regions))
	for _, r := range regions {
		start := max(r.start-contextLines, 0)
		end := min(r.end+contextLines, len(edits)-1)

		h := hunk{edits: edits[start : end+1]}
		h.oldStart, h.newStart = findHunkStarts(h.edits)
		h.oldCount, h.newCount = countHunkLines(h.edits)
		hunks = append(hunks, h)
	}
	return hunks
}

// findHunkStarts returns the first old and new line indices in the hunk.
func findHunkStarts(edits []edit) (oldStart, newStart int) {
	for _, e := range edits {
		if e.oldIdx >= 0 {
			oldStart = e.oldIdx
			break
		}
	}
	for _, e := range edits {
		if e.newIdx >= 0 {
			newStart = e.newIdx
			break
		}
	}
	return oldStart, newStart
}

// countHunkLines counts old and new lines in the hunk's edit list.
func countHunkLines(edits []edit) (oldCount, newCount int) {
	for _, e := range edits {
		switch e.kind {
		case editEqual:
			oldCount++
			newCount++
		case editDelete:
			oldCount++
		case editInsert:
			newCount++
		}
	}
	return oldCount, newCount
}

// writeTo writes the hunk in unified diff format.
func (h *hunk) writeTo(b *strings.Builder, oldLines, newLines []string) {
	fmt.Fprintf(b, "@@ -%d,%d +%d,%d @@\n",
		h.oldStart+1, h.oldCount,
		h.newStart+1, h.newCount)

	for _, e := range h.edits {
		switch e.kind {
		case editEqual:
			b.WriteByte(' ')
			b.WriteString(ensureNewline(oldLines[e.oldIdx]))
		case editDelete:
			b.WriteByte('-')
			b.WriteString(ensureNewline(oldLines[e.oldIdx]))
		case editInsert:
			b.WriteByte('+')
			b.WriteString(ensureNewline(newLines[e.newIdx]))
		}
	}
}

// ensureNewline makes sure the line ends with a newline for diff output.
func ensureNewline(line string) string {
	if strings.HasSuffix(line, "\n") {
		return line
	}
	return line + "\n"
}
