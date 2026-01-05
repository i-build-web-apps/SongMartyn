package holdingscreen

import (
	"bytes"
	"crypto/tls"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/fogleman/gg"
	"github.com/nfnt/resize"
	"songmartyn/internal/avatar"
	"songmartyn/pkg/models"
)

// HTTP client that skips TLS verification for localhost
var insecureClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

//go:embed assets/logo.jpeg
var logoFS embed.FS

const (
	canvasWidth  = 1920
	canvasHeight = 1080
)

// Colors matching the brand
var (
	cyanColor   = color.RGBA{0, 188, 212, 255}   // #00bcd4 - for "SONG"
	yellowColor = color.RGBA{234, 179, 8, 255}   // #eab308 - for "MARTYN"
	whiteColor  = color.RGBA{255, 255, 255, 255}
	grayColor   = color.RGBA{160, 160, 160, 255}
)

// Generator creates holding screen images
type Generator struct {
	logoImage    image.Image
	tempDir      string
	avatarAPIURL string
}

// NewGenerator creates a new holding screen generator
func NewGenerator(tempDir, avatarAPIURL string) (*Generator, error) {
	// Load embedded logo
	logoData, err := logoFS.ReadFile("assets/logo.jpeg")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded logo: %w", err)
	}

	logoImg, err := jpeg.Decode(bytes.NewReader(logoData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode logo: %w", err)
	}

	// Ensure temp directory exists
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	return &Generator{
		logoImage:    logoImg,
		tempDir:      tempDir,
		avatarAPIURL: avatarAPIURL,
	}, nil
}

// NextUpInfo contains information about the next song and singer
type NextUpInfo struct {
	SongTitle    string
	SongArtist   string
	SingerName   string
	AvatarConfig *models.AvatarConfig
}

// Generate creates a holding screen image and returns the file path
func (g *Generator) Generate(connectURL string, nextUp *NextUpInfo) (string, error) {
	// Create canvas
	dc := gg.NewContext(canvasWidth, canvasHeight)

	// Draw logo as full background (cover the canvas)
	g.drawBackground(dc)

	// Draw semi-transparent overlay at bottom for content area
	g.drawBottomOverlay(dc)

	// Draw QR code section (bottom left)
	g.drawQRSection(dc, connectURL)

	// Draw "Next Up" section (bottom right) - always show, with placeholder if no song
	g.drawNextUpSection(dc, nextUp)

	// Save to temp file
	outputPath := filepath.Join(g.tempDir, "holding-screen.png")
	if err := dc.SavePNG(outputPath); err != nil {
		return "", fmt.Errorf("failed to save holding screen: %w", err)
	}

	return outputPath, nil
}

// drawBackground draws the logo scaled to cover the entire canvas
func (g *Generator) drawBackground(dc *gg.Context) {
	bounds := g.logoImage.Bounds()
	srcW := float64(bounds.Dx())
	srcH := float64(bounds.Dy())

	// Calculate scale to cover (fill) the canvas while maintaining aspect ratio
	scaleX := float64(canvasWidth) / srcW
	scaleY := float64(canvasHeight) / srcH
	scale := scaleX
	if scaleY > scaleX {
		scale = scaleY
	}

	// Calculate new dimensions
	newW := uint(srcW * scale)
	newH := uint(srcH * scale)

	// Resize the image
	resized := resize.Resize(newW, newH, g.logoImage, resize.Lanczos3)

	// Calculate offset to center the image
	offsetX := (int(newW) - canvasWidth) / 2
	offsetY := (int(newH) - canvasHeight) / 2

	// Create a sub-image that fits the canvas
	subImg := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))
	draw.Draw(subImg, subImg.Bounds(), resized, image.Point{offsetX, offsetY}, draw.Src)

	dc.DrawImage(subImg, 0, 0)
}

// drawBottomOverlay draws a semi-transparent black gradient at the bottom
func (g *Generator) drawBottomOverlay(dc *gg.Context) {
	// Draw gradient from transparent to semi-opaque black at the bottom
	overlayHeight := 350.0
	startY := float64(canvasHeight) - overlayHeight

	for y := 0; y < int(overlayHeight); y++ {
		// Gradient from 0 alpha at top to 200 alpha at bottom
		alpha := float64(y) / overlayHeight * 0.85
		dc.SetRGBA(0, 0, 0, alpha)
		dc.DrawRectangle(0, startY+float64(y), canvasWidth, 1)
		dc.Fill()
	}
}

// drawQRSection draws the QR code and connection info on bottom left
func (g *Generator) drawQRSection(dc *gg.Context, connectURL string) {
	qrSize := 240 // Large for room visibility
	padding := 50.0
	qrX := padding
	qrY := float64(canvasHeight) - float64(qrSize) - padding - 30

	// Draw semi-transparent background box for QR section
	boxPadding := 20.0
	dc.SetRGBA(0, 0, 0, 0.6)
	dc.DrawRoundedRectangle(qrX-boxPadding, qrY-boxPadding-40, float64(qrSize)+boxPadding*2+320, float64(qrSize)+boxPadding*2+50, 16)
	dc.Fill()

	// Fetch QR code
	qrURL := fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=%dx%d&data=%s&bgcolor=ffffff&color=000000",
		qrSize, qrSize, url.QueryEscape(connectURL))

	qrImg, err := fetchImage(qrURL)
	if err == nil && qrImg != nil {
		dc.DrawImage(qrImg, int(qrX), int(qrY))
	} else {
		// Fallback placeholder
		dc.SetRGBA(1, 1, 1, 0.3)
		dc.DrawRoundedRectangle(qrX, qrY, float64(qrSize), float64(qrSize), 8)
		dc.Fill()
	}

	// Draw text to the right of QR
	textX := qrX + float64(qrSize) + 30
	textY := qrY + 50

	// "Scan to join!" in yellow
	dc.SetColor(yellowColor)
	if err := loadFont(dc, 42); err == nil {
		dc.DrawString("Scan to join!", textX, textY)
	}

	// URL in white
	dc.SetColor(whiteColor)
	if err := loadFont(dc, 32); err == nil {
		dc.DrawString(connectURL, textX, textY+55)
	}

	// Subtitle
	dc.SetColor(grayColor)
	if err := loadFont(dc, 24); err == nil {
		dc.DrawString("Join the karaoke session", textX, textY+100)
	}
}

// drawNextUpSection draws the "Next Up" info box on bottom right
func (g *Generator) drawNextUpSection(dc *gg.Context, nextUp *NextUpInfo) {
	boxWidth := 900.0  // ~50% of screen width
	boxHeight := 200.0 // Larger for room visibility
	padding := 50.0
	boxX := float64(canvasWidth) - boxWidth - padding
	boxY := float64(canvasHeight) - boxHeight - padding - 30
	innerPadding := 25.0
	avatarSize := 150.0 // Larger avatar

	// Draw semi-transparent background
	dc.SetRGBA(0, 0, 0, 0.6)
	dc.DrawRoundedRectangle(boxX, boxY, boxWidth, boxHeight, 16)
	dc.Fill()

	// Draw accent line on left
	dc.SetColor(yellowColor)
	dc.DrawRoundedRectangle(boxX, boxY, 6, boxHeight, 3)
	dc.Fill()

	// Draw "NEXT UP" label
	dc.SetColor(yellowColor)
	if err := loadFont(dc, 22); err == nil {
		dc.DrawString("NEXT UP", boxX+innerPadding+avatarSize+25, boxY+innerPadding+20)
	}

	// Avatar area
	avatarX := boxX + innerPadding
	avatarY := boxY + (boxHeight-avatarSize)/2

	if nextUp != nil && nextUp.AvatarConfig != nil {
		// Fetch and draw avatar
		avatarImg := g.generateAvatar(nextUp.AvatarConfig)
		if avatarImg != nil {
			bounds := avatarImg.Bounds()
			scale := avatarSize / float64(bounds.Dx())
			dc.Push()
			dc.Translate(avatarX, avatarY)
			dc.Scale(scale, scale)
			dc.DrawImage(avatarImg, 0, 0)
			dc.Pop()
		} else {
			g.drawPlaceholderAvatar(dc, avatarX, avatarY, avatarSize)
		}
	} else {
		g.drawPlaceholderAvatar(dc, avatarX, avatarY, avatarSize)
	}

	// Text area
	textX := boxX + innerPadding + avatarSize + 25

	if nextUp != nil && nextUp.SongTitle != "" {
		// Song title - large and readable
		dc.SetColor(whiteColor)
		if err := loadFont(dc, 36); err == nil {
			title := truncateString(nextUp.SongTitle, 35)
			dc.DrawString(title, textX, boxY+innerPadding+65)
		}

		// Artist
		dc.SetColor(grayColor)
		if err := loadFont(dc, 28); err == nil {
			dc.DrawString(nextUp.SongArtist, textX, boxY+innerPadding+105)
		}

		// Singer name
		dc.SetColor(cyanColor)
		if err := loadFont(dc, 24); err == nil {
			dc.DrawString(nextUp.SingerName, textX, boxY+innerPadding+145)
		}
	} else {
		// Placeholder text - large and readable
		dc.SetColor(grayColor)
		if err := loadFont(dc, 32); err == nil {
			dc.DrawString("Waiting for songs...", textX, boxY+innerPadding+70)
		}
		dc.SetColor(color.RGBA{100, 100, 100, 255})
		if err := loadFont(dc, 24); err == nil {
			dc.DrawString("Scan QR code to add a song!", textX, boxY+innerPadding+115)
		}
	}
}

// drawPlaceholderAvatar draws a placeholder avatar circle
func (g *Generator) drawPlaceholderAvatar(dc *gg.Context, x, y, size float64) {
	// Background circle
	dc.SetRGBA(0.3, 0.3, 0.3, 0.8)
	dc.DrawCircle(x+size/2, y+size/2, size/2)
	dc.Fill()

	// User icon - scale based on avatar size
	scale := size / 100.0
	dc.SetRGBA(0.5, 0.5, 0.5, 1)
	// Head
	dc.DrawCircle(x+size/2, y+size/2-10*scale, 18*scale)
	dc.Fill()
	// Body
	dc.DrawEllipse(x+size/2, y+size/2+30*scale, 28*scale, 20*scale)
	dc.Fill()
}

// generateAvatar generates an avatar image directly from the config
// This avoids HTTP requests and the associated TLS/timing issues
func (g *Generator) generateAvatar(config *models.AvatarConfig) image.Image {
	if config == nil {
		return nil
	}

	// Convert models.AvatarConfig to avatar.Config
	avatarConfig := avatar.Config{
		Env:   config.Env,
		Clo:   config.Clo,
		Head:  config.Head,
		Mouth: config.Mouth,
		Eyes:  config.Eyes,
		Top:   config.Top,
	}

	// Copy colors if present
	if config.Colors != nil {
		avatarConfig.Colors = &avatar.Colors{
			Env:   config.Colors.Env,
			Clo:   config.Colors.Clo,
			Head:  config.Colors.Head,
			Mouth: config.Colors.Mouth,
			Eyes:  config.Colors.Eyes,
			Top:   config.Colors.Top,
		}
	}

	// Generate image (256px, no environment/background)
	img, err := avatarConfig.ToImage(256, false)
	if err != nil {
		fmt.Printf("[Avatar] Failed to generate: %v\n", err)
		return nil
	}
	return img
}

// fetchImage fetches an image from a URL
func fetchImage(imageURL string) (image.Image, error) {
	var resp *http.Response
	var err error

	// Use insecure client for localhost (self-signed certs)
	if strings.Contains(imageURL, "localhost") || strings.Contains(imageURL, "127.0.0.1") {
		resp, err = insecureClient.Get(imageURL)
	} else {
		resp, err = http.Get(imageURL)
	}

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch image: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Try PNG first, then JPEG, then generic
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		img, err = jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			img, _, err = image.Decode(bytes.NewReader(data))
			if err != nil {
				return nil, fmt.Errorf("failed to decode image: %w", err)
			}
		}
	}

	return img, nil
}

// truncateString truncates a string to maxLen characters with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// SaveCurrentSingerAvatar saves the current singer's avatar to a PNG file
// Returns the file path where the avatar was saved
func (g *Generator) SaveCurrentSingerAvatar(config *models.AvatarConfig) (string, error) {
	outputPath := filepath.Join(g.tempDir, "current-singer-avatar.png")

	if config == nil {
		// No avatar config - create placeholder or remove file
		os.Remove(outputPath)
		return "", nil
	}

	// Fetch avatar image
	img := g.generateAvatar(config)
	if img == nil {
		os.Remove(outputPath)
		return "", fmt.Errorf("failed to fetch avatar")
	}

	// Save as PNG with transparency
	f, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create avatar file: %w", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return "", fmt.Errorf("failed to encode avatar PNG: %w", err)
	}

	return outputPath, nil
}

// GetAvatarFilePath returns the path where the current singer avatar is saved
func (g *Generator) GetAvatarFilePath() string {
	return filepath.Join(g.tempDir, "current-singer-avatar.png")
}

// loadFont tries to load a font face from various system locations
func loadFont(dc *gg.Context, size float64) error {
	fontPaths := []string{
		"/System/Library/Fonts/SFNS.ttf",
		"/System/Library/Fonts/SFNSRounded.ttf",
		"/Library/Fonts/Arial.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
		"C:\\Windows\\Fonts\\arial.ttf",
	}

	for _, path := range fontPaths {
		if err := dc.LoadFontFace(path, size); err == nil {
			return nil
		}
	}

	return fmt.Errorf("no suitable font found")
}
