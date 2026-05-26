import { Route, Routes } from 'react-router-dom';
import { SecretListPage } from './SecretListPage.tsx';
import { SecretEntriesPage } from '../SecretEntries';

export const SecretsManagementPage = () => {
  return (
    <Routes>
      <Route path="/" element={<SecretListPage />} />
      <Route path="/:id" element={<SecretEntriesPage />} />
    </Routes>
  );
};
