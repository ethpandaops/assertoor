import Modal from '../common/Modal';
import { tileTypeLabel, type TileType } from './types';

interface AddTileModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSelect: (type: TileType) => void;
}

interface Card {
  type: TileType;
  description: string;
  icon: string;
}

const CARDS: Card[] = [
  {
    type: 'success_rate',
    description:
      'Aggregate the last N runs of a test into a success-rate ring with per-run swatches. Best for compact, at-a-glance reliability dashboards.',
    icon: '◐',
  },
  {
    type: 'latest_result',
    description:
      'Render the markdown blob produced by the latest run of a test (the value of $ASSERTOOR_TEST_RESULT). Perfect for compatibility matrices, scoreboards, and other rich outputs.',
    icon: '📄',
  },
  {
    type: 'recent_runs',
    description:
      'Compact list of recent test runs across all tests or filtered to one. Live-updates while runs are in flight.',
    icon: '⟳',
  },
  {
    type: 'text',
    description:
      'A free-form markdown tile for headings, dividers, links to runbooks, etc.',
    icon: '✎',
  },
];

// AddTileModal lets the user pick a tile type to add. Configuration of
// the new tile happens immediately afterwards via the same TileEditor
// modal used for existing tiles.
export function AddTileModal({ isOpen, onClose, onSelect }: AddTileModalProps) {
  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Add a dashboard tile" size="lg">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        {CARDS.map((card) => (
          <button
            key={card.type}
            type="button"
            onClick={() => {
              onSelect(card.type);
              onClose();
            }}
            className="text-left p-4 border border-[var(--color-border)] rounded-md hover:border-primary-500 hover:bg-[var(--color-bg-tertiary)] transition-colors"
          >
            <div className="flex items-center gap-2 mb-2">
              <span className="text-xl leading-none">{card.icon}</span>
              <span className="font-medium">{tileTypeLabel(card.type)}</span>
            </div>
            <p className="text-xs text-[var(--color-text-secondary)]">
              {card.description}
            </p>
          </button>
        ))}
      </div>
    </Modal>
  );
}

export default AddTileModal;
