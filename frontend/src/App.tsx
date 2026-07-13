import { NavLink, Route, Routes } from 'react-router-dom'
import { APP_TITLE } from './lib/constants'
import Dashboard from './pages/Dashboard'
import Customers from './pages/Customers'
import Quotes from './pages/Quotes'
import Orders from './pages/Orders'
import Invoices from './pages/Invoices'
import Reports from './pages/Reports'

export default function App() {
  return (
    <div className="min-h-screen flex flex-col">
      <Header />
      <main className="flex-1 max-w-6xl mx-auto w-full px-6 py-8">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/customers" element={<Customers />} />
          <Route path="/quotes/*" element={<Quotes />} />
          <Route path="/orders/*" element={<Orders />} />
          <Route path="/invoices/*" element={<Invoices />} />
          <Route path="/reports" element={<Reports />} />
        </Routes>
      </main>
      <footer className="border-t border-slate-800 py-4 text-center text-sm text-slate-500">
        {APP_TITLE} · Go microservices + React SPA
      </footer>
    </div>
  )
}

function Header() {
  const link = 'px-3 py-2 text-sm font-medium text-slate-400 hover:text-white transition-colors'
  const active = 'text-violet-400'
  return (
    <header className="border-b border-slate-800 bg-slate-950/80 backdrop-blur-md sticky top-0 z-50">
      <nav className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
        <NavLink to="/" className="text-lg font-bold text-white tracking-tight">
          {APP_TITLE}
        </NavLink>
        <div className="flex items-center gap-2">
          <NavLink to="/customers" className={({ isActive }) => `${link} ${isActive ? active : ''}`}>
            Customers
          </NavLink>
          <NavLink to="/quotes" className={({ isActive }) => `${link} ${isActive ? active : ''}`}>
            Quotes
          </NavLink>
          <NavLink to="/orders" className={({ isActive }) => `${link} ${isActive ? active : ''}`}>
            Orders
          </NavLink>
          <NavLink to="/invoices" className={({ isActive }) => `${link} ${isActive ? active : ''}`}>
            Invoices
          </NavLink>
          <NavLink to="/reports" className={({ isActive }) => `${link} ${isActive ? active : ''}`}>
            Reports
          </NavLink>
        </div>
      </nav>
    </header>
  )
}
