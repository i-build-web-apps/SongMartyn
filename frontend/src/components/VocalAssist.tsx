import type { VocalAssistLevel } from '../types';
import { VOCAL_LABELS } from '../types';
import { useRoomStore, selectVocalAssist } from '../stores/roomStore';
import { wsService } from '../services/websocket';

const LEVELS: VocalAssistLevel[] = ['OFF', 'LOW', 'MED', 'HIGH'];

export function VocalAssist() {
  const currentLevel = useRoomStore(selectVocalAssist);

  const handleLevelChange = (level: VocalAssistLevel) => {
    wsService.setVocalAssist(level);
    useRoomStore.getState().setVocalAssist(level);
  };

  return (
    <div className="bg-matte-gray rounded-2xl p-4">
      <h3 className="text-sm text-gray-400 mb-3 uppercase tracking-wide">
        Vocal Assist
      </h3>

      {/* Segmented Control */}
      <div className="flex bg-matte-black rounded-xl p-1">
        {LEVELS.map((level) => (
          <button
            key={level}
            onClick={() => handleLevelChange(level)}
            className={`
              flex-1 py-3 px-4 rounded-lg text-sm font-medium
              transition-all duration-200 ease-out
              ${
                currentLevel === level
                  ? 'bg-yellow-neon text-indigo-deep shadow-lg'
                  : 'text-gray-400 hover:text-white'
              }
            `}
          >
            {VOCAL_LABELS[level]}
          </button>
        ))}
      </div>

      {/* Level indicator */}
      <div className="mt-3 flex items-center gap-2">
        <div className="flex-1 h-1 bg-matte-black rounded-full overflow-hidden">
          <div
            className="h-full bg-yellow-neon transition-all duration-300"
            style={{
              width: `${
                currentLevel === 'OFF'
                  ? 0
                  : currentLevel === 'LOW'
                  ? 33
                  : currentLevel === 'MED'
                  ? 66
                  : 100
              }%`,
            }}
          />
        </div>
        <span className="text-xs text-gray-500 w-12 text-right">
          {currentLevel === 'OFF'
            ? '0%'
            : currentLevel === 'LOW'
            ? '15%'
            : currentLevel === 'MED'
            ? '45%'
            : '80%'}
        </span>
      </div>
    </div>
  );
}
