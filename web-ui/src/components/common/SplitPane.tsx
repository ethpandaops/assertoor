import { useState, useRef, useCallback, useEffect } from 'react';

interface SplitPaneProps {
  left: React.ReactNode;
  right: React.ReactNode;
  defaultLeftWidth?: number; // percentage (0-100)
  minLeftWidth?: number; // percentage
  maxLeftWidth?: number; // percentage
  storageKey?: string;
  maxHeight?: string; // CSS max-height value (e.g., '95vh', '600px')
}

function SplitPane({
  left,
  right,
  defaultLeftWidth = 50,
  minLeftWidth = 20,
  maxLeftWidth = 80,
  storageKey,
  maxHeight,
}: SplitPaneProps) {
  const [leftWidth, setLeftWidth] = useState(() => {
    if (storageKey) {
      const saved = localStorage.getItem(`splitPane:${storageKey}`);
      if (saved) {
        const parsed = parseFloat(saved);
        if (!isNaN(parsed) && parsed >= minLeftWidth && parsed <= maxLeftWidth) {
          return parsed;
        }
      }
    }
    return defaultLeftWidth;
  });

  const [isDragging, setIsDragging] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsDragging(true);
  }, []);

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isDragging || !containerRef.current) return;

      const rect = containerRef.current.getBoundingClientRect();
      const newLeftWidth = ((e.clientX - rect.left) / rect.width) * 100;
      const clampedWidth = Math.min(Math.max(newLeftWidth, minLeftWidth), maxLeftWidth);
      setLeftWidth(clampedWidth);
    },
    [isDragging, minLeftWidth, maxLeftWidth]
  );

  const handleMouseUp = useCallback(() => {
    if (isDragging) {
      setIsDragging(false);
      if (storageKey) {
        localStorage.setItem(`splitPane:${storageKey}`, leftWidth.toString());
      }
    }
  }, [isDragging, leftWidth, storageKey]);

  useEffect(() => {
    if (isDragging) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    }

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
  }, [isDragging, handleMouseMove, handleMouseUp]);

  const heightStyle = maxHeight ? { maxHeight, height: maxHeight } : { minHeight: '400px' };

  return (
    <>
      {/* Mobile: Stacked layout */}
      <div className="lg:hidden space-y-4">
        <div className="max-h-[50vh] overflow-auto">{left}</div>
        <div className="max-h-[50vh] overflow-auto">{right}</div>
      </div>

      {/* Desktop: Split pane layout */}
      <div
        ref={containerRef}
        className="hidden lg:flex"
        style={heightStyle}
      >
        {/* Left panel */}
        <div
          className="overflow-auto"
          style={{ width: `${leftWidth}%`, flexShrink: 0 }}
        >
          {left}
        </div>

        {/* Divider */}
        <div
          className={`relative flex-shrink-0 w-1 cursor-col-resize group ${
            isDragging ? 'bg-primary-500' : 'bg-[var(--color-border)] hover:bg-primary-400'
          }`}
          onMouseDown={handleMouseDown}
        >
          {/* Drag handle indicator */}
          <div
            className={`absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-1 h-8 rounded-full transition-colors ${
              isDragging
                ? 'bg-primary-600'
                : 'bg-[var(--color-text-tertiary)] group-hover:bg-primary-500'
            }`}
          />
        </div>

        {/* Right panel */}
        <div className="flex-1 overflow-auto pl-1">
          {right}
        </div>
      </div>
    </>
  );
}

export default SplitPane;
