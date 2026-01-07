import { useState } from 'react';

// Help content definitions for complex admin features
export const HELP_CONTENT = {
  network: {
    title: 'Network Configuration',
    sections: [
      {
        heading: 'What is this?',
        content: `SongMartyn needs to know which network address to advertise to guests. When guests scan the QR code, they'll be directed to this address to join the karaoke session.`,
      },
      {
        heading: 'Choosing the Right Interface',
        content: `
- **Wi-Fi**: Best for most home setups. Guests connect to the same Wi-Fi network as the server.
- **Ethernet**: More reliable for wired setups. Use if your karaoke machine is connected via cable.
- **VPN**: Only use if guests are connecting through a VPN tunnel.

The interface marked "Active" with a green badge is currently connected and working.`,
      },
      {
        heading: 'Troubleshooting',
        content: `
- **Guests can't connect?** Make sure they're on the same network (Wi-Fi or LAN).
- **Multiple IPs shown?** Choose the one in your local network range (usually 192.168.x.x or 10.x.x.x).
- **Behind a router?** Port forwarding may be needed for external access (advanced).`,
      },
    ],
  },

  youtubeApi: {
    title: 'YouTube API Setup',
    sections: [
      {
        heading: 'What is this?',
        content: `The YouTube API allows guests to search for karaoke videos directly from YouTube. Without an API key, only your local library will be searchable.`,
      },
      {
        heading: 'Getting an API Key',
        content: `
1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project (or select existing)
3. Enable the **YouTube Data API v3**
4. Go to **Credentials** > **Create Credentials** > **API Key**
5. Copy the API key and paste it here

The free tier includes 10,000 quota units/day (about 100 searches).`,
        links: [
          { label: 'Google Cloud Console', url: 'https://console.cloud.google.com/' },
          { label: 'YouTube API Documentation', url: 'https://developers.google.com/youtube/v3/getting-started' },
        ],
      },
      {
        heading: 'Security Notes',
        content: `
- Consider restricting your API key to this server's IP address
- YouTube API keys are free but have daily quotas
- The key is stored locally in your .env file`,
      },
    ],
  },

  certificates: {
    title: 'TLS/HTTPS Certificates',
    sections: [
      {
        heading: 'Why HTTPS?',
        content: `Modern browsers require HTTPS for many features SongMartyn uses, including:
- WebSocket connections (secure real-time updates)
- Camera/microphone access (for future features)
- Service workers (offline support)

Without HTTPS, guests may see security warnings or features may not work.`,
      },
      {
        heading: 'Self-Signed Certificates',
        content: `SongMartyn automatically generates self-signed certificates on first run. These are stored in:

\`backend/certs/cert.pem\` and \`backend/certs/key.pem\`

Guests will see a browser warning the first time they connect - this is normal for self-signed certificates. They can click "Advanced" > "Proceed" to continue.`,
      },
      {
        heading: 'Using Real Certificates (Advanced)',
        content: `For production use without warnings, you can use certificates from:
- **Let's Encrypt** (free, requires domain name)
- **Your organization's CA**

Replace the files in \`backend/certs/\` with your certificate and key.`,
        links: [
          { label: "Let's Encrypt", url: 'https://letsencrypt.org/' },
          { label: 'Certbot (Easy Let\'s Encrypt)', url: 'https://certbot.eff.org/' },
        ],
      },
    ],
  },

  adminPin: {
    title: 'Admin PIN & Remote Access',
    sections: [
      {
        heading: 'How it Works',
        content: `The admin PIN controls who can access the admin panel remotely:

- **No PIN set**: Only localhost (this computer) can access admin features
- **PIN set**: Anyone with the PIN can access admin from any device on the network`,
      },
      {
        heading: 'Security Recommendations',
        content: `
- Use a strong PIN (at least 6 characters)
- Change the PIN if you suspect it's been shared
- Changing the PIN immediately logs out all remote admin sessions
- Local access (from this computer) never requires a PIN`,
      },
      {
        heading: 'When to Use Remote Admin',
        content: `Enable remote admin access when you need to:
- Control the karaoke system from your phone
- Let a co-host manage the queue
- Administer from a different room

Keep it disabled if you only manage from the server computer.`,
      },
    ],
  },

  videoPlayer: {
    title: 'Video Player (mpv)',
    sections: [
      {
        heading: 'What is mpv?',
        content: `mpv is a free, open-source media player that SongMartyn uses to play karaoke videos. It's fast, lightweight, and supports virtually all video formats including CDG+MP3 karaoke files.

mpv is developed by volunteers and is completely free to use - there's no cost or licensing to worry about.`,
      },
      {
        heading: 'Quick Setup',
        content: `Click the "Install mpv" button next to the Video Player Path field. This will:

- Detect your operating system
- Show you the exact command to install mpv
- Let you verify the installation worked

The setup wizard provides platform-specific instructions for macOS (Homebrew), Linux (apt/dnf/pacman), and Windows (Chocolatey or direct download).`,
      },
      {
        heading: 'Configuration',
        content: `The "Video Player Path" setting should be:
- \`mpv\` if mpv is in your system PATH (recommended)
- Full path like \`/usr/local/bin/mpv\` or \`C:\\Program Files\\mpv\\mpv.exe\` otherwise

After installing, use the "Install mpv" button to verify the installation. If it shows a green checkmark with the version, you're all set!`,
      },
      {
        heading: 'Troubleshooting',
        content: `
- **"mpv not found"**: Make sure mpv is installed and in your system PATH
- **No audio**: Check your system audio output settings
- **Video stuttering**: Try enabling hardware acceleration in your system settings
- **CDG files not working**: Ensure your mpv build includes CDG support (most package manager versions do)`,
        links: [
          { label: 'mpv Official Site', url: 'https://mpv.io/' },
          { label: 'mpv Installation Guide', url: 'https://mpv.io/installation/' },
        ],
      },
    ],
  },

  library: {
    title: 'Song Library Management',
    sections: [
      {
        heading: 'Adding Songs',
        content: `Point SongMartyn to folders containing your karaoke files. Supported formats include:
- Video: MP4, MKV, AVI, WebM
- Audio: MP3, FLAC, WAV, OGG

The scanner will extract metadata (title, artist) from filenames and ID3 tags.`,
      },
      {
        heading: 'Folder Structure',
        content: `For best results, organize your files like:
\`\`\`
/Karaoke/
  Artist - Song Title.mp4
  Another Artist - Another Song.mkv
\`\`\`

Or in artist folders:
\`\`\`
/Karaoke/
  Artist Name/
    Song Title.mp4
\`\`\``,
      },
      {
        heading: 'Rescanning',
        content: `Use "Rescan" when you've added new files to a folder. The scanner will:
- Find new files
- Update metadata for changed files
- Remove entries for deleted files`,
      },
    ],
  },
};

export type HelpTopic = keyof typeof HELP_CONTENT;

interface HelpModalProps {
  topic: HelpTopic;
  isOpen: boolean;
  onClose: () => void;
}

export function HelpModal({ topic, isOpen, onClose }: HelpModalProps) {
  const content = HELP_CONTENT[topic];

  if (!isOpen || !content) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/70" onClick={onClose} />
      <div className="relative bg-matte-gray rounded-2xl w-full max-w-2xl max-h-[85vh] overflow-hidden flex flex-col animate-slide-up">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-white/10">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-500/20 rounded-lg">
              <svg className="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <h2 className="text-xl font-bold text-white">{content.title}</h2>
          </div>
          <button
            onClick={onClose}
            className="p-2 text-gray-400 hover:text-white transition-colors rounded-lg hover:bg-white/5"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {content.sections.map((section, idx) => (
            <div key={idx}>
              <h3 className="text-lg font-semibold text-yellow-neon mb-3">{section.heading}</h3>
              <div className="text-gray-300 space-y-2 whitespace-pre-line leading-relaxed">
                {section.content.split('\n').map((line, lineIdx) => {
                  // Handle code blocks
                  if (line.trim().startsWith('```')) {
                    return null;
                  }
                  // Handle inline code
                  if (line.includes('`')) {
                    const parts = line.split(/`([^`]+)`/);
                    return (
                      <p key={lineIdx}>
                        {parts.map((part, partIdx) =>
                          partIdx % 2 === 1 ? (
                            <code key={partIdx} className="px-1.5 py-0.5 bg-matte-black rounded text-yellow-neon font-mono text-sm">
                              {part}
                            </code>
                          ) : (
                            <span key={partIdx}>{part}</span>
                          )
                        )}
                      </p>
                    );
                  }
                  // Handle bullet points
                  if (line.trim().startsWith('- **')) {
                    const match = line.match(/- \*\*([^*]+)\*\*:?\s*(.*)/);
                    if (match) {
                      return (
                        <p key={lineIdx} className="pl-4">
                          <span className="text-white font-medium">{match[1]}</span>
                          {match[2] && `: ${match[2]}`}
                        </p>
                      );
                    }
                  }
                  if (line.trim().startsWith('- ')) {
                    return (
                      <p key={lineIdx} className="pl-4">
                        <span className="text-gray-500 mr-2">•</span>
                        {line.trim().substring(2)}
                      </p>
                    );
                  }
                  // Handle numbered lists
                  if (/^\d+\./.test(line.trim())) {
                    return (
                      <p key={lineIdx} className="pl-4">{line.trim()}</p>
                    );
                  }
                  // Regular text
                  if (line.trim()) {
                    return <p key={lineIdx}>{line}</p>;
                  }
                  return null;
                })}
              </div>

              {/* Links */}
              {'links' in section && section.links && section.links.length > 0 && (
                <div className="mt-3 flex flex-wrap gap-2">
                  {section.links.map((link, linkIdx) => (
                    <a
                      key={linkIdx}
                      href={link.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-blue-500/20 text-blue-400 rounded-lg text-sm font-medium hover:bg-blue-500/30 transition-colors"
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                      </svg>
                      {link.label}
                    </a>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-white/10 bg-matte-black/30">
          <button
            onClick={onClose}
            className="w-full py-3 bg-yellow-neon text-indigo-deep font-semibold rounded-xl hover:scale-[1.02] transition-transform"
          >
            Got it
          </button>
        </div>
      </div>
    </div>
  );
}

// Help button component for consistent styling
interface HelpButtonProps {
  onClick: () => void;
  className?: string;
}

export function HelpButton({ onClick, className = '' }: HelpButtonProps) {
  return (
    <button
      onClick={onClick}
      className={`p-1.5 text-gray-400 hover:text-blue-400 transition-colors rounded-lg hover:bg-blue-500/10 ${className}`}
      title="Learn more"
    >
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    </button>
  );
}

// Hook for managing help modal state
export function useHelpModal() {
  const [activeHelp, setActiveHelp] = useState<HelpTopic | null>(null);

  return {
    activeHelp,
    openHelp: (topic: HelpTopic) => setActiveHelp(topic),
    closeHelp: () => setActiveHelp(null),
    isOpen: activeHelp !== null,
  };
}
