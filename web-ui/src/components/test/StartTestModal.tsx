import { useState, useEffect, useMemo, useCallback } from 'react';
import { useTests, useTestDetails, useScheduleTestRun } from '../../hooks/useApi';
import Modal from '../common/Modal';
import yaml from 'js-yaml';
import CodeMirror from '@uiw/react-codemirror';
import { yaml as yamlLang } from '@codemirror/lang-yaml';
import { githubLight, githubDark } from '@uiw/codemirror-theme-github';
import { useDarkMode } from '../../hooks/useDarkMode';
import QueuePicker from './QueuePicker';
import type { ScheduleQueueOption } from '../../types/api';

interface StartTestModalProps {
  isOpen: boolean;
  onClose: () => void;
  initialTestId?: string | null;
}

type ConfigValue = string | number | boolean | null | undefined | Record<string, unknown> | unknown[];

interface ConfigFormValues {
  [key: string]: ConfigValue;
}

function StartTestModal({ isOpen, onClose, initialTestId }: StartTestModalProps) {
  const [selectedTestId, setSelectedTestId] = useState<string>(initialTestId || '');
  const [step, setStep] = useState<'select' | 'configure'>(initialTestId ? 'configure' : 'select');
  const [editMode, setEditMode] = useState<'form' | 'yaml'>('form');
  const [formConfig, setFormConfig] = useState<ConfigFormValues>({});
  const [yamlConfig, setYamlConfig] = useState('');
  const [allowDuplicate, setAllowDuplicate] = useState(false);
  const [queue, setQueue] = useState<ScheduleQueueOption>({ mode: 'end' });
  const [error, setError] = useState<string | null>(null);
  const isDarkMode = useDarkMode();
  const cmTheme = isDarkMode ? githubDark : githubLight;

  const { data: tests, isLoading: testsLoading } = useTests();
  const { data: testDetails, isLoading: detailsLoading } = useTestDetails(selectedTestId, {
    enabled: !!selectedTestId,
  });
  const scheduleMutation = useScheduleTestRun();

  // Reset state when modal opens/closes
  useEffect(() => {
    if (isOpen) {
      if (initialTestId) {
        setSelectedTestId(initialTestId);
        setStep('configure');
      } else {
        setSelectedTestId('');
        setStep('select');
      }
      setEditMode('form');
      setFormConfig({});
      setYamlConfig('');
      setAllowDuplicate(false);
      setQueue({ mode: 'end' });
      setError(null);
    }
  }, [isOpen, initialTestId]);

  // Initialize config when test details are loaded
  // Use vars (which includes global vars merged in) when available, fall back to raw config
  useEffect(() => {
    const configSource = testDetails?.vars || testDetails?.config;
    if (configSource) {
      const initialConfig: ConfigFormValues = {};
      for (const [key, value] of Object.entries(configSource)) {
        initialConfig[key] = value as ConfigValue;
      }
      setFormConfig(initialConfig);
      setYamlConfig(yaml.dump(initialConfig));
    } else {
      setFormConfig({});
      setYamlConfig('');
    }
  }, [testDetails]);

  // Sync form config to yaml when switching modes
  const handleModeSwitch = useCallback((mode: 'form' | 'yaml') => {
    if (mode === 'yaml' && editMode === 'form') {
      // Switching from form to yaml - serialize current form values
      try {
        setYamlConfig(yaml.dump(formConfig));
      } catch {
        // Keep existing yaml if serialization fails
      }
    } else if (mode === 'form' && editMode === 'yaml') {
      // Switching from yaml to form - parse yaml
      try {
        const parsed = yaml.load(yamlConfig) as ConfigFormValues;
        if (parsed && typeof parsed === 'object') {
          setFormConfig(parsed);
        }
      } catch {
        setError('Invalid YAML syntax. Please fix before switching to form mode.');
        return;
      }
    }
    setError(null);
    setEditMode(mode);
  }, [editMode, formConfig, yamlConfig]);

  const handleTestSelect = useCallback(() => {
    if (selectedTestId) {
      setStep('configure');
    }
  }, [selectedTestId]);

  const handleBack = useCallback(() => {
    if (initialTestId) {
      // If started with a specific test, close instead of going back
      onClose();
    } else {
      setStep('select');
      setError(null);
    }
  }, [initialTestId, onClose]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    try {
      // Get config values based on current edit mode
      let config: Record<string, unknown> | undefined;

      if (editMode === 'yaml') {
        if (yamlConfig.trim()) {
          try {
            config = yaml.load(yamlConfig) as Record<string, unknown>;
          } catch (err) {
            setError(`Invalid YAML: ${err instanceof Error ? err.message : 'Parse error'}`);
            return;
          }
        }
      } else {
        if (Object.keys(formConfig).length > 0) {
          // Parse any YAML strings for object/array fields
          config = {};
          for (const [key, value] of Object.entries(formConfig)) {
            if (typeof value === 'string' && configSource && typeof configSource[key] === 'object') {
              // This was an object/array field stored as YAML string
              try {
                config[key] = yaml.load(value);
              } catch (err) {
                setError(`Invalid YAML in field "${key}": ${err instanceof Error ? err.message : 'Parse error'}`);
                return;
              }
            } else {
              config[key] = value;
            }
          }
        }
      }

      const result = await scheduleMutation.mutateAsync({
        test_id: selectedTestId,
        config: config && Object.keys(config).length > 0 ? config : undefined,
        allow_duplicate: allowDuplicate,
        // Send the structured `queue` field. The legacy `skip_queue`
        // boolean is still supported server-side but the modern
        // shape is preferred.
        queue,
      });

      // Navigate to the new test run
      window.location.href = `/run/${result.run_id}`;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to schedule test');
    }
  };

  const updateFormValue = useCallback((key: string, value: ConfigValue) => {
    setFormConfig((prev) => ({ ...prev, [key]: value }));
  }, []);

  const selectedTest = useMemo(() => {
    return tests?.find((t) => t.id === selectedTestId);
  }, [tests, selectedTestId]);

  // Use vars (with global vars merged) when available, fall back to raw config
  const configSource = testDetails?.vars || testDetails?.config;
  const hasConfig = configSource && Object.keys(configSource).length > 0;

  const handleClose = () => {
    onClose();
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      title={step === 'select' ? 'Start Test' : `Run: ${testDetails?.name || selectedTest?.name || selectedTestId}`}
      size="lg"
    >
      {step === 'select' ? (
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">Select a Test</label>
            {testsLoading ? (
              <div className="flex items-center justify-center h-20">
                <div className="animate-spin rounded-full size-6 border-b-2 border-primary-600"></div>
              </div>
            ) : (
              <select
                value={selectedTestId}
                onChange={(e) => setSelectedTestId(e.target.value)}
                className="w-full px-3 py-2 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-sm text-sm focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              >
                <option value="">Choose a test...</option>
                {tests?.map((test) => (
                  <option key={test.id} value={test.id}>
                    {test.name || test.id}
                  </option>
                ))}
              </select>
            )}
          </div>

          {selectedTestId && selectedTest && (
            <div className="bg-[var(--color-bg-secondary)] rounded-sm p-3">
              <div className="text-sm">
                <span className="text-[var(--color-text-secondary)]">Test ID: </span>
                <span className="font-mono">{selectedTestId}</span>
              </div>
              <div className="text-sm mt-1">
                <span className="text-[var(--color-text-secondary)]">Source: </span>
                <span className="px-1.5 py-0.5 bg-[var(--color-bg-tertiary)] rounded-xs text-xs">
                  {selectedTest.source}
                </span>
              </div>
            </div>
          )}

          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={handleClose} className="btn btn-secondary btn-sm">
              Cancel
            </button>
            <button
              type="button"
              onClick={handleTestSelect}
              disabled={!selectedTestId}
              className="btn btn-primary btn-sm disabled:opacity-50"
            >
              Next
            </button>
          </div>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-4">
          {detailsLoading ? (
            <div className="flex items-center justify-center h-32">
              <div className="animate-spin rounded-full size-8 border-b-2 border-primary-600"></div>
            </div>
          ) : (
            <>
              {/* Test info */}
              <div className="bg-[var(--color-bg-secondary)] rounded-sm p-3">
                <div className="text-sm">
                  <span className="text-[var(--color-text-secondary)]">Test ID: </span>
                  <span className="font-mono">{selectedTestId}</span>
                </div>
                {/* The `... && expr` form would render `0` when timeout
                    is exactly 0 — coerce to boolean to keep React happy. */}
                {!!(testDetails?.timeout && testDetails.timeout > 0) && (
                  <div className="text-sm mt-1">
                    <span className="text-[var(--color-text-secondary)]">Default Timeout: </span>
                    <span>{formatTimeout(testDetails.timeout ?? 0)}</span>
                  </div>
                )}
              </div>

              {/* Configuration section — the test's own knobs */}
              {hasConfig && (
                <section className="rounded-md border border-[var(--color-border)] bg-[var(--color-bg-primary)] p-3">
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <ConfigIcon className="size-4 text-[var(--color-text-tertiary)]" />
                      <h3 className="text-sm font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
                        Test configuration
                      </h3>
                    </div>
                    <div className="flex items-center gap-1 bg-[var(--color-bg-secondary)] rounded-sm p-0.5">
                      <button
                        type="button"
                        onClick={() => handleModeSwitch('form')}
                        className={`px-2 py-1 text-xs rounded-xs transition-colors ${
                          editMode === 'form'
                            ? 'bg-primary-600 text-white'
                            : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]'
                        }`}
                      >
                        Form
                      </button>
                      <button
                        type="button"
                        onClick={() => handleModeSwitch('yaml')}
                        className={`px-2 py-1 text-xs rounded-xs transition-colors ${
                          editMode === 'yaml'
                            ? 'bg-primary-600 text-white'
                            : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]'
                        }`}
                      >
                        YAML
                      </button>
                    </div>
                  </div>

                  {editMode === 'form' ? (
                    <div className="space-y-3">
                      {configSource &&
                        Object.entries(configSource).map(([key, defaultValue]) => (
                          <ConfigField
                            key={key}
                            name={key}
                            defaultValue={defaultValue as ConfigValue}
                            value={formConfig[key]}
                            onChange={(value) => updateFormValue(key, value)}
                            theme={cmTheme}
                          />
                        ))}
                    </div>
                  ) : (
                    <div>
                      <div className="border border-[var(--color-border)] rounded-sm overflow-hidden">
                        <CodeMirror
                          value={yamlConfig}
                          height="200px"
                          extensions={[yamlLang()]}
                          theme={cmTheme}
                          onChange={(value) => setYamlConfig(value)}
                          placeholder="# Enter configuration in YAML format"
                          basicSetup={{
                            lineNumbers: true,
                            foldGutter: true,
                            highlightActiveLine: true,
                          }}
                          className="text-sm"
                        />
                      </div>
                      <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
                        Edit configuration variables in YAML format
                      </p>
                    </div>
                  )}
                </section>
              )}

              {/* Run options — visually distinct from the per-test
                  configuration above, since these only affect how /
                  when the run lands on the runner, not what it does. */}
              <section className="rounded-md border border-[var(--color-border)] bg-[var(--color-bg-secondary)] p-3 space-y-3">
                <header className="flex items-center gap-2">
                  <SettingsIcon className="size-4 text-[var(--color-text-tertiary)]" />
                  <h3 className="text-sm font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
                    Run options
                  </h3>
                </header>

                <div>
                  <label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">
                    Queue placement
                  </label>
                  <QueuePicker value={queue} onChange={setQueue} />
                </div>

                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={allowDuplicate}
                    onChange={(e) => setAllowDuplicate(e.target.checked)}
                    className="rounded-xs border-[var(--color-border)]"
                  />
                  <span className="text-sm">
                    Allow duplicate
                    <span className="text-[var(--color-text-tertiary)]"> · run even if the test is already queued</span>
                  </span>
                </label>
              </section>

              {error && (
                <div className="p-3 bg-error-50 dark:bg-error-900/20 border border-error-200 dark:border-error-800 rounded-sm">
                  <p className="text-sm text-error-600">{error}</p>
                </div>
              )}

              <div className="flex justify-between pt-2">
                <button type="button" onClick={handleBack} className="btn btn-secondary btn-sm">
                  {initialTestId ? 'Cancel' : 'Back'}
                </button>
                <button
                  type="submit"
                  className="btn btn-primary btn-sm"
                  disabled={scheduleMutation.isPending}
                >
                  {scheduleMutation.isPending ? 'Starting...' : 'Start Test'}
                </button>
              </div>
            </>
          )}
        </form>
      )}
    </Modal>
  );
}

interface ConfigFieldProps {
  name: string;
  defaultValue: ConfigValue;
  value: ConfigValue;
  onChange: (value: ConfigValue) => void;
  theme: typeof githubLight | typeof githubDark;
}

function ConfigField({ name, defaultValue, value, onChange, theme }: ConfigFieldProps) {
  const valueType = getValueType(defaultValue);
  const currentValue = value ?? defaultValue;

  const renderInput = () => {
    switch (valueType) {
      case 'boolean':
        return (
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={Boolean(currentValue)}
              onChange={(e) => onChange(e.target.checked)}
              className="rounded-xs border-[var(--color-border)]"
            />
            <span className="text-sm text-[var(--color-text-secondary)]">
              {currentValue ? 'Enabled' : 'Disabled'}
            </span>
          </label>
        );

      case 'number':
        return (
          <input
            type="number"
            value={currentValue as number}
            onChange={(e) => onChange(e.target.valueAsNumber || 0)}
            className="w-full px-3 py-1.5 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-sm text-sm font-mono focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent"
            placeholder={String(defaultValue)}
          />
        );

      case 'object':
      case 'array':
        // Render as YAML using CodeMirror for proper editing experience
        // Store raw YAML string - parsing happens at form submission
        const yamlValue = typeof currentValue === 'string'
          ? currentValue
          : yaml.dump(currentValue, { indent: 2, lineWidth: -1 });
        return (
          <div className="border border-[var(--color-border)] rounded-sm overflow-hidden">
            <CodeMirror
              value={yamlValue}
              height="auto"
              minHeight="80px"
              maxHeight="200px"
              extensions={[yamlLang()]}
              theme={theme}
              onChange={(value) => {
                // Store raw YAML string, don't parse here
                onChange(value);
              }}
              basicSetup={{
                lineNumbers: false,
                foldGutter: false,
                highlightActiveLine: false,
              }}
              className="text-sm"
            />
          </div>
        );

      case 'string':
      default:
        // Check if it looks like a duration (e.g., "10s", "5m", "1h")
        const isDuration = typeof defaultValue === 'string' && /^\d+[smh]$/.test(defaultValue);
        if (isDuration) {
          return (
            <div className="flex items-center gap-2">
              <input
                type="text"
                value={String(currentValue || '')}
                onChange={(e) => onChange(e.target.value)}
                className="flex-1 px-3 py-1.5 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-sm text-sm font-mono focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                placeholder={String(defaultValue)}
              />
              <span className="text-xs text-[var(--color-text-tertiary)]">e.g., 10s, 5m, 1h</span>
            </div>
          );
        }

        return (
          <input
            type="text"
            value={String(currentValue || '')}
            onChange={(e) => onChange(e.target.value)}
            className="w-full px-3 py-1.5 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-sm text-sm font-mono focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent"
            placeholder={String(defaultValue)}
          />
        );
    }
  };

  return (
    <div>
      <label className="block text-xs text-[var(--color-text-secondary)] mb-1">{name}</label>
      {renderInput()}
      {valueType !== 'boolean' && (
        <p className="text-[10px] text-[var(--color-text-tertiary)] mt-0.5">
          Default: {formatDefaultValue(defaultValue)}
        </p>
      )}
    </div>
  );
}

function getValueType(value: ConfigValue): 'string' | 'number' | 'boolean' | 'object' | 'array' | 'null' {
  if (value === null || value === undefined) return 'null';
  if (typeof value === 'boolean') return 'boolean';
  if (typeof value === 'number') return 'number';
  if (Array.isArray(value)) return 'array';
  if (typeof value === 'object') return 'object';
  return 'string';
}

function formatDefaultValue(value: ConfigValue): string {
  if (value === null || value === undefined) return 'null';
  if (typeof value === 'object') return yaml.dump(value, { flowLevel: 1 }).trim();
  return String(value);
}

function formatTimeout(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return `${hours}h ${minutes}m`;
}

function ConfigIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2M9 12h6M9 16h6"
      />
    </svg>
  );
}

function SettingsIcon({ className }: { className?: string }) {
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

export default StartTestModal;
