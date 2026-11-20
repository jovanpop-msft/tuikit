package charts

import (
	"testing"

	"github.com/radix29/tuikit/core"
)

func plotHitFixture() PlotHit {
	return PlotHit{
		Title: "Batches",
		Plot:  core.Rect{X: 5, Y: 2, W: 10, H: 4},
		Series: []Series{
			{Label: "long", Color: colA, Values: []float64{1, 2, 3, 4}},
			{Label: "short", Color: colB, Values: []float64{30, 40}},
		},
	}
}

// The rightmost column is the newest bucket; a column left of the data or
// outside the plot reports none.
func TestPlotHitBucketMapsColumnsToBuckets(t *testing.T) {
	h := plotHitFixture()
	if got := h.Bucket(h.Plot.Right() - 1); got != 3 {
		t.Errorf("rightmost column = bucket %d, want the newest (3)", got)
	}
	if got := h.Bucket(h.Plot.X); got != -1 {
		t.Errorf("a column predating the data = bucket %d, want -1", got)
	}
	if got := h.Bucket(h.Plot.Right()); got != -1 {
		t.Errorf("a column right of the plot = bucket %d, want -1", got)
	}
	if got := h.Column(3); got != h.Plot.Right()-1 {
		t.Errorf("Column(3) = %d, want the rightmost column %d", got, h.Plot.Right()-1)
	}
}

// Value reads a series the way the chart drew it: aligned at the newest
// bucket. Reading At instead quotes a shorter series' wrong sample.
func TestPlotHitValueAlignsAShorterSeriesAtItsNewestEnd(t *testing.T) {
	h := plotHitFixture()
	idx := h.Bucket(h.Plot.Right() - 1)
	if got := h.Value(h.Series[1], idx); got != 40 {
		t.Errorf("short series at the newest bucket = %v, want 40", got)
	}
	if got := h.Value(h.Series[1], 0); got != 0 {
		t.Errorf("short series before it started = %v, want 0", got)
	}
}

// A snapshot chart has one instant: every column inside the plot is bucket 0,
// and nothing drifts, so Column never answers.
func TestPlotHitSnapshotResolvesEveryColumnToTheOneSample(t *testing.T) {
	h := plotHitFixture()
	h.Snapshot = true
	h.Series = []Series{{Label: "now", Color: colA, Values: []float64{7}}}

	for _, x := range []int{h.Plot.X, h.Plot.Right() - 1} {
		if got := h.Bucket(x); got != 0 {
			t.Errorf("snapshot column %d = bucket %d, want 0", x, got)
		}
	}
	if got := h.Bucket(h.Plot.X - 1); got != -1 {
		t.Errorf("snapshot column outside the plot = bucket %d, want -1", got)
	}
	if got := h.Column(0); got != -1 {
		t.Errorf("snapshot Column(0) = %d, want -1", got)
	}
	if got := h.Value(h.Series[0], 0); got != 7 {
		t.Errorf("snapshot Value = %v, want 7", got)
	}
}
