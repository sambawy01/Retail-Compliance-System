import { lazy, Suspense } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from './contexts/AuthContext'
import Layout from './components/Layout'
import Login from './pages/Login'

// Route-level code splitting: each page is loaded on demand as its own chunk.
// Login is kept eager so the unauthenticated landing screen renders instantly.
const Dashboard = lazy(() => import('./pages/Dashboard'))
const Cameras = lazy(() => import('./pages/Cameras'))
const Events = lazy(() => import('./pages/Events'))
const FaceID = lazy(() => import('./pages/FaceID'))
const Settings = lazy(() => import('./pages/Settings'))

function PageLoader() {
  return (
    <div className="flex items-center justify-center h-64 text-text-muted">
      <div className="h-6 w-6 border-2 border-accent border-t-transparent rounded-full animate-spin" />
    </div>
  )
}

function ProtectedLayout() {
  const { isAuthenticated, loading } = useAuth()
  // While checking session via /me, show loader instead of redirecting
  if (loading) return <PageLoader />
  if (!isAuthenticated) return <Navigate to="/login" replace />
  return <Layout />
}

export default function App() {
  const { isAuthenticated } = useAuth()
  return (
    <Suspense fallback={<PageLoader />}>
      <Routes>
        <Route path="/login" element={isAuthenticated ? <Navigate to="/" replace /> : <Login />} />
        <Route element={<ProtectedLayout />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/cameras" element={<Cameras />} />
          <Route path="/cameras/:id" element={<Cameras />} />
          <Route path="/events" element={<Events />} />
          <Route path="/faceid" element={<FaceID />} />
          <Route path="/settings" element={<Settings />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Suspense>
  )
}
