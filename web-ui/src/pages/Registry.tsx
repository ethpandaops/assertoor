import { useState } from 'react';
import MyTestsTab from '../components/library/MyTestsTab';
import LibraryTab from '../components/library/LibraryTab';

const ACTIVE_TAB_KEY = 'library:activeTab';
type ActiveTab = 'local' | 'shared';

function readActiveTab(): ActiveTab {
  if (typeof window === 'undefined') return 'local';
  const stored = window.localStorage.getItem(ACTIVE_TAB_KEY);
  return stored === 'shared' ? 'shared' : 'local';
}

function Registry() {
  const [activeTab, setActiveTab] = useState<ActiveTab>(readActiveTab);

  const handleTabChange = (tab: ActiveTab) => {
    setActiveTab(tab);
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(ACTIVE_TAB_KEY, tab);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Playbook Library</h1>
      </div>

      <div className="flex border-b border-[var(--color-border)]">
        <TabButton active={activeTab === 'local'} onClick={() => handleTabChange('local')}>
          Local Playbooks
        </TabButton>
        <TabButton active={activeTab === 'shared'} onClick={() => handleTabChange('shared')}>
          Shared Playbooks
        </TabButton>
      </div>

      {activeTab === 'local' ? <MyTestsTab /> : <LibraryTab />}
    </div>
  );
}

interface TabButtonProps {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}

function TabButton({ active, onClick, children }: TabButtonProps) {
  return (
    <button
      onClick={onClick}
      className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors -mb-px ${
        active
          ? 'border-primary-600 text-primary-600'
          : 'border-transparent text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]'
      }`}
    >
      {children}
    </button>
  );
}

export default Registry;
