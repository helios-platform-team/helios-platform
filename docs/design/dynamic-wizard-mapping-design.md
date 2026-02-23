# Detailed Design: Dynamic Wizard for Backstage Scaffolder

> **Author:** Team 3  
> **Date:** 2026-02-13  
> **Version:** 1.0  
> **Status:** Draft - Pending Review  
> **Sprint:** Sprint 3  

---

## Mục lục

1. [Bối cảnh & Vấn đề](#1-bối-cảnh--vấn-đề)
2. [Giải pháp đề xuất](#2-giải-pháp-đề-xuất)
3. [Kiến trúc tổng quan](#3-kiến-trúc-tổng-quan)
4. [Chi tiết thiết kế](#4-chi-tiết-thiết-kế)
5. [Luồng hoạt động](#5-luồng-hoạt-động)
6. [Hướng dẫn triển khai](#6-hướng-dẫn-triển-khai)
7. [Mở rộng trong tương lai](#7-mở-rộng-trong-tương-lai)
8. [Tham khảo](#8-tham-khảo)

---

## 1. Bối cảnh & Vấn đề

### 1.1 Backstage Scaffolder là gì?

**Backstage Software Templates** (Scaffolder) cho phép tạo các wizard form để developers tự tạo projects, services, hoặc resources. Form được định nghĩa bằng YAML schema và render bởi `react-jsonschema-form` (RJSF).

```mermaid
flowchart LR
    A[template.yaml] --> B[RJSF Parser]
    B --> C[Dynamic Form UI]
    C --> D[User Input]
    D --> E[Scaffolder Actions]
    E --> F[Generated Project]
```

### 1.2 Vấn đề hiện tại

Hiện tại, form trong Backstage template **không thể thay đổi động** dựa trên input của user:

**Ví dụ use case:**

| Scenario | Expected Behavior | Current State |
|----------|-------------------|---------------|
| User chọn "Postgres" | Hiện thêm field "DB Password" | ❌ Không thể |
| User chọn "MongoDB" | Hiện field "Connection String" | ❌ Không thể |
| User chọn "No Database" | Ẩn tất cả DB fields | ❌ Không thể |

**Current template limitation:**

```yaml
# template.yaml - Static fields only
parameters:
  - title: Database Configuration
    properties:
      databaseType:
        type: string
        enum: [postgres, mongodb, none]
      # Các field bên dưới LUÔN hiển thị, không thể ẩn/hiện động
      dbPassword:
        type: string
      connectionString:
        type: string
```

### 1.3 Mục tiêu Sprint 3

Implement **Dynamic Wizard** với khả năng:

- ✅ Hiện/ẩn fields dựa trên user selection
- ✅ Validate conditional fields
- ✅ Reusable custom field components
- ✅ Type-safe với TypeScript

---

## 2. Giải pháp đề xuất

### 2.1 React JSON Schema Form (RJSF)

**RJSF** là thư viện Backstage sử dụng để render forms từ JSON Schema. RJSF hỗ trợ **conditional rendering** thông qua:

#### 2.1.1 JSON Schema `dependencies`

```yaml
type: object
properties:
  databaseType:
    type: string
    enum: [postgres, mongodb, none]
dependencies:
  databaseType:
    oneOf:
      - properties:
          databaseType:
            const: postgres
          dbHost:
            type: string
            title: Database Host
          dbPort:
            type: number
            default: 5432
          dbPassword:
            type: string
            ui:widget: password
        required: [dbHost, dbPassword]
      - properties:
          databaseType:
            const: mongodb
          connectionString:
            type: string
            title: MongoDB Connection String
        required: [connectionString]
      - properties:
          databaseType:
            const: none
```

#### 2.1.2 JSON Schema `if/then/else`

```yaml
type: object
properties:
  databaseType:
    type: string
    enum: [postgres, mongodb, none]
allOf:
  - if:
      properties:
        databaseType:
          const: postgres
    then:
      properties:
        dbPassword:
          type: string
          title: PostgreSQL Password
      required: [dbPassword]
  - if:
      properties:
        databaseType:
          const: mongodb
    then:
      properties:
        connectionString:
          type: string
          title: MongoDB URI
      required: [connectionString]
```

### 2.2 Backstage Custom Field Extensions

Backstage cho phép tạo **Custom Field Extensions** để:

1. **Custom UI components** - Render field với UI riêng
2. **Custom validation** - Validate phức tạp hơn JSON Schema
3. **Async data fetching** - Load options từ API
4. **Inter-field communication** - Fields ảnh hưởng lẫn nhau

```mermaid
flowchart TD
    subgraph "Custom Field Extension"
        A[Field Component] --> B[Custom UI]
        A --> C[Validation Logic]
        A --> D[Form Context Access]
    end
    
    E[template.yaml] --> F["ui:field: MyCustomField"]
    F --> A
```

### 2.3 So sánh 2 approaches

| Criteria | JSON Schema Dependencies | Custom Field Extensions |
|----------|-------------------------|------------------------|
| **Complexity** | Low - YAML only | Medium - TypeScript code |
| **Flexibility** | Limited to schema rules | Full React capabilities |
| **Reusability** | Per-template | Across all templates |
| **Maintenance** | Easy | Requires code deployment |
| **Use case** | Simple show/hide | Complex interactions |

### 2.4 Recommended Approach

**Hybrid approach:**

1. **Simple conditional fields** → Use JSON Schema `dependencies` hoặc `if/then/else`
2. **Complex interactions** → Create Custom Field Extensions

---

## 3. Kiến trúc tổng quan

### 3.1 Component Architecture

```mermaid
flowchart TB
    subgraph Templates["Software Templates"]
        T1[template-postgres.yaml]
        T2[template-fullstack.yaml]
        T3[template-microservice.yaml]
    end
    
    subgraph CustomFields["Custom Field Extensions"]
        CF1[DatabasePicker]
        CF2[ConditionalGroup]
        CF3[DependentField]
    end
    
    subgraph Backstage["Backstage Portal"]
        SC[ScaffolderPlugin]
        RJSF[react-jsonschema-form]
    end
    
    Templates --> SC
    CustomFields --> SC
    SC --> RJSF
    RJSF --> UI[Dynamic Form UI]
```

### 3.2 File Structure

```
apps/portal/
├── packages/
│   └── app/
│       └── src/
│           └── scaffolder/
│               ├── index.ts                    # Export all extensions
│               ├── DatabasePickerExtension/
│               │   ├── DatabasePicker.tsx      # Component
│               │   ├── extension.ts            # Register extension
│               │   └── schema.ts               # Validation schema
│               └── ConditionalFieldExtension/
│                   ├── ConditionalField.tsx
│                   ├── extension.ts
│                   └── schema.ts
├── examples/
│   └── conditional-template/
│       └── template.yaml                       # Demo template
```

---

## 4. Chi tiết thiết kế

### 4.1 Custom Field Extension: DatabasePicker

**Mục đích:** Dropdown chọn database type, tự động hiện các fields liên quan.

```typescript
// DatabasePicker.tsx
import React from 'react';
import { FieldExtensionComponentProps } from '@backstage/plugin-scaffolder-react';
import { FormControl, InputLabel, Select, MenuItem, TextField } from '@material-ui/core';

export const DatabasePicker = ({
  onChange,
  rawErrors,
  required,
  formData,
  uiSchema,
  schema,
  formContext,
}: FieldExtensionComponentProps<string>) => {
  const [dbType, setDbType] = React.useState(formData?.type || '');
  
  const handleTypeChange = (event: React.ChangeEvent<{ value: unknown }>) => {
    const newType = event.target.value as string;
    setDbType(newType);
    onChange({
      type: newType,
      // Reset dependent fields when type changes
      ...(newType === 'postgres' && { host: '', port: 5432, password: '' }),
      ...(newType === 'mongodb' && { connectionString: '' }),
    });
  };

  return (
    <div>
      <FormControl fullWidth required={required}>
        <InputLabel>Database Type</InputLabel>
        <Select value={dbType} onChange={handleTypeChange}>
          <MenuItem value="none">No Database</MenuItem>
          <MenuItem value="postgres">PostgreSQL</MenuItem>
          <MenuItem value="mongodb">MongoDB</MenuItem>
        </Select>
      </FormControl>
      
      {dbType === 'postgres' && (
        <>
          <TextField
            label="Host"
            value={formData?.host || ''}
            onChange={(e) => onChange({ ...formData, host: e.target.value })}
            fullWidth
            margin="normal"
          />
          <TextField
            label="Port"
            type="number"
            value={formData?.port || 5432}
            onChange={(e) => onChange({ ...formData, port: parseInt(e.target.value) })}
            fullWidth
            margin="normal"
          />
          <TextField
            label="Password"
            type="password"
            value={formData?.password || ''}
            onChange={(e) => onChange({ ...formData, password: e.target.value })}
            fullWidth
            margin="normal"
            required
          />
        </>
      )}
      
      {dbType === 'mongodb' && (
        <TextField
          label="Connection String"
          value={formData?.connectionString || ''}
          onChange={(e) => onChange({ ...formData, connectionString: e.target.value })}
          fullWidth
          margin="normal"
          placeholder="mongodb://username:password@host:port/database"
          required
        />
      )}
    </div>
  );
};
```

### 4.2 Extension Registration

```typescript
// extension.ts
import { scaffolderPlugin } from '@backstage/plugin-scaffolder';
import { createScaffolderFieldExtension } from '@backstage/plugin-scaffolder-react';
import { DatabasePicker } from './DatabasePicker';

export const DatabasePickerExtension = scaffolderPlugin.provide(
  createScaffolderFieldExtension({
    name: 'DatabasePicker',
    component: DatabasePicker,
    validation: (value, validation, context) => {
      if (!value?.type || value.type === 'none') return;
      
      if (value.type === 'postgres') {
        if (!value.host) {
          validation.addError('Host is required for PostgreSQL');
        }
        if (!value.password) {
          validation.addError('Password is required for PostgreSQL');
        }
      }
      
      if (value.type === 'mongodb' && !value.connectionString) {
        validation.addError('Connection string is required for MongoDB');
      }
    },
  }),
);
```

### 4.3 App Integration

```typescript
// packages/app/src/App.tsx
import { DatabasePickerExtension } from './scaffolder';

const app = createApp({
  // ...
  components: {
    // ...
  },
});

// Register custom field extensions
const scaffolderFieldExtensions = [
  DatabasePickerExtension,
];

// In routes
<Route path="/create" element={<ScaffolderPage />}>
  {scaffolderFieldExtensions.map((Extension) => (
    <Extension key={Extension.name} />
  ))}
</Route>
```

### 4.4 Template Usage

```yaml
# template.yaml
apiVersion: scaffolder.backstage.io/v1beta3
kind: Template
metadata:
  name: create-service-with-database
  title: Create Service with Database
  description: Creates a new service with optional database configuration
spec:
  owner: team-platform
  type: service
  
  parameters:
    - title: Service Configuration
      required:
        - name
      properties:
        name:
          title: Service Name
          type: string
          pattern: '^[a-z0-9-]+$'
        
    - title: Database Configuration
      properties:
        database:
          title: Database Settings
          type: object
          ui:field: DatabasePicker
          
  steps:
    - id: fetch-base
      name: Fetch Base Template
      action: fetch:template
      input:
        url: ./skeleton
        values:
          name: ${{ parameters.name }}
          database: ${{ parameters.database }}
          
    - id: publish
      name: Publish to GitHub
      action: publish:github
      input:
        repoUrl: github.com?owner=my-org&repo=${{ parameters.name }}
```

---

## 5. Luồng hoạt động

### 5.1 User Flow

```mermaid
sequenceDiagram
    participant U as User
    participant F as Form UI
    participant CF as Custom Field
    participant V as Validator
    participant S as Scaffolder
    
    U->>F: Open template wizard
    F->>CF: Render DatabasePicker
    CF-->>F: Show dropdown only
    
    U->>CF: Select "PostgreSQL"
    CF->>CF: Show host, port, password fields
    CF-->>F: Update form state
    
    U->>CF: Fill in credentials
    CF->>V: Validate on blur
    V-->>CF: Validation result
    
    U->>F: Click "Create"
    F->>V: Full validation
    V-->>F: Pass
    F->>S: Execute template steps
    S-->>U: Success!
```

### 5.2 Data Flow

```mermaid
flowchart LR
    subgraph Input
        A[User Selection]
    end
    
    subgraph Processing
        B[Custom Field Component]
        C[Form State]
        D[JSON Schema Validation]
    end
    
    subgraph Output
        E[Template Parameters]
        F[Scaffolder Actions]
    end
    
    A --> B
    B --> C
    C --> D
    D --> E
    E --> F
```

---

## 6. Hướng dẫn triển khai

### 6.1 Prerequisites

```bash
# Ensure Backstage dependencies
cd apps/portal
yarn install

# Required packages (already in Backstage)
# @backstage/plugin-scaffolder
# @backstage/plugin-scaffolder-react
# @rjsf/core
# @rjsf/material-ui
```

### 6.2 Development Steps

1. **Create extension folder structure**
```bash
mkdir -p packages/app/src/scaffolder/DatabasePickerExtension
```

2. **Implement component** (`DatabasePicker.tsx`)

3. **Register extension** (`extension.ts`)

4. **Export from index** (`scaffolder/index.ts`)

5. **Add to App.tsx**

6. **Create demo template** (`examples/conditional-template/template.yaml`)

7. **Test locally**
```bash
yarn dev
# Navigate to /create and test the template
```

### 6.3 Testing Checklist

| Test Case | Expected Result |
|-----------|-----------------|
| Select "No Database" | No additional fields shown |
| Select "PostgreSQL" | Host, Port, Password fields appear |
| Select "MongoDB" | Connection String field appears |
| Switch from Postgres to MongoDB | Postgres fields hidden, MongoDB field shown |
| Submit without required field | Validation error shown |
| Submit with all fields | Template executes successfully |

---

## 7. Mở rộng trong tương lai

### 7.1 Additional Custom Fields

| Field Extension | Use Case |
|-----------------|----------|
| `CloudProviderPicker` | Show different options for AWS/GCP/Azure |
| `EnvironmentSelector` | Dynamic env-specific configurations |
| `FeatureFlagToggle` | Enable/disable feature-specific fields |
| `DependencyResolver` | Auto-select dependencies based on framework |

### 7.2 Advanced Features

1. **Async Options Loading** - Fetch database instances from cloud provider API
2. **Cross-step Dependencies** - Validate step 2 fields based on step 1 selections
3. **Template Inheritance** - Base templates with conditional extensions
4. **Preview Mode** - Show generated files before creation

---

## 8. Tham khảo

### 8.1 Documentation

- [react-jsonschema-form Documentation](https://rjsf-team.github.io/react-jsonschema-form/)
- [JSON Schema Conditional Subschemas](https://json-schema.org/understanding-json-schema/reference/conditionals.html)
- [Backstage Custom Field Extensions](https://backstage.io/docs/features/software-templates/writing-custom-field-extensions)
- [Backstage Scaffolder Plugin](https://backstage.io/docs/features/software-templates/)

### 8.2 Code References

- Backstage RJSF integration: `@backstage/plugin-scaffolder-react`
- Example extensions: [backstage/backstage repo](https://github.com/backstage/backstage/tree/master/plugins/scaffolder)

### 8.3 Related Issues

- Sprint 3 Epic: Dynamic Wizard Implementation
- Figma designs: [Link to designs]

---

## Appendix A: JSON Schema Conditional Examples

### A.1 Simple `if/then/else`

```yaml
type: object
properties:
  animal:
    enum: [Cat, Dog]
if:
  properties:
    animal:
      const: Cat
then:
  properties:
    meow:
      type: boolean
else:
  properties:
    bark:
      type: boolean
```

### A.2 Multiple Conditions with `allOf`

```yaml
type: object
properties:
  environment:
    enum: [dev, staging, prod]
allOf:
  - if:
      properties:
        environment:
          const: prod
    then:
      properties:
        approvalRequired:
          type: boolean
          default: true
        approver:
          type: string
      required: [approver]
```

### A.3 Nested Dependencies

```yaml
type: object
properties:
  enableAuth:
    type: boolean
dependencies:
  enableAuth:
    oneOf:
      - properties:
          enableAuth:
            const: true
          authProvider:
            type: string
            enum: [oauth2, saml, ldap]
        required: [authProvider]
        dependencies:
          authProvider:
            oneOf:
              - properties:
                  authProvider:
                    const: oauth2
                  clientId:
                    type: string
                  clientSecret:
                    type: string
                required: [clientId, clientSecret]
```

---

## Appendix B: Glossary

| Term | Definition |
|------|------------|
| **RJSF** | React JSON Schema Form - library for rendering forms from JSON Schema |
| **Scaffolder** | Backstage plugin for creating software from templates |
| **Field Extension** | Custom React component for rendering form fields |
| **JSON Schema** | Vocabulary for annotating and validating JSON documents |
| **Conditional Schema** | JSON Schema features for dynamic validation (`if/then/else`, `dependencies`) |
