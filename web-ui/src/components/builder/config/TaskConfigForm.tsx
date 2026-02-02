import { useMemo } from 'react';
import StringField from './fields/StringField';
import NumberField from './fields/NumberField';
import BooleanField from './fields/BooleanField';
import ArrayField from './fields/ArrayField';
import ObjectField from './fields/ObjectField';
import DurationField from './fields/DurationField';
import VariableSelector from './VariableSelector';
import ExpressionInput from './ExpressionInput';

interface JSONSchema {
  type?: string | string[];
  properties?: Record<string, JSONSchema>;
  required?: string[];
  items?: JSONSchema;
  default?: unknown;
  description?: string;
  enum?: unknown[];
  format?: string;
  title?: string;
  minimum?: number;
  maximum?: number;
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  additionalProperties?: boolean | JSONSchema;
}

interface TaskConfigFormProps {
  schema: Record<string, unknown>;
  config: Record<string, unknown>;
  configVars: Record<string, string>;
  onConfigChange: (key: string, value: unknown) => void;
  onConfigVarChange: (key: string, value: string) => void;
  taskId: string;
}

// String format types for different rendering
export type StringFormat = 'text' | 'address' | 'hash' | 'bigint' | 'yaml' | 'multiline';

// Parse the schema to determine field types
function getFieldType(schema: JSONSchema): string {
  const type = Array.isArray(schema.type) ? schema.type[0] : schema.type;

  if (schema.enum) return 'enum';
  if (schema.format === 'duration') return 'duration';

  switch (type) {
    case 'string':
      if (schema.format === 'duration') return 'duration';
      return 'string';
    case 'number':
    case 'integer':
      return 'number';
    case 'boolean':
      return 'boolean';
    case 'array':
      return 'array';
    case 'object':
      return 'object';
    default:
      return 'string';
  }
}

// Check if a field looks like a duration based on name or pattern
function isDurationField(name: string, schema: JSONSchema): boolean {
  if (schema.format === 'duration') return true;
  const durationNames = ['timeout', 'duration', 'interval', 'delay', 'period'];
  return durationNames.some((d) => name.toLowerCase().includes(d));
}

// Determine string format based on schema format or field name patterns
function getStringFormat(name: string, schema: JSONSchema): StringFormat {
  // Explicit format annotations
  if (schema.format) {
    switch (schema.format.toLowerCase()) {
      case 'address':
      case 'eth-address':
        return 'address';
      case 'hash':
      case 'bytes32':
      case 'hex':
        return 'hash';
      case 'bigint':
      case 'uint256':
      case 'int256':
        return 'bigint';
      case 'yaml':
      case 'json':
      case 'multiline':
        return 'multiline';
    }
  }

  // Infer from field name patterns
  const lowerName = name.toLowerCase();

  // Address patterns
  if (lowerName.includes('address') || lowerName.includes('recipient') || lowerName.includes('sender')) {
    return 'address';
  }

  // Hash/key patterns
  if (lowerName.includes('hash') || lowerName.includes('pubkey') || lowerName.includes('privatekey') ||
      lowerName.includes('secret') || lowerName.includes('signature') || lowerName.includes('root') ||
      lowerName.includes('blockroot') || lowerName.includes('stateroot')) {
    return 'hash';
  }

  // Big integer patterns (values that exceed JS number precision)
  if (lowerName.includes('wei') || lowerName.includes('gwei') || lowerName.includes('amount') ||
      lowerName.includes('balance') || lowerName.includes('value') || lowerName.includes('gasPrice') ||
      lowerName.includes('gaslimit') || lowerName.includes('maxfee') || lowerName.includes('tip')) {
    return 'bigint';
  }

  // Multiline patterns
  if (lowerName.includes('script') || lowerName.includes('yaml') || lowerName.includes('json') ||
      lowerName.includes('config') || lowerName.includes('template') || lowerName.includes('body') ||
      lowerName.includes('payload') || lowerName.includes('data') || lowerName.includes('calldata')) {
    return 'multiline';
  }

  return 'text';
}

function TaskConfigForm({
  schema,
  config,
  configVars,
  onConfigChange,
  onConfigVarChange,
  taskId,
}: TaskConfigFormProps) {
  // Parse schema properties
  const properties = useMemo(() => {
    const parsed = schema as JSONSchema;
    if (!parsed.properties) return [];

    const required = new Set(parsed.required || []);

    return Object.entries(parsed.properties).map(([name, propSchema]) => {
      const ps = propSchema as JSONSchema;
      const fieldType = getFieldType(ps);
      return {
        name,
        schema: ps,
        required: required.has(name),
        fieldType,
        isDuration: isDurationField(name, ps),
        stringFormat: fieldType === 'string' ? getStringFormat(name, ps) : undefined,
      };
    });
  }, [schema]);

  if (properties.length === 0) {
    return (
      <p className="text-sm text-[var(--color-text-tertiary)]">
        No configuration options for this task.
      </p>
    );
  }

  return (
    <div className="space-y-4">
      {properties.map((prop) => {
        const value = config[prop.name];
        const varValue = configVars[prop.name];
        const hasVar = !!varValue;

        return (
          <div key={prop.name} className="space-y-1">
            {/* Field label */}
            <div className="flex items-center justify-between">
              <label className="text-xs text-[var(--color-text-secondary)]">
                {prop.schema.title || prop.name}
                {prop.required && <span className="text-red-500 ml-1">*</span>}
              </label>
              <VariableSelector
                taskId={taskId}
                varValue={varValue}
                onVarChange={(v) => onConfigVarChange(prop.name, v)}
              />
            </div>

            {/* Field description */}
            {prop.schema.description && (
              <p className="text-xs text-[var(--color-text-tertiary)]">
                {prop.schema.description}
              </p>
            )}

            {/* Variable expression input (when using var) */}
            {hasVar ? (
              <ExpressionInput
                taskId={taskId}
                value={varValue}
                onChange={(v) => onConfigVarChange(prop.name, v)}
              />
            ) : (
              /* Render appropriate field component */
              <FieldRenderer
                fieldType={prop.isDuration ? 'duration' : prop.fieldType}
                schema={prop.schema}
                value={value}
                onChange={(v) => onConfigChange(prop.name, v)}
                stringFormat={prop.stringFormat}
              />
            )}
          </div>
        );
      })}
    </div>
  );
}

interface FieldRendererProps {
  fieldType: string;
  schema: JSONSchema;
  value: unknown;
  onChange: (value: unknown) => void;
  stringFormat?: StringFormat;
}

function FieldRenderer({ fieldType, schema, value, onChange, stringFormat }: FieldRendererProps) {
  switch (fieldType) {
    case 'boolean':
      return (
        <BooleanField
          value={value as boolean | undefined}
          defaultValue={schema.default as boolean | undefined}
          onChange={onChange}
        />
      );

    case 'number':
      return (
        <NumberField
          value={value as number | undefined}
          defaultValue={schema.default as number | undefined}
          min={schema.minimum}
          max={schema.maximum}
          onChange={onChange}
        />
      );

    case 'duration':
      return (
        <DurationField
          value={value as string | undefined}
          defaultValue={schema.default as string | undefined}
          onChange={onChange}
        />
      );

    case 'enum':
      return (
        <select
          value={value as string || ''}
          onChange={(e) => onChange(e.target.value || undefined)}
          className="w-full px-2 py-1.5 text-sm bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-sm focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent"
        >
          <option value="">Select...</option>
          {schema.enum?.map((opt) => (
            <option key={String(opt)} value={String(opt)}>
              {String(opt)}
            </option>
          ))}
        </select>
      );

    case 'array':
      return (
        <ArrayField
          value={value as unknown[] | undefined}
          itemSchema={schema.items as Record<string, unknown> | undefined}
          onChange={onChange}
        />
      );

    case 'object':
      return (
        <ObjectField
          value={value as Record<string, unknown> | undefined}
          schema={schema as unknown as Record<string, unknown>}
          onChange={onChange}
        />
      );

    case 'string':
    default:
      return (
        <StringField
          value={value as string | undefined}
          defaultValue={schema.default as string | undefined}
          placeholder={schema.description}
          pattern={schema.pattern}
          minLength={schema.minLength}
          maxLength={schema.maxLength}
          format={stringFormat}
          onChange={onChange}
        />
      );
  }
}

export default TaskConfigForm;
