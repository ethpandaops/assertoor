import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import type { DashboardTile, TextConfig } from './types';

interface TextTileProps {
  tile: DashboardTile;
  config: TextConfig;
}

// TextTile renders user-supplied markdown. Useful for headers,
// dividers, links to runbooks, etc. There is no title bar by default
// — the markdown is the title.
export function TextTile({ tile, config }: TextTileProps) {
  const content = config.markdown?.trim();

  return (
    <div className="card p-4 h-full markdown-body text-sm">
      {tile.title && (
        <h2 className="text-base font-semibold mb-2 not-prose">{tile.title}</h2>
      )}
      {content ? (
        <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
      ) : (
        <p className="text-xs text-[var(--color-text-tertiary)] italic">
          Empty markdown tile — edit it to add content.
        </p>
      )}
    </div>
  );
}

export default TextTile;
