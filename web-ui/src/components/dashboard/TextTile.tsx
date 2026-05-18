import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { tileHeightStyle, type DashboardTile, type TextConfig } from './types';

interface TextTileProps {
  tile: DashboardTile;
  config: TextConfig;
}

// TextTile renders user-supplied markdown. Useful for headers,
// dividers, links to runbooks, etc. There is no title bar by default
// — the markdown is the title.
//
// The flex-column layout exists so that when `heightPx` is configured
// the body scrolls instead of pushing the tile off the dashboard.
export function TextTile({ tile, config }: TextTileProps) {
  const content = config.markdown?.trim();

  return (
    <div
      className="card h-full flex flex-col overflow-hidden"
      style={tileHeightStyle(config.heightPx)}
    >
      {tile.title && (
        <h2 className="px-4 pt-4 pb-2 text-base font-semibold not-prose flex-shrink-0">
          {tile.title}
        </h2>
      )}
      <div className="markdown-body text-sm flex-1 overflow-auto px-4 pb-4 pt-3">
        {content ? (
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
        ) : (
          <p className="text-xs text-[var(--color-text-tertiary)] italic">
            Empty markdown tile — edit it to add content.
          </p>
        )}
      </div>
    </div>
  );
}

export default TextTile;
