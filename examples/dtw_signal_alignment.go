// Package main demonstrates a real‑world DTW scenario: detecting a short “pattern” in a longer
// sequence of cryptocurrency 1‑minute candlestick close prices using Dynamic Time Warping.
package main

import (
	"fmt"
	"math"

	"github.com/katalvlaran/lvlath/dtw"
)

// Candle represents a 1‑minute crypto candlestick.
type Candle struct {
	OpenTime   string  // timestamp of candle start (ISO 8601)
	CloseTime  string  // timestamp of candle close (ISO 8601)
	OpenPrice  float64 // opening price
	HighPrice  float64 // highest price
	LowPrice   float64 // lowest price
	ClosePrice float64 // closing price
}

// [Run on Go Playground](https://go.dev/play/p/uB9SH1Bzm86)
func main2() {
	// 1. Define the "reference" long time series (full history)
	refCandles := []Candle{
		{OpenTime: "2023-07-01 12:01:00.000000", CloseTime: "2023-07-01 12:01:59.999000", OpenPrice: 1916.31, ClosePrice: 1916.6, LowPrice: 1916.3, HighPrice: 1916.6},
		{OpenTime: "2023-07-01 12:02:00.000000", CloseTime: "2023-07-01 12:02:59.999000", OpenPrice: 1916.6, ClosePrice: 1916.8, LowPrice: 1916.23, HighPrice: 1916.23},
		{OpenTime: "2023-07-01 12:03:00.000000", CloseTime: "2023-07-01 12:03:59.999000", OpenPrice: 1916.23, ClosePrice: 1916.24, LowPrice: 1916.23, HighPrice: 1916.24},
		{OpenTime: "2023-07-01 12:04:00.000000", CloseTime: "2023-07-01 12:04:59.999000", OpenPrice: 1916.23, ClosePrice: 1916.33, LowPrice: 1916.23, HighPrice: 1916.24},
		{OpenTime: "2023-07-01 12:05:00.000000", CloseTime: "2023-07-01 12:05:59.999000", OpenPrice: 1916.24, ClosePrice: 1916.53, LowPrice: 1916.24, HighPrice: 1916.33},
		{OpenTime: "2023-07-01 12:06:00.000000", CloseTime: "2023-07-01 12:06:59.999000", OpenPrice: 1916.33, ClosePrice: 1916.34, LowPrice: 1916.29, HighPrice: 1916.3},
		{OpenTime: "2023-07-01 12:07:00.000000", CloseTime: "2023-07-01 12:07:59.999000", OpenPrice: 1916.3, ClosePrice: 1917.06, LowPrice: 1916.29, HighPrice: 1917.05},
		{OpenTime: "2023-07-01 12:08:00.000000", CloseTime: "2023-07-01 12:08:59.999000", OpenPrice: 1917.06, ClosePrice: 1917.2, LowPrice: 1916.81, HighPrice: 1916.81},
		{OpenTime: "2023-07-01 12:09:00.000000", CloseTime: "2023-07-01 12:09:59.999000", OpenPrice: 1916.82, ClosePrice: 1916.82, LowPrice: 1916.18, HighPrice: 1916.52},
		{OpenTime: "2023-07-01 12:10:00.000000", CloseTime: "2023-07-01 12:10:59.999000", OpenPrice: 1916.51, ClosePrice: 1917.63, LowPrice: 1916.51, HighPrice: 1917.55},
		{OpenTime: "2023-07-01 12:11:00.000000", CloseTime: "2023-07-01 12:11:59.999000", OpenPrice: 1917.54, ClosePrice: 1917.56, LowPrice: 1917, HighPrice: 1917.01},
		{OpenTime: "2023-07-01 12:12:00.000000", CloseTime: "2023-07-01 12:12:59.999000", OpenPrice: 1917, ClosePrice: 1917.39, LowPrice: 1916.87, HighPrice: 1917.38},
		{OpenTime: "2023-07-01 12:13:00.000000", CloseTime: "2023-07-01 12:13:59.999000", OpenPrice: 1917.39, ClosePrice: 1917.58, LowPrice: 1917.38, HighPrice: 1917.57},
		{OpenTime: "2023-07-01 12:14:00.000000", CloseTime: "2023-07-01 12:14:59.999000", OpenPrice: 1917.58, ClosePrice: 1918.4, LowPrice: 1917.57, HighPrice: 1918.2},
		{OpenTime: "2023-07-01 12:15:00.000000", CloseTime: "2023-07-01 12:15:59.999000", OpenPrice: 1918.2, ClosePrice: 1918.6, LowPrice: 1917.8, HighPrice: 1917.8},
		{OpenTime: "2023-07-01 12:16:00.000000", CloseTime: "2023-07-01 12:16:59.999000", OpenPrice: 1917.8, ClosePrice: 1917.81, LowPrice: 1916.38, HighPrice: 1916.38},
		{OpenTime: "2023-07-01 12:17:00.000000", CloseTime: "2023-07-01 12:17:59.999000", OpenPrice: 1916.38, ClosePrice: 1916.39, LowPrice: 1916.31, HighPrice: 1916.31},
		{OpenTime: "2023-07-01 12:18:00.000000", CloseTime: "2023-07-01 12:18:59.999000", OpenPrice: 1916.32, ClosePrice: 1916.32, LowPrice: 1916, HighPrice: 1916},
		{OpenTime: "2023-07-01 12:19:00.000000", CloseTime: "2023-07-01 12:19:59.999000", OpenPrice: 1916, ClosePrice: 1917, LowPrice: 1915.84, HighPrice: 1917},
		{OpenTime: "2023-07-01 12:20:00.000000", CloseTime: "2023-07-01 12:20:59.999000", OpenPrice: 1916.99, ClosePrice: 1918.2, LowPrice: 1916.99, HighPrice: 1918.2},
		{OpenTime: "2023-07-01 12:21:00.000000", CloseTime: "2023-07-01 12:21:59.999000", OpenPrice: 1918.19, ClosePrice: 1919.76, LowPrice: 1918.19, HighPrice: 1919.75},
		{OpenTime: "2023-07-01 12:22:00.000000", CloseTime: "2023-07-01 12:22:59.999000", OpenPrice: 1919.76, ClosePrice: 1921.62, LowPrice: 1919.61, HighPrice: 1921.2},
		{OpenTime: "2023-07-01 12:23:00.000000", CloseTime: "2023-07-01 12:23:59.999000", OpenPrice: 1921.21, ClosePrice: 1922.94, LowPrice: 1920.91, HighPrice: 1922.93},
	}

	// 2. Define "pattern" sequences to search for in the reference series
	pattern := []Candle{
		{OpenTime: "2023-07-01 12:04:00.000000", CloseTime: "2023-07-01 12:04:59.999000", OpenPrice: 1916.20, ClosePrice: 1916.30, LowPrice: 1916.20, HighPrice: 1916.35},
		{OpenTime: "2023-07-01 12:05:00.000000", CloseTime: "2023-07-01 12:05:59.999000", OpenPrice: 1916.25, ClosePrice: 1916.50, LowPrice: 1916.25, HighPrice: 1916.50},
		{OpenTime: "2023-07-01 12:06:00.000000", CloseTime: "2023-07-01 12:06:59.999000", OpenPrice: 1916.35, ClosePrice: 1916.35, LowPrice: 1916.3, HighPrice: 1916.4},
		{OpenTime: "2023-07-01 12:07:00.000000", CloseTime: "2023-07-01 12:07:59.999000", OpenPrice: 1916.35, ClosePrice: 1917.05, LowPrice: 1916.3, HighPrice: 1917.1},
	}

	// 3. Extract close-price series from Candle slices
	refSeries := extractCloses(refCandles)
	patSeries := extractCloses(pattern)

	// 4. Configure DTW options
	const (
		windowRadius = -1  // unlimited warping (free subsequence search)
		penalty      = 0.0 // no penalty for skips
	)
	opts := dtw.DefaultOptions()
	opts.Window = windowRadius
	opts.SlopePenalty = penalty
	opts.MemoryMode = dtw.FullMatrix
	opts.ReturnPath = true

	// 5. Slide pattern over reference and compute DTW distance for each possible alignment
	bestDist := math.Inf(1)
	var bestPath []dtw.Coord
	var bestIndex int
	refLen, patLen := len(refSeries), len(patSeries)
	for start := 0; start+patLen <= refLen; start++ {
		segment := refSeries[start : start+patLen]
		dist, path, err := dtw.DTW(segment, patSeries, &opts)
		if err != nil {
			fmt.Printf("error at start %d: %v\n", start, err)
			continue
		}
		// Track minimal distance
		if dist < bestDist {
			bestDist = dist
			bestPath = path
			bestIndex = start
		}
	}

	// 6. Print results
	fmt.Printf("Best DTW match starts at ref index %d (time %s)\n", bestIndex, refCandles[bestIndex].OpenTime)
	fmt.Printf("Minimal distance = %.2f\n", bestDist)
	fmt.Printf("Warping path (segmentIndex→patternIndex): %v\n", bestPath)

	// Expected Output:
	// Best DTW match starts at ref index 3 (time 2023-07-01 12:04:00.000000)
	// Minimal distance = 0.08
	// Warping path (segmentIndex→patternIndex): [{0 0} {1 1} {2 2} {3 3}]
}

// extractCloses converts a slice of Candle to a float64 slice of ClosePrice values.
func extractCloses(candles []Candle) []float64 {
	series := make([]float64, len(candles))
	for i, c := range candles {
		series[i] = c.ClosePrice
	}
	return series
}
