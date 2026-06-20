import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import LoginPage from './pages/LoginPage'
import DashboardPage from './pages/DashboardPage'
import StoreSetupPage from './pages/StoreSetupPage'
import ProductsPage from './pages/ProductsPage'
import ProductEditPage from './pages/ProductEditPage'
import { useAuth } from './hooks/useAuth'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { token } = useAuth()
  return token ? <>{children}</> : <Navigate to="/login" replace />
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/" element={<PrivateRoute><DashboardPage /></PrivateRoute>} />
        <Route path="/stores/new" element={<PrivateRoute><StoreSetupPage /></PrivateRoute>} />
        <Route path="/stores/:storeId/products" element={<PrivateRoute><ProductsPage /></PrivateRoute>} />
        <Route path="/stores/:storeId/products/new" element={<PrivateRoute><ProductEditPage /></PrivateRoute>} />
        <Route path="/stores/:storeId/products/:productId/edit" element={<PrivateRoute><ProductEditPage /></PrivateRoute>} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
