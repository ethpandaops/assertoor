import { useEffect, useRef, useState } from 'react';
import SwaggerUI from 'swagger-ui-react';
import 'swagger-ui-react/swagger-ui.css';
import { useTheme } from '../hooks/useTheme';

function ApiDocs() {
  const containerRef = useRef<HTMLDivElement>(null);
  const { theme } = useTheme();
  const [isLoaded, setIsLoaded] = useState(false);

  // Apply dark mode styling to swagger UI
  useEffect(() => {
    if (!containerRef.current) return;

    const applyTheme = () => {
      const swaggerContainer = containerRef.current;
      if (!swaggerContainer) return;

      if (theme === 'dark') {
        swaggerContainer.classList.add('swagger-dark');
      } else {
        swaggerContainer.classList.remove('swagger-dark');
      }
    };

    // Apply immediately and after a short delay (for dynamic content)
    applyTheme();
    const timeout = setTimeout(applyTheme, 100);

    return () => clearTimeout(timeout);
  }, [theme, isLoaded]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">API Documentation</h1>
          <p className="text-sm text-[var(--color-text-secondary)] mt-1">
            REST API reference for Assertoor
          </p>
        </div>
      </div>

      <div
        ref={containerRef}
        className="card overflow-hidden swagger-container"
      >
        <SwaggerUI
          url="/api/docs/doc.json"
          onComplete={() => setIsLoaded(true)}
          docExpansion="list"
          defaultModelsExpandDepth={1}
          displayRequestDuration={true}
          filter={true}
          showExtensions={true}
          showCommonExtensions={true}
          tryItOutEnabled={true}
        />
      </div>

      {/* Swagger UI Dark Mode Styles */}
      <style>{`
        /* Base styles */
        .swagger-container .swagger-ui {
          font-family: inherit;
        }

        .swagger-container .swagger-ui .info .title {
          font-family: inherit;
        }

        /* Hide the topbar */
        .swagger-container .swagger-ui .topbar {
          display: none;
        }

        /* Loading state */
        .swagger-container .swagger-ui .loading-container {
          padding: 40px;
        }

        /* ==================== DARK MODE ==================== */
        .swagger-dark .swagger-ui {
          background: transparent;
        }

        /* ---------- Typography ---------- */
        .swagger-dark .swagger-ui,
        .swagger-dark .swagger-ui .info .title,
        .swagger-dark .swagger-ui .info p,
        .swagger-dark .swagger-ui .info li,
        .swagger-dark .swagger-ui .info a,
        .swagger-dark .swagger-ui .info .base-url,
        .swagger-dark .swagger-ui .scheme-container,
        .swagger-dark .swagger-ui .scheme-container label,
        .swagger-dark .swagger-ui .opblock-tag,
        .swagger-dark .swagger-ui .opblock-tag small,
        .swagger-dark .swagger-ui .opblock .opblock-summary-description,
        .swagger-dark .swagger-ui .opblock .opblock-summary-operation-id,
        .swagger-dark .swagger-ui .opblock .opblock-summary-path,
        .swagger-dark .swagger-ui .opblock .opblock-summary-path__deprecated,
        .swagger-dark .swagger-ui .opblock-description-wrapper p,
        .swagger-dark .swagger-ui .opblock-description-wrapper h4,
        .swagger-dark .swagger-ui .opblock-external-docs-wrapper p,
        .swagger-dark .swagger-ui .opblock-title_normal,
        .swagger-dark .swagger-ui .opblock-title_normal p,
        .swagger-dark .swagger-ui .opblock-title_normal h4,
        .swagger-dark .swagger-ui .model-title,
        .swagger-dark .swagger-ui .model,
        .swagger-dark .swagger-ui .model-box,
        .swagger-dark .swagger-ui .model-box-control,
        .swagger-dark .swagger-ui .models-control,
        .swagger-dark .swagger-ui .model-toggle::after,
        .swagger-dark .swagger-ui table thead tr td,
        .swagger-dark .swagger-ui table thead tr th,
        .swagger-dark .swagger-ui table tbody tr td,
        .swagger-dark .swagger-ui .parameter__name,
        .swagger-dark .swagger-ui .parameter__type,
        .swagger-dark .swagger-ui .parameter__deprecated,
        .swagger-dark .swagger-ui .parameter__in,
        .swagger-dark .swagger-ui .parameter__extension,
        .swagger-dark .swagger-ui .prop-type,
        .swagger-dark .swagger-ui .prop-format,
        .swagger-dark .swagger-ui .response-col_status,
        .swagger-dark .swagger-ui .response-col_description,
        .swagger-dark .swagger-ui .response-col_links,
        .swagger-dark .swagger-ui .responses-inner h4,
        .swagger-dark .swagger-ui .responses-inner h5,
        .swagger-dark .swagger-ui .responses-header .col_header,
        .swagger-dark .swagger-ui .response-control-media-type__title,
        .swagger-dark .swagger-ui .response-control-media-type--accept-controller select,
        .swagger-dark .swagger-ui label,
        .swagger-dark .swagger-ui .btn,
        .swagger-dark .swagger-ui select,
        .swagger-dark .swagger-ui input,
        .swagger-dark .swagger-ui textarea,
        .swagger-dark .swagger-ui .filter-container .filter input,
        .swagger-dark .swagger-ui .tab li,
        .swagger-dark .swagger-ui .opblock-body pre span,
        .swagger-dark .swagger-ui .microlight,
        .swagger-dark .swagger-ui .markdown p,
        .swagger-dark .swagger-ui .markdown li,
        .swagger-dark .swagger-ui .markdown code,
        .swagger-dark .swagger-ui .renderedMarkdown p,
        .swagger-dark .swagger-ui .renderedMarkdown li,
        .swagger-dark .swagger-ui .renderedMarkdown code {
          color: var(--color-text-primary) !important;
        }

        /* Secondary text */
        .swagger-dark .swagger-ui .info .base-url,
        .swagger-dark .swagger-ui .opblock-tag small,
        .swagger-dark .swagger-ui .opblock .opblock-summary-operation-id,
        .swagger-dark .swagger-ui .parameter__in,
        .swagger-dark .swagger-ui .prop-format {
          color: var(--color-text-secondary) !important;
        }

        /* ---------- Backgrounds ---------- */
        .swagger-dark .swagger-ui .scheme-container {
          background: var(--color-bg-secondary);
          box-shadow: none;
        }

        .swagger-dark .swagger-ui .opblock-tag {
          border-bottom-color: var(--color-border);
        }

        .swagger-dark .swagger-ui .opblock-tag:hover {
          background: var(--color-bg-tertiary);
        }

        /* Operation blocks base */
        .swagger-dark .swagger-ui .opblock {
          background: var(--color-bg-secondary);
          border-color: var(--color-border);
          box-shadow: none;
        }

        /* GET operations - blue tint */
        .swagger-dark .swagger-ui .opblock.opblock-get {
          background: rgba(97, 175, 254, 0.1);
          border-color: rgba(97, 175, 254, 0.3);
        }
        .swagger-dark .swagger-ui .opblock.opblock-get .opblock-summary {
          border-color: rgba(97, 175, 254, 0.3);
        }

        /* POST operations - green tint */
        .swagger-dark .swagger-ui .opblock.opblock-post {
          background: rgba(73, 204, 144, 0.1);
          border-color: rgba(73, 204, 144, 0.3);
        }
        .swagger-dark .swagger-ui .opblock.opblock-post .opblock-summary {
          border-color: rgba(73, 204, 144, 0.3);
        }

        /* PUT operations - orange tint */
        .swagger-dark .swagger-ui .opblock.opblock-put {
          background: rgba(252, 161, 48, 0.1);
          border-color: rgba(252, 161, 48, 0.3);
        }
        .swagger-dark .swagger-ui .opblock.opblock-put .opblock-summary {
          border-color: rgba(252, 161, 48, 0.3);
        }

        /* DELETE operations - red tint */
        .swagger-dark .swagger-ui .opblock.opblock-delete {
          background: rgba(249, 62, 62, 0.1);
          border-color: rgba(249, 62, 62, 0.3);
        }
        .swagger-dark .swagger-ui .opblock.opblock-delete .opblock-summary {
          border-color: rgba(249, 62, 62, 0.3);
        }

        /* Operation block sections */
        .swagger-dark .swagger-ui .opblock .opblock-section-header {
          background: var(--color-bg-tertiary);
          box-shadow: none;
        }

        .swagger-dark .swagger-ui .opblock .opblock-section-header h4 {
          color: var(--color-text-primary);
        }

        .swagger-dark .swagger-ui .opblock-body {
          background: transparent;
        }

        /* Parameters table */
        .swagger-dark .swagger-ui .parameters-col_description {
          color: var(--color-text-primary);
        }

        .swagger-dark .swagger-ui .parameters-col_description input,
        .swagger-dark .swagger-ui .parameters-col_description select,
        .swagger-dark .swagger-ui .parameters-col_description textarea {
          background: var(--color-bg-tertiary);
          border-color: var(--color-border);
        }

        .swagger-dark .swagger-ui table.parameters > tbody > tr > td {
          border-bottom-color: var(--color-border);
        }

        /* Required parameter asterisk */
        .swagger-dark .swagger-ui .parameter__name.required::after {
          color: #f93e3e;
        }

        /* ---------- Code blocks ---------- */
        .swagger-dark .swagger-ui .opblock-body pre.microlight,
        .swagger-dark .swagger-ui .highlight-code > .microlight,
        .swagger-dark .swagger-ui pre.microlight {
          background: #1e1e2e !important;
          border-radius: 4px;
        }

        .swagger-dark .swagger-ui .highlight-code > .microlight code,
        .swagger-dark .swagger-ui .renderedMarkdown code,
        .swagger-dark .swagger-ui code {
          background: var(--color-bg-tertiary);
          color: var(--color-text-primary);
        }

        /* JSON syntax highlighting */
        .swagger-dark .swagger-ui .microlight .hljs-attr,
        .swagger-dark .swagger-ui .microlight .hljs-string {
          color: #a6e3a1 !important;
        }

        .swagger-dark .swagger-ui .microlight .hljs-number,
        .swagger-dark .swagger-ui .microlight .hljs-literal {
          color: #fab387 !important;
        }

        /* ---------- Models section ---------- */
        .swagger-dark .swagger-ui section.models {
          border-color: var(--color-border);
        }

        .swagger-dark .swagger-ui section.models h4 {
          color: var(--color-text-primary);
        }

        .swagger-dark .swagger-ui section.models .model-container {
          background: var(--color-bg-secondary);
          border-radius: 4px;
          margin: 0 0 8px 0;
        }

        .swagger-dark .swagger-ui section.models .model-container:hover {
          background: var(--color-bg-tertiary);
        }

        .swagger-dark .swagger-ui .model-box {
          background: var(--color-bg-tertiary);
        }

        .swagger-dark .swagger-ui .model {
          color: var(--color-text-primary);
        }

        .swagger-dark .swagger-ui .model .property.primitive {
          color: var(--color-text-secondary);
        }

        /* ---------- Responses ---------- */
        .swagger-dark .swagger-ui .responses-wrapper {
          background: transparent;
        }

        .swagger-dark .swagger-ui .responses-inner {
          background: transparent;
        }

        .swagger-dark .swagger-ui .response {
          background: transparent;
        }

        .swagger-dark .swagger-ui .responses-table {
          background: transparent;
        }

        .swagger-dark .swagger-ui td.response-col_status {
          color: var(--color-text-primary);
        }

        .swagger-dark .swagger-ui .response-col_description__inner p {
          color: var(--color-text-secondary);
        }

        /* ---------- Tabs ---------- */
        .swagger-dark .swagger-ui .tab li {
          color: var(--color-text-secondary);
        }

        .swagger-dark .swagger-ui .tab li.active {
          color: var(--color-text-primary);
        }

        .swagger-dark .swagger-ui .tab li:first-child::after {
          background: var(--color-border);
        }

        /* ---------- Buttons ---------- */
        .swagger-dark .swagger-ui .btn {
          border-color: var(--color-border);
          background: var(--color-bg-tertiary);
        }

        .swagger-dark .swagger-ui .btn:hover {
          background: var(--color-bg-secondary);
        }

        .swagger-dark .swagger-ui .btn.execute {
          background: #4990e2;
          border-color: #4990e2;
          color: white !important;
        }

        .swagger-dark .swagger-ui .btn.execute:hover {
          background: #357abd;
        }

        .swagger-dark .swagger-ui .btn.cancel {
          background: transparent;
          border-color: #f93e3e;
          color: #f93e3e !important;
        }

        .swagger-dark .swagger-ui .btn-group .btn {
          background: var(--color-bg-tertiary);
        }

        /* Try it out button */
        .swagger-dark .swagger-ui .try-out__btn {
          border-color: var(--color-border);
        }

        /* ---------- Inputs ---------- */
        .swagger-dark .swagger-ui input[type=text],
        .swagger-dark .swagger-ui input[type=password],
        .swagger-dark .swagger-ui input[type=search],
        .swagger-dark .swagger-ui input[type=email],
        .swagger-dark .swagger-ui input[type=file],
        .swagger-dark .swagger-ui textarea,
        .swagger-dark .swagger-ui select {
          background: var(--color-bg-tertiary);
          border: 1px solid var(--color-border);
          color: var(--color-text-primary);
        }

        .swagger-dark .swagger-ui input::placeholder,
        .swagger-dark .swagger-ui textarea::placeholder {
          color: var(--color-text-tertiary);
        }

        .swagger-dark .swagger-ui .filter-container .filter input {
          background: var(--color-bg-tertiary);
          border-color: var(--color-border);
        }

        /* ---------- SVG Icons ---------- */
        .swagger-dark .swagger-ui svg.arrow,
        .swagger-dark .swagger-ui button svg,
        .swagger-dark .swagger-ui .model-toggle::after,
        .swagger-dark .swagger-ui .expand-operation svg {
          fill: var(--color-text-primary);
        }

        .swagger-dark .swagger-ui .expand-methods svg,
        .swagger-dark .swagger-ui .models-control svg {
          fill: var(--color-text-secondary);
        }

        /* Lock icon */
        .swagger-dark .swagger-ui .authorization__btn svg {
          fill: var(--color-text-secondary);
        }

        .swagger-dark .swagger-ui .authorization__btn.locked svg {
          fill: #49cc90;
        }

        /* Copy button */
        .swagger-dark .swagger-ui .copy-to-clipboard {
          background: var(--color-bg-tertiary);
        }

        .swagger-dark .swagger-ui .copy-to-clipboard button {
          background: transparent;
        }

        /* ---------- Links ---------- */
        .swagger-dark .swagger-ui a {
          color: #89b4fa;
        }

        .swagger-dark .swagger-ui a:hover {
          color: #b4befe;
        }

        /* ---------- Scrollbar ---------- */
        .swagger-dark .swagger-ui ::-webkit-scrollbar {
          width: 8px;
          height: 8px;
        }

        .swagger-dark .swagger-ui ::-webkit-scrollbar-track {
          background: var(--color-bg-tertiary);
        }

        .swagger-dark .swagger-ui ::-webkit-scrollbar-thumb {
          background: var(--color-border);
          border-radius: 4px;
        }

        .swagger-dark .swagger-ui ::-webkit-scrollbar-thumb:hover {
          background: var(--color-text-tertiary);
        }

        /* ---------- Dialog / Modal ---------- */
        .swagger-dark .swagger-ui .dialog-ux .modal-ux {
          background: var(--color-bg-primary);
          border-color: var(--color-border);
        }

        .swagger-dark .swagger-ui .dialog-ux .modal-ux-header {
          border-color: var(--color-border);
        }

        .swagger-dark .swagger-ui .dialog-ux .modal-ux-header h3 {
          color: var(--color-text-primary);
        }

        .swagger-dark .swagger-ui .dialog-ux .modal-ux-content {
          color: var(--color-text-primary);
        }

        .swagger-dark .swagger-ui .dialog-ux .modal-ux-content p {
          color: var(--color-text-secondary);
        }

        /* ---------- Loading ---------- */
        .swagger-dark .swagger-ui .loading-container .loading::before {
          border-color: var(--color-border);
          border-top-color: var(--color-text-primary);
        }
      `}</style>
    </div>
  );
}

export default ApiDocs;
