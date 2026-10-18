package main

import (
	"fmt"
	"math"
	"strings"
)

// Braille dot bits indexed by [column][row], row 0 at the top of the cell.
var dotBits = [2][4]rune{{0x01, 0x02, 0x04, 0x40}, {0x08, 0x10, 0x20, 0x80}}

// grid is a braille canvas: each cell is 2 dots wide and 4 dots tall.
type grid struct {
	w, h  int // in cells
	cells []rune
}

func newGrid(w, h int) *grid {
	g := &grid{w: w, h: h, cells: make([]rune, w*h)}
	for i := range g.cells {
		g.cells[i] = 0x2800
	}
	return g
}

func (g *grid) set(x, y int) {
	g.cells[(y/4)*g.w+x/2] |= dotBits[x%2][y%4]
}

func (g *grid) row(r int) string {
	return string(g.cells[r*g.w : (r+1)*g.w])
}

// area plots vals (each in [0,1]) as filled columns starting at dot row y0 and
// growing in direction dir (-1 up, +1 down), with up to levels dots per column.
// The newest value lands in the rightmost dot column.
func (g *grid) area(vals []float64, y0, dir, levels int) {
	n := 2 * g.w
	if len(vals) > n {
		vals = vals[len(vals)-n:]
	}
	pad := n - len(vals)
	for i, v := range vals {
		k := int(math.Round(math.Max(0, math.Min(1, v)) * float64(levels)))
		for d := 0; d < k; d++ {
			g.set(pad+i, y0+dir*d)
		}
	}
}

// scaled divides the last n samples of vals by their peak (at least floor).
func scaled(vals []float64, n int, floor float64) []float64 {
	if len(vals) > n {
		vals = vals[len(vals)-n:]
	}
	peak := floor
	for _, v := range vals {
		peak = math.Max(peak, v)
	}
	out := make([]float64, len(vals))
	for i, v := range vals {
		out[i] = v / peak
	}
	return out
}

// rate formats bytes/s in base-1024 units, at most 5 characters (e.g. 12.4M, 820K).
func rate(v float64) string {
	const units = "BKMGT"
	i := 0
	for v >= 1000 && i < len(units)-1 {
		v /= 1024
		i++
	}
	if i > 0 && v < 99.95 {
		return fmt.Sprintf("%.1f%c", v, units[i])
	}
	return fmt.Sprintf("%.0f%c", v, units[i])
}

type style struct{ cpu, down, up string }

func (s style) paint(code, text string) string {
	if code == "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

var colors = style{cpu: "32", down: "36", up: "35"}

// frame renders the monitor into at most rows lines of cols columns.
func frame(h *history, cols, rows int, floor float64, st style) []string {
	cpuVal, downVal, upVal := "  -%", "-", "-"
	if n := len(h.cpu); n > 0 {
		cpuVal = fmt.Sprintf("%3.0f%%", h.cpu[n-1]*100)
		downVal, upVal = rate(h.down[n-1]), rate(h.up[n-1])
	}

	if rows <= 1 || cols < 16 {
		return []string{st.paint(st.cpu, "C") + " " + strings.TrimLeft(cpuVal, " ") +
			" " + st.paint(st.down, "↓"+downVal) + " " + st.paint(st.up, "↑"+upVal)}
	}

	valW := 7 // "12.4M/s"
	netVal := "↓" + downVal + " ↑" + upVal
	if rows == 2 {
		valW = 13
	}
	w := cols - 4 - valW // label, space, chart, two spaces, value
	if w < 1 {
		w = 1
	}
	line := func(label, code, chart, val string) string {
		return st.paint(code, label+" "+chart) + "  " + fmt.Sprintf("%*s", valW, val)
	}

	cpuRows := 1
	if rows >= 4 {
		cpuRows = 2
	}
	cg := newGrid(w, cpuRows)
	cg.area(h.cpu, 4*cpuRows-1, -1, 4*cpuRows)
	out := []string{line("C", st.cpu, cg.row(0), cpuVal)}
	if cpuRows == 2 {
		out = append(out, line(" ", st.cpu, cg.row(1), ""))
	}

	down := scaled(h.down, 2*w, floor)
	up := scaled(h.up, 2*w, floor)
	if rows == 2 {
		// Mirror within one row: download rises from the middle, upload hangs below it.
		ng := newGrid(w, 1)
		ng.area(down, 1, -1, 2)
		ng.area(up, 2, 1, 2)
		return append(out, line("N", st.down, ng.row(0), netVal))
	}
	dg, ug := newGrid(w, 1), newGrid(w, 1)
	dg.area(down, 3, -1, 4)
	ug.area(up, 0, 1, 4)
	return append(out,
		line("↓", st.down, dg.row(0), downVal+"/s"),
		line("↑", st.up, ug.row(0), upVal+"/s"))
}
