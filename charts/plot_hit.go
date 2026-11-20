package charts

import "github.com/radix29/tuikit/core"

// PlotHit records where one chart's plot area landed and what it plotted, so
// a click on the screen maps back to a bucket and the values drawn there. A
// host's draw pass returns one per chart; Series is the chart's own slice, so
// don't hold it past the next draw.
type PlotHit struct {
	// Title is a label naming the chart, for the host to tell hits apart
	// (and caption a readout with).
	Title  string
	Plot   core.Rect
	Series []Series

	// TimeRow is the time-scale row directly under Plot, zero-sized on a chart
	// too short for one; where a host names the pinned column's moment.
	TimeRow core.Rect

	// Snapshot marks a chart of one current sample, not a history: each series
	// has one value describing one instant, so every column resolves to index
	// 0 (else it would answer only on the rightmost column).
	Snapshot bool
}

// Bucket is the index of the bucket drawn at screen column x, or -1 when x
// is outside the plot or over a column that predates the data.
func (h PlotHit) Bucket(x int) int {
	if h.Snapshot {
		if x < h.Plot.X || x >= h.Plot.Right() {
			return -1
		}
		return 0
	}
	return BucketAt(h.Plot, x, BucketCount(h.Series))
}

// Column is where bucket idx is drawn now, the inverse of Bucket, for a host
// holding a bucket across redraws. It returns -1 once newer samples push the
// bucket off the left edge. A snapshot chart always answers -1 (nothing drifts).
func (h PlotHit) Column(idx int) int {
	if h.Snapshot {
		return -1
	}
	return ColumnAt(h.Plot, idx, BucketCount(h.Series))
}

// Value is ser's value in bucket idx (from Bucket), read the way the chart
// drew it: a history aligns every series at its newest bucket
// (Series.AtBucket), so a readout reading At would disagree with the column
// under the pointer for a series shorter than the chart.
func (h PlotHit) Value(ser Series, idx int) float64 {
	if h.Snapshot {
		return ser.At(idx)
	}
	return ser.AtBucket(idx, BucketCount(h.Series))
}
