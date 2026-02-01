import { Routes, Route } from 'react-router-dom';
import { AuthProvider } from './context/AuthContext';
import Layout from './components/common/Layout';
import Dashboard from './pages/Dashboard';
import TestRun from './pages/TestRun';
import Registry from './pages/Registry';
import TestPage from './pages/TestPage';
import Clients from './pages/Clients';

function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Dashboard />} />
          <Route path="run/:runId" element={<TestRun />} />
          <Route path="test/:testId" element={<TestPage />} />
          <Route path="registry" element={<Registry />} />
          <Route path="clients" element={<Clients />} />
        </Route>
      </Routes>
    </AuthProvider>
  );
}

export default App;
