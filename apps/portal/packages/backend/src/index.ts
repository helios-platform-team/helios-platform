/*
 * Hi!
 *
 * Note that this is an EXAMPLE Backstage backend. Please check the README.
 *
 * Happy hacking!
 */

import { resolve } from 'path';
import * as dotenv from 'dotenv';
import { createBackend } from '@backstage/backend-defaults';

// Load env vars from root .env
dotenv.config({ path: resolve(__dirname, '../../../.env'), debug: true });

const backend = createBackend();

backend.add(import('@backstage/plugin-app-backend'));
backend.add(import('@backstage/plugin-proxy-backend'));

// scaffolder plugin
backend.add(import('@backstage/plugin-scaffolder-backend'));
backend.add(import('@backstage/plugin-scaffolder-backend-module-github'));
import { scaffolderModuleCustomActions } from './extensions/scaffolder';
backend.add(scaffolderModuleCustomActions);
import { customAuth } from './extensions/auth';

backend.add(import('@backstage/plugin-scaffolder-backend-module-notifications'),
);

// auth plugin
backend.add(import('@backstage/plugin-auth-backend'));
backend.add(customAuth);
backend.add(import('@backstage/plugin-auth-backend-module-guest-provider'));
// See https://backstage.io/docs/auth/guest/provider

// catalog plugin
backend.add(import('@backstage/plugin-catalog-backend'));
backend.add(
  import('@backstage/plugin-catalog-backend-module-scaffolder-entity-model'),
);

// GitHub Org Entity Provider
backend.add(import('@backstage/plugin-catalog-backend-module-github-org'));

// See https://backstage.io/docs/features/software-catalog/configuration#subscribing-to-catalog-errors
backend.add(import('@backstage/plugin-catalog-backend-module-logs'));

// permission plugin
backend.add(import('@backstage/plugin-permission-backend'));
// See https://backstage.io/docs/permissions/getting-started for how to create your own permission policy
backend.add(
  import('@backstage/plugin-permission-backend-module-allow-all-policy'),
);

// search plugin
backend.add(import('@backstage/plugin-search-backend'));

// search engine
// See https://backstage.io/docs/features/search/search-engines
backend.add(import('@backstage/plugin-search-backend-module-pg'));

// search collators
backend.add(import('@backstage/plugin-search-backend-module-catalog'));
backend.add(import('@backstage/plugin-search-backend-module-techdocs'));

// kubernetes plugin
backend.add(import('@backstage/plugin-kubernetes-backend'));

// helios plugin
backend.add(import('@helios/plugin-database-backend'));

// notifications and signals plugins
backend.add(import('@backstage/plugin-notifications-backend'));
backend.add(import('@backstage/plugin-signals-backend'));

// database info extension
backend.add(import('./extensions/database-router-simple'));

backend.start();
