import { useMemo } from 'react';
import StringField from './fields/StringField';
import NumberField from './fields/NumberField';
import BooleanField from './fields/BooleanField';
import ArrayField from './fields/ArrayField';
import ObjectField from './fields/ObjectField';
import ExpressionMapField from './fields/ExpressionMapField';
import DurationField from './fields/DurationField';
import VariableSelector from './VariableSelector';
import ExpressionInput from './ExpressionInput';

interface JSONSchema {
  type?: string | string[];
  properties?: Record<string, JSONSchema>;
  propertyOrder?: string[];
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
  requireGroup?: string;
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
export type StringFormat = 'text' | 'address' | 'hash' | 'bigint' | 'yaml' | 'multiline' | 'shell';

// Parse the schema to determine field types
function getFieldType(schema: JSONSchema): string {
  const type = Array.isArray(schema.type) ? schema.type[0] : schema.type;

  if (schema.enum) return 'enum';
  if (schema.format === 'duration') return 'duration';
  if (schema.format === 'expressionMap') return 'expressionMap';

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

// Determine string format based on explicit schema format annotations only.
function getStringFormat(_name: string, schema: JSONSchema): StringFormat {
  if (!schema.format) return 'text';

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
    case 'shell':
    case 'script':
      return 'shell';
    default:
      return 'text';
  }
}

function TaskConfigForm({
  schema,
  config,
  configVars,
  onConfigChange,
  onConfigVarChange,
  taskId,
}: TaskConfigFormProps) {
  // Parse schema properties, preserving struct field order via propertyOrder
  const properties = useMemo(() => {
    const parsed = schema as JSONSchema;
    if (!parsed.properties) return [];

    const required = new Set(parsed.required || []);

    // Use propertyOrder to maintain Go struct field order, fall back to Object.keys
    const orderedNames = parsed.propertyOrder || Object.keys(parsed.properties);

    return orderedNames
      .filter((name) => name in parsed.properties!)
      .map((name) => {
        const ps = parsed.properties![name] as JSONSchema;
        const fieldType = getFieldType(ps);
        return {
          name,
          schema: ps,
          required: required.has(name),
          requireGroup: ps.requireGroup,
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
              <div className="flex items-center gap-1.5">
                <label className="text-xs text-[var(--color-text-secondary)]">
                  {prop.schema.title || prop.name}
                </label>
                {prop.requireGroup && (
                  <RequireGroupBadge group={prop.requireGroup} />
                )}
              </div>
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
                taskId={taskId}
              />
            )}
          </div>
        );
      })}
    </div>
  );
}

// Badge colors for requirement groups (A, B, C, ...)
const groupColors: Record<string, string> = {
  A: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  B: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
  C: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
  D: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400',
};

function RequireGroupBadge({ group }: { group: string }) {
  // Parse "A" or "A.1" format
  const groupLetter = group.split('.')[0];
  const subGroup = group.includes('.') ? group.split('.')[1] : null;
  const colorClass = groupColors[groupLetter] || 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400';

  return (
    <span className={`inline-flex items-center px-1 py-0.5 text-[10px] font-medium rounded-xs ${colorClass}`} title={`Required (group ${group})`}>
      {subGroup ? `required ${groupLetter}.${subGroup}` : 'required'}
    </span>
  );
}

interface FieldRendererProps {
  fieldType: string;
  schema: JSONSchema;
  value: unknown;
  onChange: (value: unknown) => void;
  stringFormat?: StringFormat;
  taskId: string;
}

function FieldRenderer({ fieldType, schema, value, onChange, stringFormat, taskId }: FieldRendererProps) {
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

    case 'expressionMap':
      return (
        <ExpressionMapField
          value={value as Record<string, string> | undefined}
          taskId={taskId}
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
