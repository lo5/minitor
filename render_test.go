package main

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestArea(t *testing.T) {
	g := newGrid(2, 1)
	g.area([]float64{1, 0.5, 0.25}, 3, -1, 4)
	// pad=1: col0 empty, col1 full(4), col2 two dots, col3 one dot (bottom rows).
	if got, want := g.row(0), string([]rune{0x2800 | 0x08 | 0x10 | 0x20 | 0x80, 0x2800 | 0x04 | 0x40 | 0x80}); got != want {
		t.Fatalf("up: got %q want %q", got, want)
	}
	g = newGrid(1, 1)
	g.area([]float64{0.5, 1}, 0, 1, 4)
	if got, want := g.row(0), string(rune(0x2800|0x01|0x02|0x08|0x10|0x20|0x80)); got != want {
		t.Fatalf("down: got %q want %q", got, want)
	}
	g = newGrid(1, 2)
	g.area([]float64{0.5, 1}, 7, -1, 8)
	if g.row(0) != string(rune(0x2800|0x08|0x10|0x20|0x80)) || g.row(1) != "⣿" {
		t.Fatalf("2-row: got %q %q", g.row(0), g.row(1))
	}
}

func TestRate(t *testing.T) {
	for in, want := range map[float64]string{
		0: "0B", 512: "512B", 999: "999B", 1000: "1.0K", 820 * 1024: "820K",
		12.4 * 1024 * 1024: "12.4M", 99.97 * 1024: "100K", 1023 * 1024 * 1024: "1.0G",
	} {
		if got := rate(in); got != want {
			t.Errorf("rate(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestFrameLayouts(t *testing.T) {
	var h history
	for i := 0; i < 100; i++ {
		h.push(float64(i%10)/10, float64(i*20000), float64(i*1000))
	}
	for rows, n := range map[int]int{1: 1, 2: 2, 3: 3, 4: 4, 40: 4} {
		lines := frame(&h, 50, rows, 64*1024, style{})
		if len(lines) != n {
			t.Fatalf("rows=%d: %d lines", rows, len(lines))
		}
		for _, l := range lines {
			if rows > 1 && utf8.RuneCountInString(l) != 50 {
				t.Errorf("rows=%d: width %d: %q", rows, utf8.RuneCountInString(l), l)
			}
		}
		t.Logf("rows=%d\n%s", rows, strings.Join(lines, "\n"))
	}
	if l := frame(&history{}, 50, 4, 1, style{}); len(l) != 4 {
		t.Fatal("empty history")
	}
}

func TestSampler(t *testing.T) {
	var s sampler
	if _, _, _, ok := s.sample(); ok {
		t.Fatal("first sample should be baseline")
	}
	if s.ifIndex == 0 {
		t.Fatal("no default route interface found")
	}
	time.Sleep(500 * time.Millisecond)
	cpu, down, up, ok := s.sample()
	if !ok || cpu <= 0 || cpu > 1 {
		t.Fatalf("cpu=%v ok=%v", cpu, ok)
	}
	t.Logf("if=%d cpu=%.1f%% down=%s/s up=%s/s", s.ifIndex, cpu*100, rate(down), rate(up))
}
