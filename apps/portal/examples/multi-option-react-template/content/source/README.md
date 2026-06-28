# ${{ values.repoName }} (React Application)

This project is scaffolded from the Helios Multi-Option React Scaffolder Template.

## Tech Stack Overview

- **Framework**: React 19.2.7
- **Build Tool**: ${{ values.buildTool | capitalize }}
- **Styling**: ${{ values.styling | capitalize }}
- **State Management**: ${{ values.stateManagement | capitalize }}
- **Data Fetching**: {% if values.dataFetching == 'react-query' %}React Query (TanStack Query){% else %}Standard Fetch API{% endif %}

{% if values.backendComponent -%}
## Backend Integration

This application is wired to connect to the backend component: **${{ values.backendComponent }}**.
- During local development, the app defaults to fetching from `http://localhost:${{ values.backendPort or 3001 }}/health`. Ensure your backend component is running locally on port ${{ values.backendPort or 3001 }}.
{%- endif %}

## Getting Started

### Prerequisites

- Node.js >= 24.16.0
- npm >= 10

### Installation

Install the project dependencies:

```bash
npm install
```

### Local Development

Start the development server with hot-reloads:

```bash
npm run dev
```

The application will be available at:
- **Vite**: <http://localhost:5173>
- **Webpack**: <http://localhost:8080>

### Production Build

Compile and bundle the application for production:

```bash
npm run build
```

This generates optimized static files in the `dist` directory, ready to be served by the production web server (Nginx).

### Code Quality

Run ESLint to check for code issues:

```bash
npm run lint
```
