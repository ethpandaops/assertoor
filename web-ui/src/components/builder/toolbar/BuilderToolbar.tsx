import { useCallback, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useBuilderStore } from '../../../stores/builderStore';
import { useTaskDescriptors, useRegisterTest, useTests } from '../../../hooks/useApi';
import { useAuthContext } from '../../../context/AuthContext';
import type { TaskDescriptor } from '../../../types/api';

interface BuilderToolbarProps {
  showPalette: boolean;
  showConfig: boolean;
  showAI: boolean;
  onTogglePalette: () => void;
  onToggleConfig: () => void;
  onToggleAI: () => void;
}

function BuilderToolbar({
  showPalette,
  showConfig,
  showAI,
  onTogglePalette,
  onToggleConfig,
  onToggleAI,
}: BuilderToolbarProps) {
  const navigate = useNavigate();
  const { isLoggedIn } = useAuthContext();
  const testConfig = useBuilderStore((state) => state.testConfig);
  const setTestName = useBuilderStore((state) => state.setTestName);
  const setTestId = useBuilderStore((state) => state.setTestId);
  const isDirty = useBuilderStore((state) => state.isDirty);
  const exportYaml = useBuilderStore((state) => state.exportYaml);
  const validate = useBuilderStore((state) => state.validate);
  const reset = useBuilderStore((state) => state.reset);
  const loadFromYaml = useBuilderStore((state) => state.loadFromYaml);

  const { data: descriptors } = useTaskDescriptors();
  const { data: registryTests } = useTests();
  const registerMutation = useRegisterTest();

  const [showExportModal, setShowExportModal] = useState(false);
  const [showImportModal, setShowImportModal] = useState(false);
  const [showLoadModal, setShowLoadModal] = useState(false);
  const [importYaml, setImportYaml] = useState('');

  // Build descriptor map for validation
  const descriptorMap = new Map<string, TaskDescriptor>();
  if (descriptors) {
    for (const d of descriptors) {
      descriptorMap.set(d.name, d);
    }
  }

  // Handle export
  const handleExport = useCallback(() => {
    setShowExportModal(true);
  }, []);

  // Handle copy YAML
  const handleCopyYaml = useCallback(() => {
    const yaml = exportYaml();
    navigator.clipboard.writeText(yaml);
  }, [exportYaml]);

  // Handle download YAML
  const handleDownloadYaml = useCallback(() => {
    const yaml = exportYaml();
    const blob = new Blob([yaml], { type: 'text/yaml' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${testConfig.id || 'test'}.yaml`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, [exportYaml, testConfig.id]);

  // Handle import
  const handleImport = useCallback(() => {
    if (!importYaml.trim()) return;

    if (isDirty && !confirm('You have unsaved changes. Import will replace them. Continue?')) {
      return;
    }

    const success = loadFromYaml(importYaml);
    if (success) {
      setShowImportModal(false);
      setImportYaml('');
    }
  }, [importYaml, isDirty, loadFromYaml]);

  // Handle load from registry
  const handleLoadTest = useCallback((testId: string) => {
    if (isDirty && !confirm('You have unsaved changes. Loading will replace them. Continue?')) {
      return;
    }

    setShowLoadModal(false);
    navigate(`/builder?testId=${encodeURIComponent(testId)}`);
  }, [isDirty, navigate]);

  // Handle save/register
  const handleSave = useCallback(async () => {
    // Validate first
    const errors = validate(descriptorMap);
    if (errors.some((e) => e.severity === 'error')) {
      alert('Please fix validation errors before saving.');
      return;
    }

    const yaml = exportYaml();

    try {
      await registerMutation.mutateAsync(yaml);
      alert('Test saved successfully!');
    } catch (err) {
      alert(`Failed to save test: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  }, [validate, descriptorMap, exportYaml, registerMutation]);

  // Handle new test
  const handleNew = useCallback(() => {
    if (isDirty && !confirm('You have unsaved changes. Create new test?')) {
      return;
    }
    reset();
  }, [isDirty, reset]);

  return (
    <div className="flex items-center justify-between px-3 py-2 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-t-lg">
      {/* Left side - Test info */}
      <div className="flex items-center gap-4">
        {/* Test name */}
        <div className="flex items-center gap-2">
          <input
            type="text"
            value={testConfig.name}
            onChange={(e) => setTestName(e.target.value)}
            placeholder="Test Name"
            className="px-2 py-1 text-sm font-medium bg-transparent border-b border-transparent hover:border-[var(--color-border)] focus:border-primary-500 focus:outline-none transition-colors"
          />
          {isDirty && (
            <span className="px-1.5 py-0.5 text-xs bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400 rounded">
              Modified
            </span>
          )}
        </div>

        {/* Test ID */}
        <div className="flex items-center gap-1 text-xs text-[var(--color-text-secondary)]">
          <span>ID:</span>
          <input
            type="text"
            value={testConfig.id || ''}
            onChange={(e) => setTestId(e.target.value)}
            placeholder="(auto)"
            className="w-24 px-1 py-0.5 font-mono bg-transparent border border-transparent hover:border-[var(--color-border)] focus:border-primary-500 focus:outline-none rounded transition-colors"
          />
        </div>
      </div>

      {/* Right side - Actions */}
      <div className="flex items-center gap-2">
        {/* Toggle buttons */}
        <div className="flex items-center gap-1 border-r border-[var(--color-border)] pr-2">
          <button
            onClick={onTogglePalette}
            className={`p-1.5 rounded transition-colors ${
              showPalette
                ? 'bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300'
                : 'hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)]'
            }`}
            title={showPalette ? 'Hide palette' : 'Show palette'}
          >
            <PaletteIcon className="size-4" />
          </button>
          <button
            onClick={onToggleConfig}
            className={`p-1.5 rounded transition-colors ${
              showConfig
                ? 'bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300'
                : 'hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)]'
            }`}
            title={showConfig ? 'Hide config panel' : 'Show config panel'}
          >
            <ConfigIcon className="size-4" />
          </button>
          <button
            onClick={onToggleAI}
            className={`p-1.5 rounded transition-colors ${
              showAI
                ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                : 'hover:bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)]'
            }`}
            title={showAI ? 'Hide AI assistant' : 'Show AI assistant'}
          >
            <AIIcon className="size-4" />
          </button>
        </div>

        {/* File operations */}
        <button
          onClick={handleNew}
          className="px-2 py-1 text-xs hover:bg-[var(--color-bg-tertiary)] rounded transition-colors"
          title="New test"
        >
          New
        </button>
        <button
          onClick={() => setShowLoadModal(true)}
          className="px-2 py-1 text-xs hover:bg-[var(--color-bg-tertiary)] rounded transition-colors"
          title="Load from registry"
        >
          Load
        </button>
        <button
          onClick={() => setShowImportModal(true)}
          className="px-2 py-1 text-xs hover:bg-[var(--color-bg-tertiary)] rounded transition-colors"
          title="Import YAML"
        >
          Import
        </button>
        <button
          onClick={handleExport}
          className="px-2 py-1 text-xs hover:bg-[var(--color-bg-tertiary)] rounded transition-colors"
          title="Export YAML"
        >
          Export
        </button>

        {/* Save button */}
        {isLoggedIn && (
          <button
            onClick={handleSave}
            disabled={registerMutation.isPending}
            className="px-3 py-1 text-xs bg-primary-600 text-white rounded hover:bg-primary-700 transition-colors disabled:opacity-50"
          >
            {registerMutation.isPending ? 'Saving...' : 'Save Test'}
          </button>
        )}
      </div>

      {/* Export Modal */}
      {showExportModal && (
        <ExportModal
          yaml={exportYaml()}
          onClose={() => setShowExportModal(false)}
          onCopy={handleCopyYaml}
          onDownload={handleDownloadYaml}
        />
      )}

      {/* Import Modal */}
      {showImportModal && (
        <ImportModal
          value={importYaml}
          onChange={setImportYaml}
          onImport={handleImport}
          onClose={() => {
            setShowImportModal(false);
            setImportYaml('');
          }}
        />
      )}

      {/* Load from Registry Modal */}
      {showLoadModal && (
        <LoadModal
          tests={registryTests || []}
          onLoad={handleLoadTest}
          onClose={() => setShowLoadModal(false)}
        />
      )}
    </div>
  );
}

interface ExportModalProps {
  yaml: string;
  onClose: () => void;
  onCopy: () => void;
  onDownload: () => void;
}

function ExportModal({ yaml, onClose, onCopy, onDownload }: ExportModalProps) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-[var(--color-bg-primary)] rounded-lg shadow-xl w-full max-w-2xl max-h-[80vh] flex flex-col">
        <div className="flex items-center justify-between p-4 border-b border-[var(--color-border)]">
          <h3 className="text-lg font-semibold">Export Test YAML</h3>
          <button onClick={onClose} className="p-1 hover:bg-[var(--color-bg-tertiary)] rounded">
            <CloseIcon className="size-5" />
          </button>
        </div>
        <div className="flex-1 overflow-hidden p-4">
          <pre className="h-96 overflow-auto p-3 bg-[var(--color-bg-secondary)] rounded text-sm font-mono whitespace-pre-wrap">
            {yaml}
          </pre>
        </div>
        <div className="flex items-center justify-end gap-2 p-4 border-t border-[var(--color-border)]">
          <button
            onClick={onCopy}
            className="px-3 py-1.5 text-sm hover:bg-[var(--color-bg-tertiary)] rounded transition-colors"
          >
            Copy to Clipboard
          </button>
          <button
            onClick={onDownload}
            className="px-3 py-1.5 text-sm bg-primary-600 text-white rounded hover:bg-primary-700 transition-colors"
          >
            Download File
          </button>
        </div>
      </div>
    </div>
  );
}

interface ImportModalProps {
  value: string;
  onChange: (value: string) => void;
  onImport: () => void;
  onClose: () => void;
}

function ImportModal({ value, onChange, onImport, onClose }: ImportModalProps) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-[var(--color-bg-primary)] rounded-lg shadow-xl w-full max-w-2xl max-h-[80vh] flex flex-col">
        <div className="flex items-center justify-between p-4 border-b border-[var(--color-border)]">
          <h3 className="text-lg font-semibold">Import Test YAML</h3>
          <button onClick={onClose} className="p-1 hover:bg-[var(--color-bg-tertiary)] rounded">
            <CloseIcon className="size-5" />
          </button>
        </div>
        <div className="flex-1 overflow-hidden p-4">
          <textarea
            value={value}
            onChange={(e) => onChange(e.target.value)}
            placeholder="Paste your test YAML here..."
            className="w-full h-96 p-3 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded text-sm font-mono resize-none focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
        </div>
        <div className="flex items-center justify-end gap-2 p-4 border-t border-[var(--color-border)]">
          <button
            onClick={onClose}
            className="px-3 py-1.5 text-sm hover:bg-[var(--color-bg-tertiary)] rounded transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={onImport}
            disabled={!value.trim()}
            className="px-3 py-1.5 text-sm bg-primary-600 text-white rounded hover:bg-primary-700 transition-colors disabled:opacity-50"
          >
            Import
          </button>
        </div>
      </div>
    </div>
  );
}

interface LoadModalProps {
  tests: Array<{ id: string; name: string }>;
  onLoad: (testId: string) => void;
  onClose: () => void;
}

function LoadModal({ tests, onLoad, onClose }: LoadModalProps) {
  const [search, setSearch] = useState('');

  const filteredTests = tests.filter((test) => {
    const searchLower = search.toLowerCase();
    return (
      test.id.toLowerCase().includes(searchLower) ||
      test.name.toLowerCase().includes(searchLower)
    );
  });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-[var(--color-bg-primary)] rounded-lg shadow-xl w-full max-w-2xl max-h-[80vh] flex flex-col">
        <div className="flex items-center justify-between p-4 border-b border-[var(--color-border)]">
          <h3 className="text-lg font-semibold">Load Test from Registry</h3>
          <button onClick={onClose} className="p-1 hover:bg-[var(--color-bg-tertiary)] rounded">
            <CloseIcon className="size-5" />
          </button>
        </div>
        <div className="p-4 border-b border-[var(--color-border)]">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by name or ID..."
            className="w-full px-3 py-2 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
        </div>
        <div className="flex-1 overflow-auto p-4">
          {filteredTests.length === 0 ? (
            <div className="text-center text-[var(--color-text-secondary)] py-8">
              {tests.length === 0 ? 'No tests in registry' : 'No tests match your search'}
            </div>
          ) : (
            <div className="space-y-2">
              {filteredTests.map((test) => (
                <button
                  key={test.id}
                  onClick={() => onLoad(test.id)}
                  className="w-full p-3 text-left bg-[var(--color-bg-secondary)] hover:bg-[var(--color-bg-tertiary)] rounded border border-[var(--color-border)] transition-colors"
                >
                  <div className="font-medium">{test.name}</div>
                  <div className="text-xs text-[var(--color-text-secondary)] font-mono mt-1">
                    {test.id}
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>
        <div className="flex items-center justify-end gap-2 p-4 border-t border-[var(--color-border)]">
          <button
            onClick={onClose}
            className="px-3 py-1.5 text-sm hover:bg-[var(--color-bg-tertiary)] rounded transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}

// Icons
function PaletteIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M4 6h16M4 12h16M4 18h7"
      />
    </svg>
  );
}

function ConfigIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
      />
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
      />
    </svg>
  );
}

function CloseIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
    </svg>
  );
}

function AIIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
      />
    </svg>
  );
}

export default BuilderToolbar;
