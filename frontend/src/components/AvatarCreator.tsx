import { useState, useEffect, useCallback } from 'react';
import type { AvatarConfig, AvatarColors } from '../types';
import { ColorPicker, getRandomColors } from './ColorPicker';

export type { AvatarConfig, AvatarColors };

const MAX_VALUE = 48;

// Part keys that have numeric values (exclude colors)
type PartKey = 'env' | 'clo' | 'head' | 'mouth' | 'eyes' | 'top';

// Part labels for the UI
const PART_LABELS: Record<PartKey, string> = {
  env: 'Background',
  clo: 'Clothes',
  head: 'Head',
  mouth: 'Mouth',
  eyes: 'Eyes',
  top: 'Hair/Top',
};

// Part order for display
const PART_ORDER: PartKey[] = ['env', 'top', 'head', 'eyes', 'mouth', 'clo'];

interface AvatarCreatorProps {
  initialConfig?: AvatarConfig;
  onSave: (config: AvatarConfig) => void;
  onCancel?: () => void;
  size?: number;
  showButtons?: boolean;
  randomizeTrigger?: number; // Increment to trigger randomize from parent
}

export function AvatarCreator({
  initialConfig,
  onSave,
  onCancel,
  size = 180,
  showButtons = true,
  randomizeTrigger = 0
}: AvatarCreatorProps) {
  const [config, setConfig] = useState<AvatarConfig>(
    initialConfig || {
      env: 0,
      clo: 0,
      head: 0,
      mouth: 0,
      eyes: 0,
      top: 0,
    }
  );
  const [colors, setColors] = useState<AvatarColors>(initialConfig?.colors || {});
  const [isLoading, setIsLoading] = useState(false);

  // Generate random config with random colors
  const randomize = useCallback(async () => {
    setIsLoading(true);
    // Generate random colors from palette
    const randomColors = getRandomColors();
    setColors(randomColors);

    try {
      const response = await fetch('/api/avatar/random');
      if (response.ok) {
        const data = await response.json();
        // Use server-generated parts but our random colors
        setConfig({
          env: data.env,
          clo: data.clo,
          head: data.head,
          mouth: data.mouth,
          eyes: data.eyes,
          top: data.top,
        });
      } else {
        // Fallback to client-side random
        setConfig({
          env: Math.floor(Math.random() * MAX_VALUE),
          clo: Math.floor(Math.random() * MAX_VALUE),
          head: Math.floor(Math.random() * MAX_VALUE),
          mouth: Math.floor(Math.random() * MAX_VALUE),
          eyes: Math.floor(Math.random() * MAX_VALUE),
          top: Math.floor(Math.random() * MAX_VALUE),
        });
      }
    } catch {
      // Fallback to client-side random
      setConfig({
        env: Math.floor(Math.random() * MAX_VALUE),
        clo: Math.floor(Math.random() * MAX_VALUE),
        head: Math.floor(Math.random() * MAX_VALUE),
        mouth: Math.floor(Math.random() * MAX_VALUE),
        eyes: Math.floor(Math.random() * MAX_VALUE),
        top: Math.floor(Math.random() * MAX_VALUE),
      });
    }
    setIsLoading(false);
  }, []);

  // Randomize on initial mount if no config provided
  useEffect(() => {
    if (!initialConfig) {
      randomize();
    }
  }, [initialConfig, randomize]);

  // Increment or decrement a part value
  const adjustPart = (part: PartKey, delta: number) => {
    setConfig(prev => ({
      ...prev,
      [part]: (prev[part] + delta + MAX_VALUE) % MAX_VALUE,
    }));
  };

  // Update a custom color
  const setPartColor = (part: PartKey, color: string | undefined) => {
    setColors(prev => {
      const updated = { ...prev };
      if (color) {
        updated[part] = color;
      } else {
        delete updated[part];
      }
      return updated;
    });
  };

  // Build avatar URL with current config and colors
  const buildAvatarUrl = () => {
    let url = `/api/avatar?env=${config.env}&clo=${config.clo}&head=${config.head}&mouth=${config.mouth}&eyes=${config.eyes}&top=${config.top}`;

    // Add color parameters if set
    if (colors.env) url += `&c_env=${encodeURIComponent(colors.env)}`;
    if (colors.clo) url += `&c_clo=${encodeURIComponent(colors.clo)}`;
    if (colors.head) url += `&c_head=${encodeURIComponent(colors.head)}`;
    if (colors.mouth) url += `&c_mouth=${encodeURIComponent(colors.mouth)}`;
    if (colors.eyes) url += `&c_eyes=${encodeURIComponent(colors.eyes)}`;
    if (colors.top) url += `&c_top=${encodeURIComponent(colors.top)}`;

    return url;
  };

  const avatarUrl = buildAvatarUrl();

  // Combine config with colors for saving
  const handleSave = () => {
    const hasColors = Object.keys(colors).length > 0;
    onSave({
      ...config,
      colors: hasColors ? colors : undefined,
    });
  };

  // Auto-save when config or colors change (for inline editing)
  useEffect(() => {
    if (!showButtons) {
      const hasColors = Object.keys(colors).length > 0;
      onSave({
        ...config,
        colors: hasColors ? colors : undefined,
      });
    }
  }, [config, colors, showButtons, onSave]);

  // Trigger randomize from parent
  useEffect(() => {
    if (randomizeTrigger > 0) {
      randomize();
    }
  }, [randomizeTrigger, randomize]);

  return (
    <div className="flex flex-col items-center gap-4">
      {/* Avatar Preview */}
      <div
        className="relative rounded-full overflow-hidden bg-gray-800 shadow-lg"
        style={{ width: size, height: size }}
      >
        {isLoading ? (
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="animate-spin w-8 h-8 border-2 border-purple-500 border-t-transparent rounded-full" />
          </div>
        ) : (
          <img
            key={avatarUrl}
            src={avatarUrl}
            alt="Avatar preview"
            className="w-full h-full"
            style={{ imageRendering: 'auto' }}
          />
        )}
      </div>

      {/* Part Controls */}
      <div className="w-full max-w-xs space-y-2">
        {PART_ORDER.map(part => (
          <div key={part} className="flex items-center justify-between gap-2">
            <span className="text-sm text-gray-400 w-20">{PART_LABELS[part]}</span>
            <div className="flex items-center gap-1">
              <button
                onClick={() => adjustPart(part, -1)}
                className="p-2 rounded-lg bg-gray-700 hover:bg-gray-600 active:bg-gray-500 transition-colors"
                aria-label={`Previous ${PART_LABELS[part]}`}
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                </svg>
              </button>
              <span className="w-12 text-center text-sm font-mono text-gray-300">
                {config[part] + 1}/{MAX_VALUE}
              </span>
              <button
                onClick={() => adjustPart(part, 1)}
                className="p-2 rounded-lg bg-gray-700 hover:bg-gray-600 active:bg-gray-500 transition-colors"
                aria-label={`Next ${PART_LABELS[part]}`}
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </button>
              {/* Color picker */}
              <ColorPicker
                value={colors[part]}
                onChange={(color) => setPartColor(part, color)}
                label={PART_LABELS[part]}
              />
            </div>
          </div>
        ))}
      </div>

      {/* Action Buttons */}
      {showButtons && (
        <div className="flex gap-3 mt-2">
          <button
            onClick={randomize}
            disabled={isLoading}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-gray-700 hover:bg-gray-600 active:bg-gray-500 transition-colors disabled:opacity-50"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            Random
          </button>
          {onCancel && (
            <button
              onClick={onCancel}
              className="px-4 py-2 rounded-lg bg-gray-700 hover:bg-gray-600 active:bg-gray-500 transition-colors"
            >
              Cancel
            </button>
          )}
          <button
            onClick={handleSave}
            className="px-4 py-2 rounded-lg bg-purple-600 hover:bg-purple-500 active:bg-purple-400 transition-colors font-medium"
          >
            Save
          </button>
        </div>
      )}
    </div>
  );
}

// Build avatar URL with optional colors
export function buildAvatarUrl(config: AvatarConfig): string {
  let url = `/api/avatar?env=${config.env}&clo=${config.clo}&head=${config.head}&mouth=${config.mouth}&eyes=${config.eyes}&top=${config.top}`;

  // Add color parameters if set
  if (config.colors) {
    const c = config.colors;
    if (c.env) url += `&c_env=${encodeURIComponent(c.env)}`;
    if (c.clo) url += `&c_clo=${encodeURIComponent(c.clo)}`;
    if (c.head) url += `&c_head=${encodeURIComponent(c.head)}`;
    if (c.mouth) url += `&c_mouth=${encodeURIComponent(c.mouth)}`;
    if (c.eyes) url += `&c_eyes=${encodeURIComponent(c.eyes)}`;
    if (c.top) url += `&c_top=${encodeURIComponent(c.top)}`;
  }

  return url;
}

// Simple avatar display component
export function Avatar({
  config,
  size = 48,
  className = ''
}: {
  config: AvatarConfig;
  size?: number;
  className?: string;
}) {
  const avatarUrl = buildAvatarUrl(config);

  return (
    <img
      src={avatarUrl}
      alt="Avatar"
      className={`rounded-full ${className}`}
      style={{ width: size, height: size }}
    />
  );
}

// Parse avatar config from JSON string
export function parseAvatarConfig(json: string | null | undefined): AvatarConfig | null {
  if (!json) return null;
  try {
    const parsed = JSON.parse(json);
    if (
      typeof parsed.env === 'number' &&
      typeof parsed.clo === 'number' &&
      typeof parsed.head === 'number' &&
      typeof parsed.mouth === 'number' &&
      typeof parsed.eyes === 'number' &&
      typeof parsed.top === 'number'
    ) {
      return parsed;
    }
  } catch {
    // Invalid JSON
  }
  return null;
}

// Convert config to JSON string
export function stringifyAvatarConfig(config: AvatarConfig): string {
  return JSON.stringify(config);
}
