package main

import (
	"encoding/json"
	"fmt"
	"image/png"
	"log"
	"net/http"

	"songmartyn/internal/avatar"
)

// handleAvatarPNG generates a PNG avatar from config parameters
func handleAvatarPNG(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()

	// Parse config from query parameters
	config := avatar.Config{
		Env:   parseIntParam(q.Get("env"), 0),
		Clo:   parseIntParam(q.Get("clo"), 0),
		Head:  parseIntParam(q.Get("head"), 0),
		Mouth: parseIntParam(q.Get("mouth"), 0),
		Eyes:  parseIntParam(q.Get("eyes"), 0),
		Top:   parseIntParam(q.Get("top"), 0),
	}

	// Parse custom colors (optional)
	if hasColorParams(q) {
		config.Colors = &avatar.Colors{
			Env:   q.Get("c_env"),
			Clo:   q.Get("c_clo"),
			Head:  q.Get("c_head"),
			Mouth: q.Get("c_mouth"),
			Eyes:  q.Get("c_eyes"),
			Top:   q.Get("c_top"),
		}
	}

	// Check if JSON config is provided
	if configJSON := q.Get("config"); configJSON != "" {
		if parsed, err := avatar.FromJSON(configJSON); err == nil {
			config = parsed
		}
	}

	config.Normalize()

	// Parse size (default 256)
	size := parseIntParam(q.Get("size"), 256)
	if size < 32 {
		size = 32
	}
	if size > 512 {
		size = 512
	}

	// Check if we should include environment (background)
	includeEnv := q.Get("noenv") != "true"

	// Generate PNG
	img, err := config.ToImage(size, includeEnv)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate avatar: %v", err), http.StatusInternalServerError)
		return
	}

	// Encode as PNG
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if err := png.Encode(w, img); err != nil {
		log.Printf("Failed to encode PNG: %v", err)
	}
}

// handleAvatarDebug returns debug info about avatar rendering
func handleAvatarDebug(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()

	// Parse config from query parameters
	config := avatar.Config{
		Env:   parseIntParam(q.Get("env"), 0),
		Clo:   parseIntParam(q.Get("clo"), 0),
		Head:  parseIntParam(q.Get("head"), 0),
		Mouth: parseIntParam(q.Get("mouth"), 0),
		Eyes:  parseIntParam(q.Get("eyes"), 0),
		Top:   parseIntParam(q.Get("top"), 0),
	}

	// Parse custom colors (optional)
	if hasColorParams(q) {
		config.Colors = &avatar.Colors{
			Env:   q.Get("c_env"),
			Clo:   q.Get("c_clo"),
			Head:  q.Get("c_head"),
			Mouth: q.Get("c_mouth"),
			Eyes:  q.Get("c_eyes"),
			Top:   q.Get("c_top"),
		}
	}

	// Check if JSON config is provided
	if configJSON := q.Get("config"); configJSON != "" {
		if parsed, err := avatar.FromJSON(configJSON); err == nil {
			config = parsed
		}
	}

	config.Normalize()

	includeEnv := q.Get("noenv") != "true"

	// Get raw SVG and normalized SVG
	rawSVG := config.ToSVGWithEnv(includeEnv)
	normalizedSVG := avatar.NormalizeSVG(rawSVG)

	response := map[string]interface{}{
		"config":         config,
		"raw_svg":        rawSVG,
		"normalized_svg": normalizedSVG,
		"preview":        config.Preview(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
