package cost

import (
	"fmt"
	"strconv"
	"strings"
)

// Bar represents a single bar in a chart.
type Bar struct {
	Label    string
	Value    int
	MaxValue int
}

// ChartOptions controls chart rendering.
type ChartOptions struct {
	Title      string
	Width      int    // Chart width in characters (default 60)
	Height     int    // Number of bars (0 = auto)
	ShowValues bool   // Show numeric values
	Vertical   bool   // Vertical bar chart (default horizontal)
	ScaleLabel string // Label for the value axis
}

// ASCIIBarChart generates an ASCII bar chart.
func ASCIIBarChart(bars []Bar, opts ChartOptions) string {
	var sb strings.Builder

	if opts.Title != "" {
		sb.WriteString(opts.Title + "\n")
	}

	// Find max value for scaling
	maxVal := 0
	for _, bar := range bars {
		if bar.Value > maxVal {
			maxVal = bar.Value
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	// Override with provided max value
	for i := range bars {
		if bars[i].MaxValue == 0 {
			bars[i].MaxValue = maxVal
		}
	}

	// Set defaults
	width := opts.Width
	if width == 0 {
		width = 60
	}

	// Horizontal bar chart
	if !opts.Vertical {
		// Print each bar
		for _, bar := range bars {
			// Calculate bar length
			barWidth := int(float64(bar.Value) / float64(bar.MaxValue) * float64(width))
			if barWidth > width {
				barWidth = width
			}

			// Create bar
			barStr := strings.Repeat("█", barWidth)
			emptyStr := strings.Repeat("░", width-barWidth)

			// Format value with commas
			valueStr := formatNumber(bar.Value)

			// Print bar
			label := bar.Label
			if len(label) > 15 {
				label = label[:12] + "..."
			}

			line := fmt.Sprintf("%-15s │%s%s│ %s", label, barStr, emptyStr, valueStr)
			sb.WriteString(line + "\n")
		}

		// Print scale label
		if opts.ScaleLabel != "" {
			sb.WriteString(fmt.Sprintf("Scale: 0%s%s\n", strings.Repeat("─", width-10), opts.ScaleLabel))
		}
	} else {
		// Vertical bar chart
		for _, bar := range bars {
			label := bar.Label
			if len(label) > 10 {
				label = label[:7] + "..."
			}

			// Calculate bar height
			barHeight := int(float64(bar.Value) / float64(bar.MaxValue) * 20)
			if barHeight > 20 {
				barHeight = 20
			}

			// Print value
			if opts.ShowValues {
				sb.WriteString(fmt.Sprintf(" %6s\n", formatNumber(bar.Value)))
			} else {
				sb.WriteString("      \n")
			}

			// Print bar (from bottom up)
			for i := 20; i > 0; i-- {
				if i <= barHeight {
					sb.WriteString("    ████ │\n")
				} else {
					sb.WriteString("        │\n")
				}
			}

			// Print label
			sb.WriteString(fmt.Sprintf("    %s\n", label))
			sb.WriteString("        └\n")
		}
	}

	return sb.String()
}

// ASCIILineChart generates an ASCII line chart showing trends over time.
func ASCIILineChart(dataPoints []struct {
	Label string
	Value int
}, opts ChartOptions,
) string {
	var sb strings.Builder

	if opts.Title != "" {
		sb.WriteString(opts.Title + "\n")
	}

	if len(dataPoints) == 0 {
		sb.WriteString("(no data)\n")

		return sb.String()
	}

	// Find max value for scaling
	maxVal := 0
	for _, dp := range dataPoints {
		if dp.Value > maxVal {
			maxVal = dp.Value
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	height := 15
	width := len(dataPoints) * 8
	if width > 80 {
		width = 80
	}

	// Create chart grid
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, width)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Draw axis lines
	for i := range height {
		grid[i][0] = '│'
	}
	for j := 1; j < width; j++ {
		grid[height-1][j] = '─'
	}
	grid[height-1][0] = '┴'

	// Plot data points
	for i, dp := range dataPoints {
		x := i*8 + 4
		if x >= width {
			break
		}

		// Calculate y position
		y := height - 1 - int(float64(dp.Value)/float64(maxVal)*float64(height-2))
		if y < 0 {
			y = 0
		}
		if y >= height {
			y = height - 1
		}

		// Plot point
		grid[y][x] = '●'

		// Connect to previous point
		if i > 0 {
			prevX := (i-1)*8 + 4
			prevY := height - 1 - int(float64(dataPoints[i-1].Value)/float64(maxVal)*float64(height-2))
			if prevY < 0 {
				prevY = 0
			}
			if prevY >= height {
				prevY = height - 1
			}

			// Draw line from prev to current
			drawLine(grid, prevX, prevY, x, y)
		}

		// Add x-axis label
		if i%2 == 0 && len(dp.Label) > 0 {
			label := dp.Label
			if len(label) > 3 {
				label = label[:3]
			}
			for j := range label {
				pos := x + j - 1
				if pos < width && pos >= 1 {
					grid[height-1][pos] = rune(label[0])
				}
			}
		}
	}

	// Add y-axis labels (0, 50%, 100%)
	for _, pct := range []int{0, 50, 100} {
		y := height - 1 - int(float64(pct)/100.0*float64(height-2))
		if y >= 0 && y < height {
			label := fmt.Sprintf("%d%%", pct)
			for j := range label {
				if j > 3 {
					break
				}
				pos := 1 - j
				if pos >= 0 {
					grid[y][pos] = rune(label[j])
				}
			}
		}
	}

	// Render grid
	for _, row := range grid {
		sb.WriteString(string(row) + "\n")
	}

	// Add legend
	sb.WriteString(fmt.Sprintf("Max: %s\n", formatNumber(maxVal)))

	return sb.String()
}

// drawLine draws a line from (x1,y1) to (x2,y2) using Bresenham's algorithm.
func drawLine(grid [][]rune, x1, y1, x2, y2 int) {
	dx := abs(x2 - x1)
	dy := -abs(y2 - y1)
	sx := -1
	if x1 < x2 {
		sx = 1
	}
	sy := -1
	if y1 < y2 {
		sy = 1
	}

	err := dx + dy
	if err == 0 {
		return
	}

	x, y := x1, y1
	maxIterations := dx + abs(dy) + 10 // Safety limit
	iterations := 0

	for {
		iterations++
		if iterations > maxIterations {
			return // Safety check to prevent infinite loop
		}

		// Plot point
		if x >= 0 && x < len(grid[0]) && y >= 0 && y < len(grid) {
			grid[y][x] = '·'
		}

		if x == x2 && y == y2 {
			break
		}

		e2 := 2 * err
		if e2 >= dy {
			err -= dy
			x += sx
		}
		if e2 <= dx {
			err += dx
			y += sy
		}
	}
}

// ASCIIPieChart generates an ASCII pie chart.
func ASCIIPieChart(slices []struct {
	Label   string
	Value   int
	Percent float64
}, opts ChartOptions,
) string {
	var sb strings.Builder

	if opts.Title != "" {
		sb.WriteString(opts.Title + "\n")
	}

	total := 0
	for _, slice := range slices {
		total += slice.Value
	}
	if total == 0 {
		total = 1
	}

	// Print legend with percentages
	for _, slice := range slices {
		pct := float64(slice.Value) / float64(total) * 100
		label := slice.Label
		if len(label) > 20 {
			label = label[:17] + "..."
		}

		// Create simple bar representation
		barWidth := int(pct / 2)
		barStr := strings.Repeat("█", barWidth)

		sb.WriteString(fmt.Sprintf("%-20s │%s%s│ %.1f%%\n",
			label, barStr, strings.Repeat("░", 50-barWidth), pct))
	}

	return sb.String()
}

// formatNumber formats a number with thousand separators.
func formatNumber(n int) string {
	str := strconv.Itoa(n)

	// Reverse the string
	runes := []rune(str)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	// Add commas every 3 digits
	var result []rune
	for i, ch := range runes {
		if i > 0 && i%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, ch)
	}

	// Reverse back
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}

	return x
}
