// Extract multiavatar data and generate Go code
const fs = require('fs');

// Load the multiavatar source
const source = fs.readFileSync('./multiavatar-source.js', 'utf8');

const themes = {};
const sP = {};

// Extract themes - find section between "var themes = {" and closing "}"
const themesStart = source.indexOf('var themes = {');
const themesEnd = source.indexOf('\n  }', themesStart) + 4;
const themesSection = source.substring(themesStart, themesEnd);

// Parse each theme block
for (let i = 0; i <= 15; i++) {
  const themeId = i.toString().padStart(2, '0');
  themes[themeId] = { A: {}, B: {}, C: {} };

  // Find this theme's content
  const themeStart = themesSection.indexOf(`"${themeId}"`);
  if (themeStart === -1) continue;

  // Find next theme or end
  const nextThemeId = (i + 1).toString().padStart(2, '0');
  let themeEnd = themesSection.indexOf(`"${nextThemeId}"`, themeStart);
  if (themeEnd === -1) themeEnd = themesSection.length;

  const themeContent = themesSection.substring(themeStart, themeEnd);

  for (const variant of ['A', 'B', 'C']) {
    // Find variant block
    const variantStart = themeContent.indexOf(`"${variant}":`);
    if (variantStart === -1) continue;

    // Find the variant's closing brace
    let braceCount = 0;
    let variantEnd = variantStart;
    let inString = false;
    for (let j = variantStart; j < themeContent.length; j++) {
      const char = themeContent[j];
      if (char === '"' && themeContent[j-1] !== '\\') inString = !inString;
      if (!inString) {
        if (char === '{') braceCount++;
        if (char === '}') {
          braceCount--;
          if (braceCount === 0) {
            variantEnd = j + 1;
            break;
          }
        }
      }
    }

    const variantContent = themeContent.substring(variantStart, variantEnd);

    for (const part of ['env', 'clo', 'head', 'mouth', 'eyes', 'top']) {
      const partPattern = new RegExp(`"${part}":\\s*\\[([^\\]]*?)\\]`);
      const partMatch = partPattern.exec(variantContent);
      if (partMatch) {
        const arrayContent = partMatch[1];
        const colors = [];
        const colorPattern = /"([^"]+)"/g;
        let colorMatch;
        while ((colorMatch = colorPattern.exec(arrayContent)) !== null) {
          colors.push(colorMatch[1]);
        }
        themes[themeId][variant][part] = colors;
      }
    }
  }
}

// Initialize parts
for (let i = 0; i < 16; i++) {
  const id = i.toString().padStart(2, '0');
  sP[id] = { env: '', clo: '', head: '', mouth: '', eyes: '', top: '' };
}

// Common constants
const ENV_PATH = '<path d="M33.83,33.83a115.5,115.5,0,1,1,0,163.34,115.49,115.49,0,0,1,0-163.34Z" style="fill:#01;"/>';
const HEAD_PATH = '<path d="m115.5 51.75a63.75 63.75 0 0 0-10.5 126.63v14.09a115.5 115.5 0 0 0-53.729 19.027 115.5 115.5 0 0 0 128.46 0 115.5 115.5 0 0 0-53.729-19.029v-14.084a63.75 63.75 0 0 0 53.25-62.881 63.75 63.75 0 0 0-63.65-63.75 63.75 63.75 0 0 0-0.09961 0z" style="fill:#000;"/>';
const STR = 'stroke-linecap:round;stroke-linejoin:round;stroke-width:';

// Extract SVG parts using regex that handles the string concatenation pattern
// The format is: sP['XX']['part'] = 'svg content'+str+'more content';
const partsRegex = /sP\['(\d+)'\]\['(env|clo|head|mouth|eyes|top)'\]\s*=\s*(.*?);(?=\n)/g;

let partMatch;
while ((partMatch = partsRegex.exec(source)) !== null) {
  const [, id, partName, valueExpr] = partMatch;

  if (!sP[id]) sP[id] = {};

  let value = valueExpr.trim();

  // Handle special references
  if (value === 'env') {
    value = ENV_PATH;
  } else if (value === 'head') {
    value = HEAD_PATH;
  } else {
    // Process the string - it might contain str variable references
    // Pattern: 'string'+str+'more' or just 'string'

    // Replace str variable with actual value (handle all patterns)
    value = value.replace(/'\+str\+'/g, STR);
    value = value.replace(/\+str\+/g, STR);
    value = value.replace(/str\+'/g, STR);
    value = value.replace(/'\+str/g, STR);

    // Remove quotes and concatenation operators
    value = value.replace(/'\s*\+\s*'/g, '');

    // Remove leading/trailing quotes
    if (value.startsWith("'")) value = value.slice(1);
    if (value.endsWith("'")) value = value.slice(0, -1);
    if (value.endsWith(";")) value = value.slice(0, -1);
  }

  sP[id][partName] = value;
}

console.log('Themes extracted:', Object.keys(themes).length);
console.log('Parts extracted:', Object.keys(sP).length);

// Verify all parts have content
for (const id of Object.keys(sP)) {
  const parts = sP[id];
  for (const partName of ['env', 'clo', 'head', 'mouth', 'eyes', 'top']) {
    if (!parts[partName]) {
      console.log(`Missing: ${id}.${partName}`);
    }
  }
}

// Generate Go code for themes
let themesGo = `package avatar

// Theme color definitions - auto-generated from multiavatar.js
// Each theme has 3 color variants (A, B, C) with colors for each part

type PartColors struct {
\tEnv   []string
\tClo   []string
\tHead  []string
\tMouth []string
\tEyes  []string
\tTop   []string
}

type ThemeVariants struct {
\tA PartColors
\tB PartColors
\tC PartColors
}

var Themes = map[string]ThemeVariants{
`;

for (const themeId of Object.keys(themes).sort()) {
  const theme = themes[themeId];
  themesGo += `\t"${themeId}": {\n`;

  for (const variant of ['A', 'B', 'C']) {
    const v = theme[variant] || {};
    themesGo += `\t\t${variant}: PartColors{\n`;
    themesGo += `\t\t\tEnv:   []string{${(v.env || []).map(c => `"${c}"`).join(', ')}},\n`;
    themesGo += `\t\t\tClo:   []string{${(v.clo || []).map(c => `"${c}"`).join(', ')}},\n`;
    themesGo += `\t\t\tHead:  []string{${(v.head || []).map(c => `"${c}"`).join(', ')}},\n`;
    themesGo += `\t\t\tMouth: []string{${(v.mouth || []).map(c => `"${c}"`).join(', ')}},\n`;
    themesGo += `\t\t\tEyes:  []string{${(v.eyes || []).map(c => `"${c}"`).join(', ')}},\n`;
    themesGo += `\t\t\tTop:   []string{${(v.top || []).map(c => `"${c}"`).join(', ')}},\n`;
    themesGo += `\t\t},\n`;
  }

  themesGo += `\t},\n`;
}

themesGo += `}\n`;

fs.writeFileSync('./backend/internal/avatar/themes.go', themesGo);
console.log('Generated themes.go');

// Generate Go code for parts
let partsGo = `package avatar

// SVG part definitions - auto-generated from multiavatar.js
// Each theme (00-15) has 6 SVG parts

type PartSVGs struct {
\tEnv   string
\tClo   string
\tHead  string
\tMouth string
\tEyes  string
\tTop   string
}

// Common SVG elements
const (
\tSvgStart    = \`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 231 231">\`
\tSvgEnd      = \`</svg>\`
\tStrokeStyle = \`stroke-linecap:round;stroke-linejoin:round;stroke-width:\`
)

var Parts = map[string]PartSVGs{
`;

// Escape backticks in SVG strings for Go raw strings
function escapeForGoRawString(s) {
  // Go raw strings can't contain backticks, so we need to use string concatenation
  if (s.includes('`')) {
    // Split on backticks and rejoin with escaped version
    return '`' + s.replace(/`/g, '` + "`" + `') + '`';
  }
  return '`' + s + '`';
}

for (const id of Object.keys(sP).sort()) {
  const parts = sP[id];
  partsGo += `\t"${id}": {\n`;
  partsGo += `\t\tEnv:   ${escapeForGoRawString(parts.env || '')},\n`;
  partsGo += `\t\tClo:   ${escapeForGoRawString(parts.clo || '')},\n`;
  partsGo += `\t\tHead:  ${escapeForGoRawString(parts.head || '')},\n`;
  partsGo += `\t\tMouth: ${escapeForGoRawString(parts.mouth || '')},\n`;
  partsGo += `\t\tEyes:  ${escapeForGoRawString(parts.eyes || '')},\n`;
  partsGo += `\t\tTop:   ${escapeForGoRawString(parts.top || '')},\n`;
  partsGo += `\t},\n`;
}

partsGo += `}\n`;

fs.writeFileSync('./backend/internal/avatar/parts.go', partsGo);
console.log('Generated parts.go');

// Print sample
console.log('\nSample part 00 mouth:', sP['00'].mouth?.substring(0, 100));
