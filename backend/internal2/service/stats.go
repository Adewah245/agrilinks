package service

import "sort"

// median returns the median of a slice of float64. Median (not mean) is used
// throughout this package because a handful of far-off submissions
// (typos, bad-faith entries) can skew a mean badly with small sample sizes,
// which is exactly the regime crowd-sourced price data lives in early on.
func median(values []float64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}
	sorted := make([]float64, n)
	copy(sorted, values)
	sort.Float64s(sorted)

	mid := n / 2
	if n%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

// medianAbsoluteDeviation (MAD) measures spread the same robust way median
// measures center. Used to build an outlier threshold that adapts to how
// volatile a commodity/market's recent prices actually are, instead of a
// fixed "+/-20%" rule that would be too strict for volatile crops and too
// loose for stable ones.
func medianAbsoluteDeviation(values []float64, med float64) float64 {
	if len(values) == 0 {
		return 0
	}
	deviations := make([]float64, len(values))
	for i, v := range values {
		d := v - med
		if d < 0 {
			d = -d
		}
		deviations[i] = d
	}
	return median(deviations)
}

func percentDeviation(value, reference float64) float64 {
	if reference == 0 {
		return 0
	}
	d := (value - reference) / reference
	if d < 0 {
		d = -d
	}
	return d
}
