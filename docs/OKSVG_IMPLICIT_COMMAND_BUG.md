# oksvg Bug Report: Implicit/Repeated SVG Path Commands Not Handled Correctly

## Summary

`oksvg` (github.com/srwiley/oksvg) does not correctly handle SVG paths that use implicit/repeated commands as defined in the SVG specification. When multiple path segments of the same type are chained without repeating the command letter, oksvg misinterprets the path data, resulting in incorrect rendering.

## SVG Path Specification Background

Per the [SVG Path specification](https://www.w3.org/TR/SVG/paths.html#PathDataBNF), command letters can be omitted for repeated commands of the same type:

> "The command letter can be eliminated on subsequent commands if the same command is used multiple times in a row (e.g., you can drop the second "L" in "M 0 0 L 50 50 L 100 100")."

For arc commands specifically:
```
M 0,0 A 50,50 0 0 1 100,0 50,50 0 0 1 200,0
```
Is equivalent to:
```
M 0,0 A 50,50 0 0 1 100,0 A 50,50 0 0 1 200,0
```

## The Bug

oksvg fails to parse the implicit form correctly. When arc commands are chained without repeating the `A`, oksvg appears to misinterpret subsequent arc parameters, resulting in drastically incorrect fills.

### Minimal Reproduction

```go
package main

import (
    "bytes"
    "fmt"
    "image"
    "image/png"
    "os"

    "github.com/srwiley/oksvg"
    "github.com/srwiley/rasterx"
)

func testPath(name, path string) {
    svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 231 231">` +
        `<path d="` + path + `" fill="#00BCD4"/></svg>`

    icon, _ := oksvg.ReadIconStream(bytes.NewReader([]byte(svg)))
    icon.SetTarget(0, 0, 256, 256)

    img := image.NewRGBA(image.Rect(0, 0, 256, 256))
    scanner := rasterx.NewScannerGV(256, 256, img, img.Bounds())
    raster := rasterx.NewDasher(256, 256, scanner)
    icon.Draw(raster, 1.0)

    // Count filled pixels
    count := 0
    for y := 0; y < 256; y++ {
        for x := 0; x < 256; x++ {
            _, _, _, a := img.At(x, y).RGBA()
            if a > 0 { count++ }
        }
    }
    fmt.Printf("%s: %.1f%% filled\n", name, float64(count)/65536*100)
}

func main() {
    // Implicit arc commands (SVG spec compliant, but broken in oksvg)
    implicit := "M115.5,51.75 A63.75,63.75 0 0 0 105,178.38 " +
                "A115.5,115.5 0 0 0 51.271,211.497 " +      // explicit A
                "115.5,115.5 0 0 0 179.731,211.497 " +      // implicit (no A)
                "115.5,115.5 0 0 0 126.002,192.468 Z"       // implicit (no A)

    // Explicit arc commands (workaround)
    explicit := "M115.5,51.75 A63.75,63.75 0 0 0 105,178.38 " +
                "A115.5,115.5 0 0 0 51.271,211.497 " +
                "A115.5,115.5 0 0 0 179.731,211.497 " +     // explicit A
                "A115.5,115.5 0 0 0 126.002,192.468 Z"      // explicit A

    testPath("implicit", implicit)
    testPath("explicit", explicit)
}
```

### Output

```
implicit: 86.5% filled   <- WRONG (should be ~20-30%)
explicit: 20.5% filled   <- CORRECT
```

The implicit form fills 86.5% of the canvas (essentially the entire viewBox), while the explicit form correctly renders a head/body shape filling only ~20% of the canvas.

## Affected Commands

This bug likely affects all path commands that support implicit repetition:
- `M`/`m` (moveto, with implicit lineto for subsequent points)
- `L`/`l` (lineto)
- `H`/`h` (horizontal lineto)
- `V`/`v` (vertical lineto)
- `C`/`c` (cubic bezier)
- `S`/`s` (smooth cubic bezier)
- `Q`/`q` (quadratic bezier)
- `T`/`t` (smooth quadratic bezier)
- `A`/`a` (arc) - **confirmed affected**

## Workaround

Pre-process SVG paths to ensure every segment has an explicit command letter. We implemented this in our path normalization code:

```go
// Instead of writing just a space between repeated commands:
// result.WriteByte(' ')

// Write the command letter explicitly:
result.WriteString(" A") // for arcs
result.WriteString(" L") // for lineto
result.WriteString(" C") // for cubic bezier
// etc.
```

## Visual Comparison

### Before Fix (Implicit Commands)
The head path renders as a massive blob covering ~86% of the canvas, completely obscuring the background circle.

### After Fix (Explicit Commands)
The head path renders correctly as a head/body shape covering ~20-30% of the canvas, with the background circle visible behind it.

## Testing Rig

We built a visual comparison tool to debug this issue, available at:

```
https://localhost:8443/avatar-debug
```

This page renders avatars side-by-side:
- **Left**: SVG rendered by the browser (correct reference)
- **Right**: PNG rendered by oksvg/rasterx

Features:
- Generate random avatars or view all 16 base designs
- View raw SVG and normalized SVG for each avatar
- Adjustable render size (64px - 512px)
- Immediate visual comparison to spot rendering differences

Source: [SongMartyn](https://github.com/i-build-web-apps/SongMartyn) - a self-hosted karaoke system using Multiavatar-style SVG avatars.

## Environment

- oksvg version: latest (tested January 2025)
- rasterx version: latest
- Go version: 1.21+

## Suggested Fix

In oksvg's path parser, after parsing a command's parameters, check if more parameters follow that match the same command type. If so, treat them as additional segments of the same command rather than attempting to parse them as a new command.

The relevant SVG spec section is [9.3.9 The grammar for path data](https://www.w3.org/TR/SVG/paths.html#PathDataBNF).

## References

- SVG Path Specification: https://www.w3.org/TR/SVG/paths.html
- oksvg Repository: https://github.com/srwiley/oksvg
- rasterx Repository: https://github.com/srwiley/rasterx
