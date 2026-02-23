/**
 * Custom Scaffolder Field Extensions
 * 
 * This module exports all custom field extensions for the Backstage Scaffolder.
 * 
 * Available Extensions:
 * - DatabasePicker: Dynamic database configuration field
 * 
 * Usage:
 * 1. Import in App.tsx
 * 2. Pass to ScaffolderPage as children
 * 3. Use ui:field in template.yaml
 */

export { DatabasePickerFieldExtension } from './DatabasePickerExtension/extension';
