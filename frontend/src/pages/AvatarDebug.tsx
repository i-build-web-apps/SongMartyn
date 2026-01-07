import { useState, useEffect } from 'react';

interface AvatarConfig {
  env: number;
  clo: number;
  head: number;
  mouth: number;
  eyes: number;
  top: number;
  colors?: {
    env?: string;
    clo?: string;
    head?: string;
    mouth?: string;
    eyes?: string;
    top?: string;
  };
}

interface AvatarDebugInfo {
  config: AvatarConfig;
  raw_svg: string;
  normalized_svg: string;
  preview: Record<string, string>;
}

// Generate random avatar config
function randomConfig(): AvatarConfig {
  const randomInt = (max: number) => Math.floor(Math.random() * max);
  const randomColor = () => {
    const colors = [
      '#FFDFC4', '#F0C08A', '#D2956B', '#A67449', '#6B4423',
      '#FF4444', '#FF9500', '#FFCC00', '#4CAF50', '#2196F3', '#9C27B0', '#E91E8C',
      '#FFFFFF', '#9E9E9E', '#424242', '#000000',
      '#00BCD4', '#FF6B9D', '#7C4DFF', '#00E676',
    ];
    return colors[randomInt(colors.length)];
  };

  return {
    env: randomInt(48),
    clo: randomInt(48),
    head: randomInt(48),
    mouth: randomInt(48),
    eyes: randomInt(48),
    top: randomInt(48),
    colors: {
      env: randomColor(),
      clo: randomColor(),
      head: randomColor(),
      mouth: randomColor(),
      eyes: randomColor(),
      top: randomColor(),
    },
  };
}

// Build query string for avatar
function buildAvatarQuery(config: AvatarConfig, extra?: Record<string, string>): string {
  const params = new URLSearchParams({
    env: config.env.toString(),
    clo: config.clo.toString(),
    head: config.head.toString(),
    mouth: config.mouth.toString(),
    eyes: config.eyes.toString(),
    top: config.top.toString(),
  });

  if (config.colors) {
    if (config.colors.env) params.set('c_env', config.colors.env);
    if (config.colors.clo) params.set('c_clo', config.colors.clo);
    if (config.colors.head) params.set('c_head', config.colors.head);
    if (config.colors.mouth) params.set('c_mouth', config.colors.mouth);
    if (config.colors.eyes) params.set('c_eyes', config.colors.eyes);
    if (config.colors.top) params.set('c_top', config.colors.top);
  }

  if (extra) {
    Object.entries(extra).forEach(([k, v]) => params.set(k, v));
  }

  return params.toString();
}

function AvatarComparison({ config, size = 256 }: { config: AvatarConfig; size?: number }) {
  const [debugInfo, setDebugInfo] = useState<AvatarDebugInfo | null>(null);
  const [showSvg, setShowSvg] = useState<'raw' | 'normalized' | null>(null);

  const query = buildAvatarQuery(config);
  const svgUrl = `/api/avatar?${query}`;
  const pngUrl = `/api/avatar/png?${query}&size=${size}`;

  useEffect(() => {
    fetch(`/api/avatar/debug?${query}`)
      .then(res => res.json())
      .then(setDebugInfo)
      .catch(console.error);
  }, [query]);

  return (
    <div className="bg-gray-800 rounded-xl p-4 space-y-4">
      {/* Config display */}
      <div className="text-xs text-gray-400 font-mono">
        env:{config.env} clo:{config.clo} head:{config.head} mouth:{config.mouth} eyes:{config.eyes} top:{config.top}
      </div>

      {/* Side by side comparison */}
      <div className="flex gap-4">
        {/* SVG */}
        <div className="flex-1">
          <div className="text-sm text-gray-400 mb-2">SVG (Browser Rendered)</div>
          <div className="bg-gray-900 rounded-lg p-2 flex items-center justify-center" style={{ minHeight: size }}>
            <img
              src={svgUrl}
              alt="SVG Avatar"
              style={{ width: size, height: size }}
              className="rounded"
            />
          </div>
        </div>

        {/* PNG */}
        <div className="flex-1">
          <div className="text-sm text-gray-400 mb-2">PNG (oksvg Rendered)</div>
          <div className="bg-gray-900 rounded-lg p-2 flex items-center justify-center" style={{ minHeight: size }}>
            <img
              src={pngUrl}
              alt="PNG Avatar"
              style={{ width: size, height: size }}
              className="rounded"
            />
          </div>
        </div>
      </div>

      {/* Debug buttons */}
      <div className="flex gap-2">
        <button
          onClick={() => setShowSvg(showSvg === 'raw' ? null : 'raw')}
          className={`px-3 py-1 rounded text-sm ${showSvg === 'raw' ? 'bg-blue-600' : 'bg-gray-700'} text-white`}
        >
          Raw SVG
        </button>
        <button
          onClick={() => setShowSvg(showSvg === 'normalized' ? null : 'normalized')}
          className={`px-3 py-1 rounded text-sm ${showSvg === 'normalized' ? 'bg-blue-600' : 'bg-gray-700'} text-white`}
        >
          Normalized SVG
        </button>
      </div>

      {/* SVG code display */}
      {showSvg && debugInfo && (
        <div className="bg-gray-900 rounded-lg p-3 overflow-x-auto">
          <pre className="text-xs text-green-400 whitespace-pre-wrap break-all">
            {showSvg === 'raw' ? debugInfo.raw_svg : debugInfo.normalized_svg}
          </pre>
        </div>
      )}

      {/* Design preview */}
      {debugInfo?.preview && (
        <div className="text-xs text-gray-500 font-mono">
          Designs: {Object.entries(debugInfo.preview).map(([k, v]) => `${k}:${v}`).join(' ')}
        </div>
      )}
    </div>
  );
}

export default function AvatarDebug() {
  const [avatars, setAvatars] = useState<AvatarConfig[]>([]);
  const [size, setSize] = useState(256);

  const generateAvatars = (count: number) => {
    setAvatars(Array.from({ length: count }, () => randomConfig()));
  };

  // Generate all 16 base designs (design 0 with variant A)
  const generateAllDesigns = () => {
    const configs: AvatarConfig[] = [];
    for (let design = 0; design < 16; design++) {
      configs.push({
        env: design,
        clo: design,
        head: design,
        mouth: design,
        eyes: design,
        top: design,
      });
    }
    setAvatars(configs);
  };

  // Test specific problematic designs
  const testStrokeDesigns = () => {
    // Designs that use strokes (from grep results)
    const strokedDesigns = [0, 1, 2, 3, 4, 5];
    const configs: AvatarConfig[] = strokedDesigns.map(design => ({
      env: design,
      clo: design,
      head: design,
      mouth: design,
      eyes: design,
      top: design,
    }));
    setAvatars(configs);
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white p-6">
      <div className="max-w-6xl mx-auto">
        <h1 className="text-3xl font-bold mb-6">Avatar Debug</h1>
        <p className="text-gray-400 mb-6">
          Compare SVG (browser-rendered) vs PNG (oksvg-rendered) to identify rendering differences.
        </p>

        {/* Controls */}
        <div className="flex flex-wrap gap-4 mb-6">
          <button
            onClick={() => generateAvatars(4)}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg font-medium"
          >
            Generate 4 Random
          </button>
          <button
            onClick={() => generateAvatars(8)}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg font-medium"
          >
            Generate 8 Random
          </button>
          <button
            onClick={generateAllDesigns}
            className="px-4 py-2 bg-purple-600 hover:bg-purple-700 rounded-lg font-medium"
          >
            All 16 Designs
          </button>
          <button
            onClick={testStrokeDesigns}
            className="px-4 py-2 bg-orange-600 hover:bg-orange-700 rounded-lg font-medium"
          >
            Stroke Designs
          </button>

          <div className="flex items-center gap-2 ml-auto">
            <label className="text-gray-400">Size:</label>
            <select
              value={size}
              onChange={(e) => setSize(Number(e.target.value))}
              className="bg-gray-800 rounded px-3 py-2 text-white"
            >
              <option value={64}>64px</option>
              <option value={128}>128px</option>
              <option value={256}>256px</option>
              <option value={512}>512px</option>
            </select>
          </div>
        </div>

        {/* Avatar grid */}
        {avatars.length === 0 ? (
          <div className="text-center text-gray-500 py-12">
            Click a button above to generate avatars for comparison
          </div>
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {avatars.map((config, i) => (
              <AvatarComparison key={i} config={config} size={size} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
