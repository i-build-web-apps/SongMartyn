package avatar

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"image"
	"math/big"
	"regexp"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// Colors represents custom color overrides for avatar parts
type Colors struct {
	Env   string `json:"env,omitempty"`
	Clo   string `json:"clo,omitempty"`
	Head  string `json:"head,omitempty"`
	Mouth string `json:"mouth,omitempty"`
	Eyes  string `json:"eyes,omitempty"`
	Top   string `json:"top,omitempty"`
}

// Color palette for random selection - matches frontend ColorPicker
var ColorPalette = []string{
	// Skin tones
	"#FFDFC4", "#F0C08A", "#D2956B", "#A67449", "#6B4423",
	// Basic colors
	"#FF4444", "#FF9500", "#FFCC00", "#4CAF50", "#2196F3", "#9C27B0", "#E91E8C",
	// Neutrals
	"#FFFFFF", "#9E9E9E", "#424242", "#000000",
	// Accents
	"#00BCD4", "#FF6B9D", "#7C4DFF", "#00E676",
}

// Skin tone colors for head part
var SkinTones = []string{
	"#FFDFC4", "#F0C08A", "#D2956B", "#A67449", "#6B4423",
}

// Config represents the avatar customization options
// Each field is 0-47, where:
//   - value % 16 = part design (0-15)
//   - value / 16 = color variant (0=A, 1=B, 2=C)
type Config struct {
	Env    int     `json:"env"`
	Clo    int     `json:"clo"`
	Head   int     `json:"head"`
	Mouth  int     `json:"mouth"`
	Eyes   int     `json:"eyes"`
	Top    int     `json:"top"`
	Colors *Colors `json:"colors,omitempty"`
}

// MaxPartValue is the maximum value for each part (48 combinations: 16 designs × 3 colors)
const MaxPartValue = 48

// NumDesigns is the number of different designs per part
const NumDesigns = 16

// NewRandom creates a new random avatar configuration
func NewRandom() Config {
	return Config{
		Env:   randomInt(MaxPartValue),
		Clo:   randomInt(MaxPartValue),
		Head:  randomInt(MaxPartValue),
		Mouth: randomInt(MaxPartValue),
		Eyes:  randomInt(MaxPartValue),
		Top:   randomInt(MaxPartValue),
	}
}

// NewRandomWithColors creates a new random avatar with random colors from the palette
func NewRandomWithColors() Config {
	return Config{
		Env:   randomInt(MaxPartValue),
		Clo:   randomInt(MaxPartValue),
		Head:  randomInt(MaxPartValue),
		Mouth: randomInt(MaxPartValue),
		Eyes:  randomInt(MaxPartValue),
		Top:   randomInt(MaxPartValue),
		Colors: &Colors{
			Env:   randomColor(),
			Clo:   randomColor(),
			Head:  randomSkinTone(),
			Mouth: randomColor(),
			Eyes:  randomColor(),
			Top:   randomColor(),
		},
	}
}

// randomColor returns a random color from the palette
func randomColor() string {
	return ColorPalette[randomInt(len(ColorPalette))]
}

// randomSkinTone returns a random skin tone color
func randomSkinTone() string {
	return SkinTones[randomInt(len(SkinTones))]
}

// randomInt returns a random int in [0, max)
func randomInt(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

// ToJSON serializes the config to JSON
func (c Config) ToJSON() string {
	data, _ := json.Marshal(c)
	return string(data)
}

// FromJSON deserializes a config from JSON
func FromJSON(data string) (Config, error) {
	var c Config
	err := json.Unmarshal([]byte(data), &c)
	return c, err
}

// getDesignAndVariant extracts the design index and color variant from a part value
func getDesignAndVariant(value int) (designIdx string, variant string) {
	// Normalize to valid range
	if value < 0 {
		value = 0
	}
	if value >= MaxPartValue {
		value = MaxPartValue - 1
	}

	// Design is 0-15 (16 different designs)
	design := value % NumDesigns
	designIdx = fmt.Sprintf("%02d", design)

	// Variant is 0, 1, or 2 (A, B, or C)
	variantIdx := value / NumDesigns
	switch variantIdx {
	case 0:
		variant = "A"
	case 1:
		variant = "B"
	default:
		variant = "C"
	}

	return designIdx, variant
}

// getColors returns the colors for a specific part, design, and variant
func getColors(part, designIdx, variant string) []string {
	theme, ok := Themes[designIdx]
	if !ok {
		return nil
	}

	var colors PartColors
	switch variant {
	case "A":
		colors = theme.A
	case "B":
		colors = theme.B
	case "C":
		colors = theme.C
	default:
		colors = theme.A
	}

	switch part {
	case "env":
		return colors.Env
	case "clo":
		return colors.Clo
	case "head":
		return colors.Head
	case "mouth":
		return colors.Mouth
	case "eyes":
		return colors.Eyes
	case "top":
		return colors.Top
	}
	return nil
}

// getSVG returns the SVG template for a specific part and design
func getSVG(part, designIdx string) string {
	parts, ok := Parts[designIdx]
	if !ok {
		return ""
	}

	switch part {
	case "env":
		return parts.Env
	case "clo":
		return parts.Clo
	case "head":
		return parts.Head
	case "mouth":
		return parts.Mouth
	case "eyes":
		return parts.Eyes
	case "top":
		return parts.Top
	}
	return ""
}

// applyColors replaces color placeholders in SVG with actual colors
// Placeholders are in format: fill:#01; or fill:#001; etc.
// If customColor is provided (non-empty), it overrides the first/primary color
func applyColors(svg string, colors []string, customColor string) string {
	if len(colors) == 0 {
		return svg
	}

	result := svg

	// If custom color is provided, replace the primary fill color
	if customColor != "" {
		// Strategy 1: Replace the theme's first color if it appears in the SVG
		if len(colors) > 0 {
			result = strings.ReplaceAll(result, colors[0]+";", customColor+";")
		}

		// Strategy 2: Replace the first fill color in the SVG using regex
		// This handles parts where the SVG has hardcoded colors not matching theme
		fillPattern := regexp.MustCompile(`fill:(#[0-9a-fA-F]{3,6});`)
		replaced := false
		result = fillPattern.ReplaceAllStringFunc(result, func(match string) string {
			if !replaced {
				replaced = true
				return "fill:" + customColor + ";"
			}
			return match
		})
	}

	// Replace placeholders like #01, #001, #0001 etc. with colors
	// The multiavatar format uses #XX; where XX is the color index (01-based)
	for i, color := range colors {
		colorToUse := color
		if i == 0 && customColor != "" {
			colorToUse = customColor
		}

		// Try different placeholder formats
		placeholder := fmt.Sprintf("#%02d;", i+1)
		result = strings.ReplaceAll(result, placeholder, colorToUse+";")

		// Also handle single digit format
		placeholder = fmt.Sprintf("#%d;", i+1)
		result = strings.ReplaceAll(result, placeholder, colorToUse+";")
	}

	return result
}

// ToSVG generates the complete SVG for this avatar configuration
func (c Config) ToSVG() string {
	return c.ToSVGWithEnv(true)
}

// getCustomColor returns the custom color for a part, if set
func (c Config) getCustomColor(part string) string {
	if c.Colors == nil {
		return ""
	}
	switch part {
	case "env":
		return c.Colors.Env
	case "clo":
		return c.Colors.Clo
	case "head":
		return c.Colors.Head
	case "mouth":
		return c.Colors.Mouth
	case "eyes":
		return c.Colors.Eyes
	case "top":
		return c.Colors.Top
	}
	return ""
}

// ToSVGWithEnv generates the SVG with optional environment/background
func (c Config) ToSVGWithEnv(includeEnv bool) string {
	var parts []string

	// Build each part with its colors applied
	partConfigs := []struct {
		name  string
		value int
	}{
		{"env", c.Env},
		{"head", c.Head},
		{"clo", c.Clo},
		{"mouth", c.Mouth},
		{"eyes", c.Eyes},
		{"top", c.Top},
	}

	for _, pc := range partConfigs {
		if pc.name == "env" && !includeEnv {
			continue
		}

		designIdx, variant := getDesignAndVariant(pc.value)
		svgTemplate := getSVG(pc.name, designIdx)
		colors := getColors(pc.name, designIdx, variant)
		customColor := c.getCustomColor(pc.name)

		// Apply colors to the SVG template (with optional custom color override)
		partSVG := applyColors(svgTemplate, colors, customColor)
		parts = append(parts, partSVG)
	}

	return SvgStart + strings.Join(parts, "") + SvgEnd
}

// Normalize ensures all values are within valid range
func (c *Config) Normalize() {
	normalize := func(v int) int {
		if v < 0 {
			return 0
		}
		if v >= MaxPartValue {
			return MaxPartValue - 1
		}
		return v
	}

	c.Env = normalize(c.Env)
	c.Clo = normalize(c.Clo)
	c.Head = normalize(c.Head)
	c.Mouth = normalize(c.Mouth)
	c.Eyes = normalize(c.Eyes)
	c.Top = normalize(c.Top)
}

// IncrementPart increments a specific part by delta (wrapping around)
func (c *Config) IncrementPart(part string, delta int) {
	increment := func(v, d int) int {
		v = (v + d) % MaxPartValue
		if v < 0 {
			v += MaxPartValue
		}
		return v
	}

	switch part {
	case "env":
		c.Env = increment(c.Env, delta)
	case "clo":
		c.Clo = increment(c.Clo, delta)
	case "head":
		c.Head = increment(c.Head, delta)
	case "mouth":
		c.Mouth = increment(c.Mouth, delta)
	case "eyes":
		c.Eyes = increment(c.Eyes, delta)
	case "top":
		c.Top = increment(c.Top, delta)
	}
}

// colorPlaceholderRegex matches color placeholders like #01; or #001;
var colorPlaceholderRegex = regexp.MustCompile(`#0*(\d+);`)

// Regex patterns for SVG normalization
var (
	// Match style attribute: style="..."
	styleAttrRegex = regexp.MustCompile(`\s+style="([^"]*)"`)
	// Match individual CSS properties within style
	cssPropRegex = regexp.MustCompile(`([a-z-]+)\s*:\s*([^;]+);?`)
	// Match px units that need to be stripped for attributes
	pxUnitRegex = regexp.MustCompile(`^([\d.]+)px$`)
	// Match path d attribute
	pathDAttrRegex = regexp.MustCompile(`(<path[^>]*\sd=")([^"]+)("[^>]*>)`)
)

// normalizePathCommands converts relative path commands to absolute for oksvg compatibility
// oksvg has issues with relative arc commands ('a'), so we convert them to absolute ('A')
func normalizePathCommands(svg string) string {
	return pathDAttrRegex.ReplaceAllStringFunc(svg, func(match string) string {
		parts := pathDAttrRegex.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}
		prefix := parts[1]  // <path ... d="
		pathData := parts[2] // the path data
		suffix := parts[3]   // " ...>

		// Convert relative commands to absolute
		normalizedPath := convertRelativeToAbsolute(pathData)
		return prefix + normalizedPath + suffix
	})
}

// convertRelativeToAbsolute converts relative SVG path commands to absolute
// This specifically fixes the 'a' (relative arc) command which oksvg handles incorrectly
func convertRelativeToAbsolute(pathData string) string {
	var result strings.Builder
	var curX, curY float64
	var startX, startY float64 // For Z command

	i := 0
	n := len(pathData)

	for i < n {
		// Skip whitespace
		for i < n && (pathData[i] == ' ' || pathData[i] == '\t' || pathData[i] == '\n' || pathData[i] == '\r' || pathData[i] == ',') {
			result.WriteByte(pathData[i])
			i++
		}
		if i >= n {
			break
		}

		cmd := pathData[i]
		i++

		switch cmd {
		case 'M': // Absolute moveto
			result.WriteByte('M')
			// Parse coordinate pairs
			for {
				x, y, newI, ok := parseCoordPair(pathData, i)
				if !ok {
					break
				}
				i = newI
				curX, curY = x, y
				startX, startY = x, y
				result.WriteString(formatFloat(x) + "," + formatFloat(y))
				// Check for more coordinates (implicit lineto)
				if !hasMoreNumbers(pathData, i) {
					break
				}
				result.WriteString(" L") // oksvg: explicit command for implicit lineto
			}

		case 'm': // Relative moveto - convert to absolute
			result.WriteByte('M')
			first := true
			for {
				dx, dy, newI, ok := parseCoordPair(pathData, i)
				if !ok {
					break
				}
				i = newI
				if first {
					curX += dx
					curY += dy
					first = false
				} else {
					curX += dx
					curY += dy
				}
				startX, startY = curX, curY
				result.WriteString(formatFloat(curX) + "," + formatFloat(curY))
				if !hasMoreNumbers(pathData, i) {
					break
				}
				result.WriteString(" L") // oksvg: explicit command for implicit lineto
			}

		case 'L': // Absolute lineto
			result.WriteByte('L')
			for {
				x, y, newI, ok := parseCoordPair(pathData, i)
				if !ok {
					break
				}
				i = newI
				curX, curY = x, y
				result.WriteString(formatFloat(x) + "," + formatFloat(y))
				if !hasMoreNumbers(pathData, i) {
					break
				}
				result.WriteString(" L") // oksvg: explicit command
			}

		case 'l': // Relative lineto - convert to absolute
			result.WriteByte('L')
			for {
				dx, dy, newI, ok := parseCoordPair(pathData, i)
				if !ok {
					break
				}
				i = newI
				curX += dx
				curY += dy
				result.WriteString(formatFloat(curX) + "," + formatFloat(curY))
				if !hasMoreNumbers(pathData, i) {
					break
				}
				result.WriteString(" L") // oksvg: explicit command
			}

		case 'H': // Absolute horizontal lineto
			result.WriteByte('H')
			for {
				x, newI, ok := parseNumber(pathData, i)
				if !ok {
					break
				}
				i = newI
				curX = x
				result.WriteString(formatFloat(x))
				if !hasMoreNumbers(pathData, i) {
					break
				}
				result.WriteString(" H") // oksvg: explicit command
			}

		case 'h': // Relative horizontal lineto - convert to absolute
			result.WriteByte('H')
			for {
				dx, newI, ok := parseNumber(pathData, i)
				if !ok {
					break
				}
				i = newI
				curX += dx
				result.WriteString(formatFloat(curX))
				if !hasMoreNumbers(pathData, i) {
					break
				}
				result.WriteString(" H") // oksvg: explicit command
			}

		case 'V': // Absolute vertical lineto
			result.WriteByte('V')
			for {
				y, newI, ok := parseNumber(pathData, i)
				if !ok {
					break
				}
				i = newI
				curY = y
				result.WriteString(formatFloat(y))
				if !hasMoreNumbers(pathData, i) {
					break
				}
				result.WriteString(" V") // oksvg: explicit command
			}

		case 'v': // Relative vertical lineto - convert to absolute
			result.WriteByte('V')
			for {
				dy, newI, ok := parseNumber(pathData, i)
				if !ok {
					break
				}
				i = newI
				curY += dy
				result.WriteString(formatFloat(curY))
				if !hasMoreNumbers(pathData, i) {
					break
				}
				result.WriteString(" V") // oksvg: explicit command
			}

		case 'A': // Absolute arc - pass through
			result.WriteByte('A')
			for {
				// rx ry x-axis-rotation large-arc-flag sweep-flag x y
				rx, newI, ok := parseNumber(pathData, i)
				if !ok {
					break
				}
				i = newI
				ry, newI, ok := parseNumber(pathData, i)
				if !ok {
					break
				}
				i = newI
				rotation, newI, ok := parseNumber(pathData, i)
				if !ok {
					break
				}
				i = newI
				largeArc, newI, ok := parseNumber(pathData, i)
				if !ok {
					break
				}
				i = newI
				sweep, newI, ok := parseNumber(pathData, i)
				if !ok {
					break
				}
				i = newI
				x, y, newI, ok := parseCoordPair(pathData, i)
				if !ok {
					break
				}
				i = newI
				curX, curY = x, y
				result.WriteString(formatFloat(rx) + "," + formatFloat(ry) + " " + formatFloat(rotation) + " " + formatFloat(largeArc) + " " + formatFloat(sweep) + " " + formatFloat(x) + "," + formatFloat(y))
				if !hasMoreNumbers(pathData, i) {
					break
				}
				result.WriteString(" A") // oksvg requires explicit command for each segment
			}

		case 'a': // Relative arc - convert to absolute (THIS IS THE KEY FIX)
			result.WriteByte('A')
			for {
				// rx ry x-axis-rotation large-arc-flag sweep-flag dx dy
				rx, newI, ok := parseNumber(pathData, i)
				if !ok {
					break
				}
				i = newI
				ry, newI, ok := parseNumber(pathData, i)
				if !ok {
					break
				}
				i = newI
				rotation, newI, ok := parseNumber(pathData, i)
				if !ok {
					break
				}
				i = newI
				largeArc, newI, ok := parseNumber(pathData, i)
				if !ok {
					break
				}
				i = newI
				sweep, newI, ok := parseNumber(pathData, i)
				if !ok {
					break
				}
				i = newI
				dx, dy, newI, ok := parseCoordPair(pathData, i)
				if !ok {
					break
				}
				i = newI
				curX += dx
				curY += dy
				result.WriteString(formatFloat(rx) + "," + formatFloat(ry) + " " + formatFloat(rotation) + " " + formatFloat(largeArc) + " " + formatFloat(sweep) + " " + formatFloat(curX) + "," + formatFloat(curY))
				if !hasMoreNumbers(pathData, i) {
					break
				}
				result.WriteString(" A") // oksvg requires explicit command for each segment
			}

		case 'Q': // Absolute quadratic bezier
			result.WriteByte('Q')
			for {
				x1, y1, newI, ok := parseCoordPair(pathData, i)
				if !ok {
					break
				}
				i = newI
				x, y, newI, ok := parseCoordPair(pathData, i)
				if !ok {
					break
				}
				i = newI
				curX, curY = x, y
				result.WriteString(formatFloat(x1) + "," + formatFloat(y1) + " " + formatFloat(x) + "," + formatFloat(y))
				if !hasMoreNumbers(pathData, i) {
					break
				}
				result.WriteString(" Q") // oksvg: explicit command
			}

		case 'q': // Relative quadratic bezier - convert to absolute
			result.WriteByte('Q')
			for {
				dx1, dy1, newI, ok := parseCoordPair(pathData, i)
				if !ok {
					break
				}
				i = newI
				dx, dy, newI, ok := parseCoordPair(pathData, i)
				if !ok {
					break
				}
				i = newI
				x1, y1 := curX+dx1, curY+dy1
				curX += dx
				curY += dy
				result.WriteString(formatFloat(x1) + "," + formatFloat(y1) + " " + formatFloat(curX) + "," + formatFloat(curY))
				if !hasMoreNumbers(pathData, i) {
					break
				}
				result.WriteString(" Q") // oksvg: explicit command
			}

		case 'C': // Absolute cubic bezier
			result.WriteByte('C')
			for {
				x1, y1, newI, ok := parseCoordPair(pathData, i)
				if !ok {
					break
				}
				i = newI
				x2, y2, newI, ok := parseCoordPair(pathData, i)
				if !ok {
					break
				}
				i = newI
				x, y, newI, ok := parseCoordPair(pathData, i)
				if !ok {
					break
				}
				i = newI
				curX, curY = x, y
				result.WriteString(formatFloat(x1) + "," + formatFloat(y1) + " " + formatFloat(x2) + "," + formatFloat(y2) + " " + formatFloat(x) + "," + formatFloat(y))
				if !hasMoreNumbers(pathData, i) {
					break
				}
				result.WriteString(" C") // oksvg: explicit command
			}

		case 'c': // Relative cubic bezier - convert to absolute
			result.WriteByte('C')
			for {
				dx1, dy1, newI, ok := parseCoordPair(pathData, i)
				if !ok {
					break
				}
				i = newI
				dx2, dy2, newI, ok := parseCoordPair(pathData, i)
				if !ok {
					break
				}
				i = newI
				dx, dy, newI, ok := parseCoordPair(pathData, i)
				if !ok {
					break
				}
				i = newI
				x1, y1 := curX+dx1, curY+dy1
				x2, y2 := curX+dx2, curY+dy2
				curX += dx
				curY += dy
				result.WriteString(formatFloat(x1) + "," + formatFloat(y1) + " " + formatFloat(x2) + "," + formatFloat(y2) + " " + formatFloat(curX) + "," + formatFloat(curY))
				if !hasMoreNumbers(pathData, i) {
					break
				}
				result.WriteString(" C") // oksvg: explicit command
			}

		case 'Z', 'z': // Close path
			result.WriteByte('Z')
			curX, curY = startX, startY

		default:
			// Unknown command, pass through
			result.WriteByte(cmd)
		}
	}

	return result.String()
}

// formatFloat formats a float with reasonable precision to avoid floating point artifacts
func formatFloat(f float64) string {
	// Round to 4 decimal places to avoid floating point precision issues
	// that can cause arc commands to not close properly
	rounded := float64(int(f*10000+0.5)) / 10000
	return fmt.Sprintf("%g", rounded)
}

// parseNumber parses a number from path data starting at index i
func parseNumber(pathData string, i int) (float64, int, bool) {
	n := len(pathData)

	// Skip whitespace and commas
	for i < n && (pathData[i] == ' ' || pathData[i] == '\t' || pathData[i] == '\n' || pathData[i] == '\r' || pathData[i] == ',') {
		i++
	}
	if i >= n {
		return 0, i, false
	}

	// Check if this is a number (digit, minus, or decimal point)
	if !isNumberStart(pathData[i]) {
		return 0, i, false
	}

	start := i

	// Handle sign
	if i < n && (pathData[i] == '-' || pathData[i] == '+') {
		i++
	}

	// Parse digits before decimal
	for i < n && pathData[i] >= '0' && pathData[i] <= '9' {
		i++
	}

	// Parse decimal point and digits after
	if i < n && pathData[i] == '.' {
		i++
		for i < n && pathData[i] >= '0' && pathData[i] <= '9' {
			i++
		}
	}

	// Parse exponent
	if i < n && (pathData[i] == 'e' || pathData[i] == 'E') {
		i++
		if i < n && (pathData[i] == '-' || pathData[i] == '+') {
			i++
		}
		for i < n && pathData[i] >= '0' && pathData[i] <= '9' {
			i++
		}
	}

	if i == start {
		return 0, i, false
	}

	numStr := pathData[start:i]
	var val float64
	fmt.Sscanf(numStr, "%f", &val)
	return val, i, true
}

// parseCoordPair parses two numbers (x,y coordinate pair) from path data
func parseCoordPair(pathData string, i int) (float64, float64, int, bool) {
	x, i, ok := parseNumber(pathData, i)
	if !ok {
		return 0, 0, i, false
	}
	y, i, ok := parseNumber(pathData, i)
	if !ok {
		return 0, 0, i, false
	}
	return x, y, i, true
}

// isNumberStart checks if a character can start a number
func isNumberStart(c byte) bool {
	return (c >= '0' && c <= '9') || c == '-' || c == '+' || c == '.'
}

// hasMoreNumbers checks if there are more numbers following
func hasMoreNumbers(pathData string, i int) bool {
	n := len(pathData)
	// Skip whitespace and commas
	for i < n && (pathData[i] == ' ' || pathData[i] == '\t' || pathData[i] == '\n' || pathData[i] == '\r' || pathData[i] == ',') {
		i++
	}
	if i >= n {
		return false
	}
	return isNumberStart(pathData[i])
}

// NormalizeSVG converts inline style attributes to element attributes
// and fixes path commands for oksvg compatibility
// This improves compatibility with oksvg which handles attributes better than CSS styles
func NormalizeSVG(svg string) string {
	// First, normalize path commands (convert relative arcs to absolute)
	svg = normalizePathCommands(svg)

	// Then convert style attributes to element attributes
	return styleAttrRegex.ReplaceAllStringFunc(svg, func(match string) string {
		// Extract the style content
		styleMatch := styleAttrRegex.FindStringSubmatch(match)
		if len(styleMatch) < 2 {
			return match
		}
		styleContent := styleMatch[1]

		// Parse CSS properties
		var attrs []string
		props := cssPropRegex.FindAllStringSubmatch(styleContent, -1)

		for _, prop := range props {
			if len(prop) < 3 {
				continue
			}
			name := strings.TrimSpace(prop[1])
			value := strings.TrimSpace(prop[2])

			// Convert CSS property to SVG attribute
			attrName, attrValue := cssToSVGAttribute(name, value)
			if attrName != "" {
				attrs = append(attrs, fmt.Sprintf(`%s="%s"`, attrName, attrValue))
			}
		}

		if len(attrs) == 0 {
			return ""
		}
		return " " + strings.Join(attrs, " ")
	})
}

// cssToSVGAttribute converts a CSS property name/value to SVG attribute name/value
func cssToSVGAttribute(name, value string) (string, string) {
	// Strip px units for numeric attributes
	if pxMatch := pxUnitRegex.FindStringSubmatch(value); len(pxMatch) > 1 {
		value = pxMatch[1]
	}

	// Map CSS properties to SVG attributes
	switch name {
	case "fill":
		return "fill", value
	case "stroke":
		return "stroke", value
	case "stroke-width":
		return "stroke-width", value
	case "stroke-linecap":
		return "stroke-linecap", value
	case "stroke-linejoin":
		return "stroke-linejoin", value
	case "stroke-miterlimit":
		return "stroke-miterlimit", value
	case "stroke-dasharray":
		return "stroke-dasharray", value
	case "stroke-dashoffset":
		return "stroke-dashoffset", value
	case "stroke-opacity":
		return "stroke-opacity", value
	case "fill-opacity":
		return "fill-opacity", value
	case "opacity":
		return "opacity", value
	case "fill-rule":
		return "fill-rule", value
	case "clip-rule":
		return "clip-rule", value
	case "font-family":
		return "font-family", value
	case "font-size":
		return "font-size", value
	case "font-weight":
		return "font-weight", value
	case "text-anchor":
		return "text-anchor", value
	case "dominant-baseline":
		return "dominant-baseline", value
	case "display":
		// Skip display:none elements entirely would need different handling
		return "display", value
	case "visibility":
		return "visibility", value
	default:
		// Unknown property - skip it
		return "", ""
	}
}

// Preview returns a simplified preview of the avatar (just the design indices)
func (c Config) Preview() map[string]string {
	result := make(map[string]string)

	parts := []struct {
		name  string
		value int
	}{
		{"env", c.Env},
		{"clo", c.Clo},
		{"head", c.Head},
		{"mouth", c.Mouth},
		{"eyes", c.Eyes},
		{"top", c.Top},
	}

	for _, p := range parts {
		design, variant := getDesignAndVariant(p.value)
		result[p.name] = design + variant
	}

	return result
}

// ToImage generates a rasterized image of the avatar at the specified size
func (c Config) ToImage(size int, includeEnv bool) (image.Image, error) {
	svg := c.ToSVGWithEnv(includeEnv)

	// Normalize SVG: convert inline style attributes to element attributes
	// This improves oksvg compatibility, especially for stroke properties
	svg = NormalizeSVG(svg)

	// Parse SVG
	icon, err := oksvg.ReadIconStream(bytes.NewReader([]byte(svg)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse SVG: %w", err)
	}

	// Set target size
	icon.SetTarget(0, 0, float64(size), float64(size))

	// Create RGBA image
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Rasterize
	scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
	raster := rasterx.NewDasher(size, size, scanner)
	icon.Draw(raster, 1.0)

	return img, nil
}
