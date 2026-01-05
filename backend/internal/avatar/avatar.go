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
