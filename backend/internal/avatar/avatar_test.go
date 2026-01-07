package avatar

import (
	"strings"
	"testing"
)

func TestNormalizeSVG(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple fill",
			input:    `<path d="M0 0" style="fill:#ff0000;"/>`,
			expected: `<path d="M0 0" fill="#ff0000"/>`,
		},
		{
			name:     "fill and stroke",
			input:    `<path d="M0 0" style="fill:#ff0000;stroke:#000;"/>`,
			expected: `<path d="M0 0" fill="#ff0000" stroke="#000"/>`,
		},
		{
			name:     "stroke with width in px",
			input:    `<path d="M0 0" style="stroke:#333;stroke-width:6.1999px;"/>`,
			expected: `<path d="M0 0" stroke="#333" stroke-width="6.1999"/>`,
		},
		{
			name:     "complex stroke properties",
			input:    `<path d="M0 0" style="fill:none;stroke-linecap:round;stroke-linejoin:round;stroke-width:5.9998px;stroke:#b8b8b8;"/>`,
			expected: `<path d="M0 0" fill="none" stroke-linecap="round" stroke-linejoin="round" stroke-width="5.9998" stroke="#b8b8b8"/>`,
		},
		{
			name:     "no style attribute",
			input:    `<path d="M0 0" fill="#ff0000"/>`,
			expected: `<path d="M0 0" fill="#ff0000"/>`,
		},
		{
			name:     "empty style",
			input:    `<path d="M0 0" style=""/>`,
			expected: `<path d="M0 0"/>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeSVG(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeSVG() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNormalizeSVGPreservesOtherAttributes(t *testing.T) {
	input := `<rect x="79.795" y="98.627" width="71.471" height="8.5859" ry="4.2929" style="fill:#b3b3b3;"/>`
	result := NormalizeSVG(input)

	// Should preserve x, y, width, height, ry attributes
	if !strings.Contains(result, `x="79.795"`) {
		t.Error("Lost x attribute")
	}
	if !strings.Contains(result, `ry="4.2929"`) {
		t.Error("Lost ry attribute")
	}
	// Should convert style to fill attribute
	if !strings.Contains(result, `fill="#b3b3b3"`) {
		t.Error("Style not converted to fill attribute")
	}
	// Should not have style attribute anymore
	if strings.Contains(result, "style=") {
		t.Error("style attribute should be removed")
	}
}

func TestToImage(t *testing.T) {
	// Test that ToImage doesn't panic and produces valid output
	config := Config{
		Env:   0,
		Clo:   0,
		Head:  0,
		Mouth: 0,
		Eyes:  0,
		Top:   0,
	}

	img, err := config.ToImage(128, true)
	if err != nil {
		t.Fatalf("ToImage failed: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 128 {
		t.Errorf("Image size = %dx%d, want 128x128", bounds.Dx(), bounds.Dy())
	}
}

func TestToImageAllDesigns(t *testing.T) {
	// Test all 16 designs render without error
	for design := 0; design < NumDesigns; design++ {
		config := Config{
			Env:   design,
			Clo:   design,
			Head:  design,
			Mouth: design,
			Eyes:  design,
			Top:   design,
		}

		_, err := config.ToImage(64, true)
		if err != nil {
			t.Errorf("Design %d failed to render: %v", design, err)
		}
	}
}

func TestToImageWithCustomColors(t *testing.T) {
	config := Config{
		Env:   5,
		Clo:   10,
		Head:  15,
		Mouth: 20,
		Eyes:  25,
		Top:   30,
		Colors: &Colors{
			Env:   "#FF0000",
			Clo:   "#00FF00",
			Head:  "#FFDFC4",
			Mouth: "#0000FF",
			Eyes:  "#FFFF00",
			Top:   "#FF00FF",
		},
	}

	img, err := config.ToImage(128, true)
	if err != nil {
		t.Fatalf("ToImage with custom colors failed: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 128 {
		t.Errorf("Image size = %dx%d, want 128x128", bounds.Dx(), bounds.Dy())
	}
}
