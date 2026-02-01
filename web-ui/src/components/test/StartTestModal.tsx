import { useState, useEffect, useMemo, useCallback } from 'react';
import { useTests, useTestDetails, useScheduleTestRun } from '../../hooks/useApi';
import Modal from '../common/Modal';
import yaml from 'js-yaml';

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
  const [skipQueue, setSkipQueue] = useState(false);
  const [error, setError] = useState<string | null>(null);

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
      setSkipQueue(false);
      setError(null);
    }
  }, [isOpen, initialTestId]);

  // Initialize config when test details are loaded
  useEffect(() => {
    if (testDetails?.config) {
      const initialConfig: ConfigFormValues = {};
      for (const [key, value] of Object.entries(testDetails.config)) {
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
          config = formConfig as Record<string, unknown>;
        }
      }

      const result = await scheduleMutation.mutateAsync({
        test_id: selectedTestId,
        config: config && Object.keys(config).length > 0 ? config : undefined,
        allow_duplicate: allowDuplicate,
        skip_queue: skipQueue,
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

  const hasConfig = testDetails?.config && Object.keys(testDetails.config).length > 0;

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
                {testDetails?.timeout && testDetails.timeout > 0 && (
                  <div className="text-sm mt-1">
                    <span className="text-[var(--color-text-secondary)]">Default Timeout: </span>
                    <span>{formatTimeout(testDetails.timeout)}</span>
                  </div>
                )}
              </div>

              {/* Configuration section */}
              {hasConfig && (
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="text-sm font-medium">Configuration Variables</h3>
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
                      {testDetails?.config &&
                        Object.entries(testDetails.config).map(([key, defaultValue]) => (
                          <ConfigField
                            key={key}
                            name={key}
                            defaultValue={defaultValue as ConfigValue}
                            value={formConfig[key]}
                            onChange={(value) => updateFormValue(key, value)}
                          />
                        ))}
                    </div>
                  ) : (
                    <div>
                      <textarea
                        value={yamlConfig}
                        onChange={(e) => setYamlConfig(e.target.value)}
                        className="w-full h-48 px-3 py-2 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-sm font-mono text-sm focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent resize-none"
                        placeholder="# Enter configuration in YAML format"
                      />
                      <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
                        Edit configuration variables in YAML format
                      </p>
                    </div>
                  )}
                </div>
              )}

              {/* Options */}
              <div className="space-y-2">
                <h3 className="text-sm font-medium">Options</h3>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={allowDuplicate}
                    onChange={(e) => setAllowDuplicate(e.target.checked)}
                    className="rounded-xs border-[var(--color-border)]"
                  />
                  <span className="text-sm">Allow duplicate (run even if test is already running)</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={skipQueue}
                    onChange={(e) => setSkipQueue(e.target.checked)}
                    className="rounded-xs border-[var(--color-border)]"
                  />
                  <span className="text-sm">Skip queue (start immediately)</span>
                </label>
              </div>

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
}

function ConfigField({ name, defaultValue, value, onChange }: ConfigFieldProps) {
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
        return (
          <textarea
            value={typeof currentValue === 'string' ? currentValue : JSON.stringify(currentValue, null, 2)}
            onChange={(e) => {
              try {
                onChange(JSON.parse(e.target.value));
              } catch {
                // Keep as string if JSON parse fails
                onChange(e.target.value);
              }
            }}
            className="w-full h-24 px-3 py-1.5 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-sm text-sm font-mono focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent resize-none"
            placeholder={JSON.stringify(defaultValue, null, 2)}
          />
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
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

function formatTimeout(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return `${hours}h ${minutes}m`;
}

export default StartTestModal;
