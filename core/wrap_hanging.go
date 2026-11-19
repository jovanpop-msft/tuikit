package core

import "strings"

// WrapHanging wraps each line wider than w columns at a space, keeping the
// line's own spacing — WrapText collapses whitespace runs, which would scramble
// a two-column layout such as a key reference ("  Ctrl+X      Cut"). A
// continuation hangs under the second column: the column just past the first
// run of two or more spaces after the line's indent, else the indent itself,
// so a wrapped entry still reads as one entry under its key. A hang deeper than
// half of w falls back to the indent (capped at w/2), so the continuation keeps
// room for words.
//
// src[i] is the index in lines of the line rows[i] came from, so a caller can
// style every row of a wrapped line by its source (a heading whose
// continuation hangs indented is still a heading). A w of zero or less returns
// the lines unwrapped.
func WrapHanging(lines []string, w int) (rows []string, src []int) {
	rows = make([]string, 0, len(lines))
	src = make([]int, 0, len(lines))
	for i, line := range lines {
		if w <= 0 {
			rows, src = append(rows, line), append(src, i)
			continue
		}
		hang := hangColumn(line)
		if hang > w/2 {
			hang = min(leadingSpaces(line), w/2)
		}
		for DisplayWidth(line) > w {
			first, rest := splitHanging(line, w, hang)
			rows, src = append(rows, first), append(src, i)
			line = strings.Repeat(" ", hang) + rest
		}
		rows, src = append(rows, line), append(src, i)
	}
	return rows, src
}

// splitHanging cuts line into a head of at most w columns and the rest,
// breaking at the last space that fits beyond the hang (so the continuation
// always makes progress), or mid-word when no such space exists. The head is
// right-trimmed and the rest left-trimmed of spaces.
func splitHanging(line string, w, hang int) (head, rest string) {
	cut, col := -1, 0
	for i, r := range line {
		rw := DisplayWidth(string(r))
		if col+rw > w {
			if cut < 0 {
				cut = i
			}
			break
		}
		if r == ' ' && col > hang {
			cut = i
		}
		col += rw
	}
	if cut <= 0 {
		cut = len(line)
	}
	return strings.TrimRight(line[:cut], " "), strings.TrimLeft(line[cut:], " ")
}

// leadingSpaces is the number of leading spaces in line.
func leadingSpaces(line string) int { return len(line) - len(strings.TrimLeft(line, " ")) }

// hangColumn is the column a wrapped line continues at: just past the first run
// of two or more spaces after the indent (the key/description gap), or the
// indent when the line has no such gap.
func hangColumn(line string) int {
	ind := leadingSpaces(line)
	if gap := strings.Index(line[ind:], "  "); gap >= 0 {
		start := ind + gap
		return DisplayWidth(line[:start]) + leadingSpaces(line[start:])
	}
	return ind
}
