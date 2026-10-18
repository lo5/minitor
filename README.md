# minitor

A tiny CPU and network monitor for the macOS terminal: four lines of braille charts, nothing else.

```
C ⠀⠀⣠⣴⠀⠀⠀⣠⣴⠀⠀⠀⣠⣴⠀⠀⠀⣠⣴⠀⠀⠀⣠⣴⠀⠀      42%
  ⣤⣾⣿⣿⢀⣤⣾⣿⣿⢀⣤⣾⣿⣿⢀⣤⣾⣿⣿⢀⣤⣾⣿⣿
↓ ⣀⣀⣀⣀⣤⣤⣤⣤⣤⣤⣶⣶⣶⣶⣶⣶⣾⣿⣿⣿⣿⣿⣿⣿⣿  12.4M/s
↑ ⠉⠉⠉⠉⠛⠛⠛⠛⠛⠛⠿⠿⠿⠿⠿⠿⢿⣿⣿⣿⣿⣿⣿⣿⣿   820K/s
```

- **C**: total CPU across all cores.
- **↓ / ↑**: bytes/s on the default-route interface. Each direction scales to its own visible peak, never below `-floor` KiB/s.
- Fills the pane width. In panes shorter than 4 rows it shrinks: 3 rows, then 2, then a single text line.

```
go build -ldflags="-s -w" && ./minitor [-n seconds] [-floor KiB/s]
```

Quit with `q` or Ctrl-C. Set `NO_COLOR` to disable color.
