// minitor is a tiny terminal CPU and network monitor for macOS.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

const maxHistory = 4096

// history holds one sample per tick: CPU busy fraction and network bytes/s.
type history struct {
	cpu, down, up []float64
}

func (h *history) push(cpu, down, up float64) {
	h.cpu = append(h.cpu, cpu)
	h.down = append(h.down, down)
	h.up = append(h.up, up)
	if len(h.cpu) > maxHistory {
		keep := maxHistory / 2
		h.cpu = append([]float64(nil), h.cpu[len(h.cpu)-keep:]...)
		h.down = append([]float64(nil), h.down[len(h.down)-keep:]...)
		h.up = append([]float64(nil), h.up[len(h.up)-keep:]...)
	}
}

type sampler struct {
	prevCPU [4]uint32
	haveCPU bool

	ifIndex         int
	resolved        time.Time
	prevIn, prevOut uint64
	prevNet         time.Time
	haveNet         bool
}

// sample returns the CPU busy fraction and network rates since the previous call.
// ok is false on the first call, which only records a baseline.
func (s *sampler) sample() (cpu, down, up float64, ok bool) {
	now := time.Now()

	if t, err := cpuTicks(); err == nil {
		if s.haveCPU {
			var d [4]float64
			for i := range t {
				d[i] = float64(t[i] - s.prevCPU[i]) // uint32 math survives counter wrap
			}
			busy := d[0] + d[1] + d[3]
			if total := busy + d[2]; total > 0 {
				cpu = busy / total
			}
		}
		ok = s.haveCPU
		s.prevCPU, s.haveCPU = t, true
	}

	if s.resolved.IsZero() || now.Sub(s.resolved) >= 5*time.Second {
		s.resolved = now
		if idx := defaultIfIndex(); idx != s.ifIndex {
			s.ifIndex, s.haveNet = idx, false
		}
	}
	if s.ifIndex == 0 {
		return cpu, 0, 0, ok
	}
	in, out, err := ifBytes(s.ifIndex)
	if err != nil {
		s.haveNet = false
		return cpu, 0, 0, ok
	}
	if s.haveNet && in >= s.prevIn && out >= s.prevOut {
		if dt := now.Sub(s.prevNet).Seconds(); dt > 0 {
			down = float64(in-s.prevIn) / dt
			up = float64(out-s.prevOut) / dt
		}
	}
	s.prevIn, s.prevOut, s.prevNet, s.haveNet = in, out, now, true
	return cpu, down, up, ok
}

func main() {
	interval := flag.Float64("n", 1, "sample interval in seconds")
	floorK := flag.Float64("floor", 64, "minimum network chart scale in KiB/s")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: minitor [-n seconds] [-floor KiB/s]\n\nq or Ctrl-C quits.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *interval < 0.05 || *floorK <= 0 {
		flag.Usage()
		os.Exit(2)
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Fprintln(os.Stderr, "minitor: stdout is not a terminal")
		os.Exit(1)
	}
	run(time.Duration(*interval*float64(time.Second)), *floorK*1024)
}

func run(interval time.Duration, floor float64) {
	st := colors
	if os.Getenv("NO_COLOR") != "" {
		st = style{}
	}

	in := int(os.Stdin.Fd())
	if old, err := term.MakeRaw(in); err == nil {
		defer term.Restore(in, old)
	}
	// Alt screen, hidden cursor, no autowrap (overlong lines clip instead of scrolling).
	os.Stdout.WriteString("\x1b[?1049h\x1b[?25l\x1b[?7l")
	defer os.Stdout.WriteString("\x1b[?7h\x1b[?25h\x1b[?1049l")

	quit := make(chan struct{}, 1)
	go func() {
		buf := make([]byte, 64)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			for _, b := range buf[:n] {
				if b == 'q' || b == 'Q' || b == 3 {
					quit <- struct{}{}
					return
				}
			}
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)

	var h history
	var s sampler
	s.sample()

	draw := func() {
		cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			cols, rows = 80, 4
		}
		lines := frame(&h, cols, rows, floor, st)
		// Clear each line before writing it: with autowrap off the cursor parks on the
		// last column, so a trailing erase would eat a full-width line's final char.
		out := "\x1b[H\x1b[2K" + strings.Join(lines, "\r\n\x1b[2K")
		if len(lines) < rows {
			out += fmt.Sprintf("\x1b[%dH\x1b[J", len(lines)+1)
		}
		os.Stdout.WriteString(out)
	}
	draw()

	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if cpu, down, up, ok := s.sample(); ok {
				h.push(cpu, down, up)
			}
			draw()
		case <-winch:
			draw()
		case <-quit:
			return
		case <-sigs:
			return
		}
	}
}
