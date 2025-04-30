package hdr

import (
	"bytes"

	"github.com/HdrHistogram/hdrhistogram-go"
)

// HdrPlot creates a percentile distribution plot from a slice of int64 values.
// It returns the percentile distribution as a formatted string.
func HdrPlot(data []int64) (string, error) {
	// Create a histogram with a resolution of 1 microsecond
	// The maximum value can be set according to your needs, here it's set to 30 million microseconds (30 seconds)
	histogram := hdrhistogram.New(1, 30 * 1000000, 5)

	// Add the data to the histogram
	for _, value := range data {
		histogram.RecordValue(value)
	}

	// Create a buffer to capture the output of the PercentilesPrint function
	var buf bytes.Buffer

	// Calculate and print the percentiles to the buffer
	_, err := histogram.PercentilesPrint(&buf, 1, 1.0)
	if err != nil {
		return "", err
	}

	// Get the output as a string
	return buf.String(), nil
}
