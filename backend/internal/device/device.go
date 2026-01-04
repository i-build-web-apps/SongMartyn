package device

import (
	"regexp"
	"strings"
)

// DeviceInfo contains parsed device information
type DeviceInfo struct {
	Name     string // Friendly name like "iPhone" or "Windows PC"
	Type     string // "mobile", "tablet", "desktop", "unknown"
	OS       string // "iOS", "Android", "Windows", "macOS", "Linux"
	Browser  string // "Safari", "Chrome", "Firefox", etc.
}

// ParseUserAgent extracts device information from a User-Agent string
func ParseUserAgent(ua string) DeviceInfo {
	info := DeviceInfo{
		Name:    "Unknown Device",
		Type:    "unknown",
		OS:      "Unknown",
		Browser: "Unknown",
	}

	if ua == "" {
		return info
	}

	ua = strings.ToLower(ua)

	// Detect OS and device type
	switch {
	case strings.Contains(ua, "iphone"):
		info.Name = "iPhone"
		info.Type = "mobile"
		info.OS = "iOS"
		// Try to get specific model
		if model := extractiPhoneModel(ua); model != "" {
			info.Name = model
		}

	case strings.Contains(ua, "ipad"):
		info.Name = "iPad"
		info.Type = "tablet"
		info.OS = "iOS"

	case strings.Contains(ua, "android"):
		info.OS = "Android"
		if strings.Contains(ua, "mobile") {
			info.Name = "Android Phone"
			info.Type = "mobile"
		} else if strings.Contains(ua, "tablet") {
			info.Name = "Android Tablet"
			info.Type = "tablet"
		} else {
			info.Name = "Android Device"
			info.Type = "mobile"
		}
		// Try to get specific model
		if model := extractAndroidModel(ua); model != "" {
			info.Name = model
		}

	case strings.Contains(ua, "macintosh") || strings.Contains(ua, "mac os"):
		info.Name = "Mac"
		info.Type = "desktop"
		info.OS = "macOS"

	case strings.Contains(ua, "windows"):
		info.Name = "Windows PC"
		info.Type = "desktop"
		info.OS = "Windows"

	case strings.Contains(ua, "linux"):
		info.Name = "Linux PC"
		info.Type = "desktop"
		info.OS = "Linux"

	case strings.Contains(ua, "cros"):
		info.Name = "Chromebook"
		info.Type = "desktop"
		info.OS = "ChromeOS"
	}

	// Detect browser
	switch {
	case strings.Contains(ua, "firefox"):
		info.Browser = "Firefox"
	case strings.Contains(ua, "edg"):
		info.Browser = "Edge"
	case strings.Contains(ua, "opr") || strings.Contains(ua, "opera"):
		info.Browser = "Opera"
	case strings.Contains(ua, "chrome") && !strings.Contains(ua, "chromium"):
		info.Browser = "Chrome"
	case strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome"):
		info.Browser = "Safari"
	case strings.Contains(ua, "chromium"):
		info.Browser = "Chromium"
	}

	return info
}

// extractiPhoneModel tries to extract a specific iPhone model
func extractiPhoneModel(ua string) string {
	// iPhone models are often in the format "iPhone14,2" etc.
	// But User-Agent typically just says "iPhone" so we can't get specific model
	// We could potentially detect iOS version
	re := regexp.MustCompile(`cpu iphone os (\d+)`)
	if matches := re.FindStringSubmatch(ua); len(matches) > 1 {
		return "iPhone (iOS " + matches[1] + ")"
	}
	return ""
}

// extractAndroidModel tries to extract a specific Android device model
func extractAndroidModel(ua string) string {
	// Android User-Agents often contain the device model
	// Format: "Android X.X; MODEL Build/..."
	re := regexp.MustCompile(`android [\d.]+;\s*([^)]+?)\s*(?:build|;|\))`)
	if matches := re.FindStringSubmatch(ua); len(matches) > 1 {
		model := strings.TrimSpace(matches[1])
		// Clean up common prefixes
		model = strings.TrimPrefix(model, "en-us; ")
		model = strings.TrimPrefix(model, "en-gb; ")
		if len(model) > 0 && len(model) < 30 {
			return model
		}
	}
	return ""
}

// GetFriendlyName returns a short, friendly device name
func GetFriendlyName(ua string) string {
	info := ParseUserAgent(ua)
	return info.Name
}

// GetDeviceIcon returns an emoji icon for the device type
func GetDeviceIcon(ua string) string {
	info := ParseUserAgent(ua)
	switch info.Type {
	case "mobile":
		return "📱"
	case "tablet":
		return "📱"
	case "desktop":
		switch info.OS {
		case "macOS":
			return "🖥️"
		case "Windows":
			return "💻"
		case "Linux":
			return "🐧"
		default:
			return "🖥️"
		}
	default:
		return "📟"
	}
}
