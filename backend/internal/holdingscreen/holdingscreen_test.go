package holdingscreen

import (
	"testing"
)

func TestStripEmoji(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no emojis",
			input:    "Welcome to Karaoke Night!",
			expected: "Welcome to Karaoke Night!",
		},
		{
			name:     "single emoji at start",
			input:    "🎤 Welcome to Karaoke Night!",
			expected: "Welcome to Karaoke Night!",
		},
		{
			name:     "single emoji at end",
			input:    "Welcome to Karaoke Night! 🎉",
			expected: "Welcome to Karaoke Night!",
		},
		{
			name:     "multiple emojis",
			input:    "🎤 Sing your heart out! 🎵🎶",
			expected: "Sing your heart out!",
		},
		{
			name:     "emoji only",
			input:    "🎤🎵🎶",
			expected: "",
		},
		{
			name:     "emojis with spaces",
			input:    "🎤 🎵 🎶",
			expected: "",
		},
		{
			name:     "mixed content",
			input:    "🔥 The stage is yours! ⭐ Let's party! 🎉",
			expected: "The stage is yours! Let's party!",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "fire emoji",
			input:    "🔥 Hot stuff!",
			expected: "Hot stuff!",
		},
		{
			name:     "sparkles and stars",
			input:    "✨ Sparkle ⭐ Star",
			expected: "Sparkle Star",
		},
		{
			name:     "party emojis",
			input:    "🎉🥳 Party time! 🎊",
			expected: "Party time!",
		},
		{
			name:     "music emojis",
			input:    "🎵 Music 🎶 Notes 🎸 Guitar",
			expected: "Music Notes Guitar",
		},
		{
			name:     "hearts",
			input:    "💜 Purple heart",
			expected: "Purple heart",
		},
		{
			name:     "cheers emoji",
			input:    "🍻 Cheers!",
			expected: "Cheers!",
		},
		{
			name:     "consecutive spaces after strip",
			input:    "Hello  🎤  World",
			expected: "Hello World",
		},
		{
			name:     "leading/trailing spaces after strip",
			input:    "  🎤 Test 🎵  ",
			expected: "Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripEmoji(tt.input)
			if result != tt.expected {
				t.Errorf("stripEmoji(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
