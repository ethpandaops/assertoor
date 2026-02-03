import { useEffect, useState, useCallback } from 'react';
import CodeMirror from '@uiw/react-codemirror';
import { yaml as yamlLang } from '@codemirror/lang-yaml';
import { githubLight, githubDark } from '@uiw/codemirror-theme-github';
import { useBuilderStore } from '../../../stores/builderStore';
import { validateYamlSyntax, formatYaml } from '../../../utils/builder/yamlSerializer';

// Hook to detect dark mode
function useDarkMode() {
  const [isDark, setIsDark] = useState(() =>
    document.documentElement.classList.contains('dark')
  );

  useEffect(() => {
    const observer = new MutationObserver(() => {
      setIsDark(document.documentElement.classList.contains('dark'));
    });
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] });
    return () => observer.disconnect();
  }, []);

  return isDark;
}

function BuilderYaml() {
  const yamlSource = useBuilderStore((state) => state.yamlSource);
  const setYamlSource = useBuilderStore((state) => state.setYamlSource);
  const syncFromYaml = useBuilderStore((state) => state.syncFromYaml);
  const isDarkMode = useDarkMode();

  const [localYaml, setLocalYaml] = useState(yamlSource);
  const [syntaxError, setSyntaxError] = useState<string | null>(null);
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);

  // Sync local state with store
  useEffect(() => {
    setLocalYaml(yamlSource);
    setHasUnsavedChanges(false);
  }, [yamlSource]);

  // Handle editor changes
  const handleChange = useCallback((value: string) => {
    setLocalYaml(value);
    setHasUnsavedChanges(value !== yamlSource);

    // Validate syntax
    const result = validateYamlSyntax(value);
    setSyntaxError(result.valid ? null : result.error || 'Invalid YAML');
  }, [yamlSource]);

  // Apply changes
  const handleApply = useCallback(() => {
    if (syntaxError) return;

    setYamlSource(localYaml);
    const success = syncFromYaml();
    if (success) {
      setHasUnsavedChanges(false);
    }
  }, [localYaml, syntaxError, setYamlSource, syncFromYaml]);

  // Discard changes
  const handleDiscard = useCallback(() => {
    setLocalYaml(yamlSource);
    setHasUnsavedChanges(false);
    setSyntaxError(null);
  }, [yamlSource]);

  // Format YAML
  const handleFormat = useCallback(() => {
    if (syntaxError) return;
    const formatted = formatYaml(localYaml);
    setLocalYaml(formatted);
    setHasUnsavedChanges(formatted !== yamlSource);
  }, [localYaml, yamlSource, syntaxError]);

  const cmTheme = isDarkMode ? githubDark : githubLight;

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-[var(--color-border)] bg-[var(--color-bg-secondary)]">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">YAML Editor</span>
          {hasUnsavedChanges && (
            <span className="text-xs px-1.5 py-0.5 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400 rounded">
              Modified
            </span>
          )}
          {syntaxError && (
            <span className="text-xs px-1.5 py-0.5 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded">
              Syntax Error
            </span>
          )}
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={handleFormat}
            disabled={!!syntaxError}
            className="px-2 py-1 text-xs bg-[var(--color-bg-tertiary)] hover:bg-[var(--color-bg-primary)] rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            title="Format YAML"
          >
            Format
          </button>
          {hasUnsavedChanges && (
            <>
              <button
                onClick={handleDiscard}
                className="px-2 py-1 text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] bg-[var(--color-bg-tertiary)] hover:bg-[var(--color-bg-primary)] rounded transition-colors"
              >
                Discard
              </button>
              <button
                onClick={handleApply}
                disabled={!!syntaxError}
                className="px-2 py-1 text-xs bg-primary-600 text-white rounded hover:bg-primary-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Apply Changes
              </button>
            </>
          )}
        </div>
      </div>

      {/* Error message */}
      {syntaxError && (
        <div className="px-3 py-2 bg-red-50 dark:bg-red-900/20 border-b border-red-200 dark:border-red-800">
          <div className="flex items-start gap-2">
            <ErrorIcon className="size-4 text-red-500 shrink-0 mt-0.5" />
            <p className="text-xs text-red-700 dark:text-red-400 font-mono">
              {syntaxError}
            </p>
          </div>
        </div>
      )}

      {/* Editor */}
      <div className="flex-1 overflow-hidden">
        <CodeMirror
          value={localYaml}
          height="100%"
          extensions={[yamlLang()]}
          theme={cmTheme}
          onChange={handleChange}
          placeholder="# Enter your test configuration in YAML format"
          basicSetup={{
            lineNumbers: true,
            foldGutter: true,
            highlightActiveLine: true,
            autocompletion: true,
            bracketMatching: true,
            indentOnInput: true,
          }}
          className="h-full text-sm"
        />
      </div>

      {/* Help text */}
      <div className="px-3 py-2 border-t border-[var(--color-border)] text-xs text-[var(--color-text-tertiary)] bg-[var(--color-bg-secondary)]">
        <div className="flex items-center justify-between">
          <span>Edit test configuration directly in YAML</span>
          <div className="flex items-center gap-3">
            <span><kbd className="px-1 bg-[var(--color-bg-tertiary)] rounded">Ctrl+Z</kbd> Undo</span>
            <span><kbd className="px-1 bg-[var(--color-bg-tertiary)] rounded">Ctrl+Y</kbd> Redo</span>
          </div>
        </div>
      </div>
    </div>
  );
}

function ErrorIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
      />
    </svg>
  );
}

export default BuilderYaml;
