import { useState, useEffect, useCallback } from 'react';
import CodeMirror from '@uiw/react-codemirror';
import { githubLight, githubDark } from '@uiw/codemirror-theme-github';
import { yaml as yamlLang } from '@codemirror/lang-yaml';
import { useDarkMode } from '../../../../hooks/useDarkMode';
import Modal from '../../../common/Modal';
import type { Extension } from '@codemirror/state';

export type EditorLanguage = 'plain' | 'yaml';

interface TextEditorModalProps {
  isOpen: boolean;
  onClose: () => void;
  value: string;
  onSave: (value: string) => void;
  title?: string;
  language?: EditorLanguage;
  placeholder?: string;
}

const languageExtensions: Record<EditorLanguage, Extension[]> = {
  plain: [],
  yaml: [yamlLang()],
};

function TextEditorModal({
  isOpen,
  onClose,
  value,
  onSave,
  title = 'Edit Text',
  language = 'plain',
  placeholder,
}: TextEditorModalProps) {
  const [localValue, setLocalValue] = useState(value);
  const isDarkMode = useDarkMode();
  const cmTheme = isDarkMode ? githubDark : githubLight;

  // Sync local value when modal opens
  useEffect(() => {
    if (isOpen) {
      setLocalValue(value);
    }
  }, [isOpen, value]);

  const handleSave = useCallback(() => {
    onSave(localValue);
    onClose();
  }, [localValue, onSave, onClose]);

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={title} size="xl">
      <div className="flex flex-col gap-3">
        <div className="border border-[var(--color-border)] rounded-sm overflow-hidden">
          <CodeMirror
            value={localValue}
            height="60vh"
            theme={cmTheme}
            extensions={languageExtensions[language]}
            onChange={setLocalValue}
            placeholder={placeholder}
            basicSetup={{
              lineNumbers: true,
              foldGutter: language === 'yaml',
              highlightActiveLine: true,
              bracketMatching: true,
              indentOnInput: true,
            }}
            className="text-sm"
          />
        </div>

        <p className="text-xs text-[var(--color-text-tertiary)]">
          Use <kbd className="px-1 py-0.5 bg-[var(--color-bg-tertiary)] rounded-xs text-[10px]">Tab</kbd> to indent.
          Press <kbd className="px-1 py-0.5 bg-[var(--color-bg-tertiary)] rounded-xs text-[10px]">Escape</kbd> to close without saving.
        </p>

        <div className="flex justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            className="px-3 py-1.5 text-sm bg-[var(--color-bg-tertiary)] hover:bg-[var(--color-bg-secondary)] rounded-sm transition-colors"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={handleSave}
            className="px-3 py-1.5 text-sm bg-primary-600 text-white rounded-sm hover:bg-primary-700 transition-colors"
          >
            Save
          </button>
        </div>
      </div>
    </Modal>
  );
}

export default TextEditorModal;
