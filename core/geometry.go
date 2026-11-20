package core

// ---------------------------------------------------------------------------
// Geometry
// ---------------------------------------------------------------------------

// Rect describes a rectangular region on the terminal screen.
type Rect struct {
	X, Y, W, H int
}

// Right returns the exclusive right edge (X + W).
func (r Rect) Right() int { return r.X + r.W }

// Bottom returns the exclusive bottom edge (Y + H).
func (r Rect) Bottom() int { return r.Y + r.H }

// Inner returns the rectangle inset by d on every side.
func (r Rect) Inner(d int) Rect {
	return Rect{r.X + d, r.Y + d, r.W - 2*d, r.H - 2*d}
}

// Contains reports whether (x,y) is inside the rectangle.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// IsZero reports whether the rect is empty.
func (r Rect) IsZero() bool { return r.W == 0 && r.H == 0 }

// SplitColumns divides r into side-by-side columns of equal width separated
// by gutter cells. It fits as many of the n requested as fit at no less than
// minW each and returns that many (fewer than n, or none, when r is too
// narrow; none when n <= 0 or r is empty). The last column absorbs the
// rounding remainder so the columns span r exactly.
//
// A host that wants "all or nothing" (one full-width panel rather than fewer
// columns) checks len(result) < n and substitutes []Rect{r}.
func SplitColumns(r Rect, n, gutter, minW int) []Rect {
	if n <= 0 || r.W <= 0 || r.H <= 0 {
		return nil
	}
	minW = max(minW, 1)
	gutter = max(gutter, 0)
	fit := min(n, (r.W+gutter)/(minW+gutter))
	if fit <= 0 {
		return nil
	}
	each := (r.W - (fit-1)*gutter) / fit
	out := make([]Rect, 0, fit)
	x := r.X
	for i := range fit {
		w := each
		if i == fit-1 {
			w = r.Right() - x
		}
		out = append(out, Rect{X: x, Y: r.Y, W: w, H: r.H})
		x += w + gutter
	}
	return out
}
