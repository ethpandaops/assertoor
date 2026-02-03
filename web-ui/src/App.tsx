import { lazy, Suspense } from 'react';
import { Routes, Route } from 'react-router-dom';
import { AuthProvider } from './context/AuthContext';
import Layout from './components/common/Layout';

// Lazy load page components for code splitting
const Dashboard = lazy(() => import(/* webpackChunkName: "page-dashboard" */ './pages/Dashboard'));
const TestRun = lazy(() => import(/* webpackChunkName: "page-testrun" */ './pages/TestRun'));
const Registry = lazy(() => import(/* webpackChunkName: "page-registry" */ './pages/Registry'));
const TestPage = lazy(() => import(/* webpackChunkName: "page-test" */ './pages/TestPage'));
const Clients = lazy(() => import(/* webpackChunkName: "page-clients" */ './pages/Clients'));
const TestBuilder = lazy(() => import(/* webpackChunkName: "page-builder" */ './pages/TestBuilder'));
const ApiDocs = lazy(() => import(/* webpackChunkName: "page-api-docs" */ './pages/ApiDocs'));

function PageLoader() {
  return (
    <div className="flex items-center justify-center h-64">
      <div className="animate-spin rounded-full size-8 border-b-2 border-primary-600"></div>
    </div>
  );
}

function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route
            index
            element={
              <Suspense fallback={<PageLoader />}>
                <Dashboard />
              </Suspense>
            }
          />
          <Route
            path="run/:runId"
            element={
              <Suspense fallback={<PageLoader />}>
                <TestRun />
              </Suspense>
            }
          />
          <Route
            path="test/:testId"
            element={
              <Suspense fallback={<PageLoader />}>
                <TestPage />
              </Suspense>
            }
          />
          <Route
            path="registry"
            element={
              <Suspense fallback={<PageLoader />}>
                <Registry />
              </Suspense>
            }
          />
          <Route
            path="clients"
            element={
              <Suspense fallback={<PageLoader />}>
                <Clients />
              </Suspense>
            }
          />
          <Route
            path="builder"
            element={
              <Suspense fallback={<PageLoader />}>
                <TestBuilder />
              </Suspense>
            }
          />
          <Route
            path="api-docs"
            element={
              <Suspense fallback={<PageLoader />}>
                <ApiDocs />
              </Suspense>
            }
          />
        </Route>
      </Routes>
    </AuthProvider>
  );
}

export default App;
