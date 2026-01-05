import { useState, useRef, useEffect } from 'react';

// Color palette - 20 colors organized in groups
export const COLOR_PALETTE = {
  // Skin tones
  skin: [
    { hex: '#FFDFC4', name: 'Light' },
    { hex: '#F0C08A', name: 'Fair' },
    { hex: '#D2956B', name: 'Medium' },
    { hex: '#A67449', name: 'Tan' },
    { hex: '#6B4423', name: 'Dark' },
  ],
  // Basic colors
  basic: [
    { hex: '#FF4444', name: 'Red' },
    { hex: '#FF9500', name: 'Orange' },
    { hex: '#FFCC00', name: 'Yellow' },
    { hex: '#4CAF50', name: 'Green' },
    { hex: '#2196F3', name: 'Blue' },
    { hex: '#9C27B0', name: 'Purple' },
    { hex: '#E91E8C', name: 'Pink' },
  ],
  // Neutrals
  neutral: [
    { hex: '#FFFFFF', name: 'White' },
    { hex: '#9E9E9E', name: 'Gray' },
    { hex: '#424242', name: 'Dark Gray' },
    { hex: '#000000', name: 'Black' },
  ],
  // Fun/accent colors
  accent: [
    { hex: '#00BCD4', name: 'Cyan' },
    { hex: '#FF6B9D', name: 'Coral' },
    { hex: '#7C4DFF', name: 'Violet' },
    { hex: '#00E676', name: 'Mint' },
  ],
};

// Flat array of all colors for easy iteration
export const ALL_COLORS = [
  ...COLOR_PALETTE.skin,
  ...COLOR_PALETTE.basic,
  ...COLOR_PALETTE.neutral,
  ...COLOR_PALETTE.accent,
];

interface ColorPickerProps {
  value?: string; // Current color (hex)
  onChange: (color: string | undefined) => void;
  label?: string;
}

export function ColorPicker({ value, onChange, label }: ColorPickerProps) {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // Close on click outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen]);

  const handleColorSelect = (hex: string) => {
    onChange(hex);
    setIsOpen(false);
  };

  const handleReset = () => {
    onChange(undefined);
    setIsOpen(false);
  };

  return (
    <div className="relative" ref={containerRef}>
      {/* Color button trigger */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="p-2 rounded-lg bg-gray-700 hover:bg-gray-600 active:bg-gray-500 transition-colors relative"
        aria-label={label ? `Pick ${label} color` : 'Pick color'}
        title={value || 'Default color'}
      >
        {value ? (
          <div
            className="w-4 h-4 rounded border border-white/30"
            style={{ backgroundColor: value }}
          />
        ) : (
          <svg className="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
          </svg>
        )}
      </button>

      {/* Color picker popover */}
      {isOpen && (
        <div className="absolute z-50 mt-2 right-0 p-3 bg-gray-800 rounded-xl shadow-xl border border-gray-700 min-w-[200px]">
          {/* Reset button */}
          <button
            onClick={handleReset}
            className="w-full mb-2 px-3 py-1.5 text-sm text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors flex items-center gap-2"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            Reset to default
          </button>

          {/* Skin tones section */}
          <div className="mb-2">
            <div className="text-xs text-gray-500 mb-1">Skin Tones</div>
            <div className="flex gap-1">
              {COLOR_PALETTE.skin.map((color) => (
                <ColorSwatch
                  key={color.hex}
                  color={color}
                  isSelected={value === color.hex}
                  onClick={() => handleColorSelect(color.hex)}
                />
              ))}
            </div>
          </div>

          {/* Basic colors section */}
          <div className="mb-2">
            <div className="text-xs text-gray-500 mb-1">Colors</div>
            <div className="flex flex-wrap gap-1">
              {COLOR_PALETTE.basic.map((color) => (
                <ColorSwatch
                  key={color.hex}
                  color={color}
                  isSelected={value === color.hex}
                  onClick={() => handleColorSelect(color.hex)}
                />
              ))}
            </div>
          </div>

          {/* Neutrals & accent section */}
          <div>
            <div className="text-xs text-gray-500 mb-1">Neutrals & Accents</div>
            <div className="flex flex-wrap gap-1">
              {[...COLOR_PALETTE.neutral, ...COLOR_PALETTE.accent].map((color) => (
                <ColorSwatch
                  key={color.hex}
                  color={color}
                  isSelected={value === color.hex}
                  onClick={() => handleColorSelect(color.hex)}
                />
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

interface ColorSwatchProps {
  color: { hex: string; name: string };
  isSelected: boolean;
  onClick: () => void;
}

function ColorSwatch({ color, isSelected, onClick }: ColorSwatchProps) {
  const isLight = isLightColor(color.hex);

  return (
    <button
      onClick={onClick}
      className={`
        w-7 h-7 rounded-lg transition-all
        ${isSelected ? 'ring-2 ring-purple-500 ring-offset-2 ring-offset-gray-800 scale-110' : 'hover:scale-105'}
        ${color.hex === '#FFFFFF' ? 'border border-gray-600' : ''}
      `}
      style={{ backgroundColor: color.hex }}
      title={color.name}
    >
      {isSelected && (
        <svg
          className={`w-4 h-4 mx-auto ${isLight ? 'text-gray-800' : 'text-white'}`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
        </svg>
      )}
    </button>
  );
}

// Helper to determine if a color is light (for contrast)
function isLightColor(hex: string): boolean {
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  // Using relative luminance formula
  const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
  return luminance > 0.5;
}

// Get a random color from the palette
export function getRandomColor(): string {
  const randomIndex = Math.floor(Math.random() * ALL_COLORS.length);
  return ALL_COLORS[randomIndex].hex;
}

// Get random colors for all avatar parts
export function getRandomColors(): {
  env: string;
  clo: string;
  head: string;
  mouth: string;
  eyes: string;
  top: string;
} {
  // Use skin tones for head, random for others
  const skinColors = COLOR_PALETTE.skin;
  const randomSkin = skinColors[Math.floor(Math.random() * skinColors.length)].hex;

  return {
    env: getRandomColor(),
    clo: getRandomColor(),
    head: randomSkin,
    mouth: getRandomColor(),
    eyes: getRandomColor(),
    top: getRandomColor(),
  };
}
