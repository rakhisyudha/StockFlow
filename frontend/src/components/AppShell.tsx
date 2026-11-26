import type { ReactNode } from 'react'
import { NavLink } from 'react-router-dom'

interface AppShellProps {
  children: ReactNode
}

export function AppShell({ children }: AppShellProps) {
  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="brand">
          <span className="brand-text">
            <strong>StockFlow</strong>
            <small>Warehouse manager</small>
          </span>
        </div>
        <nav className="app-nav" aria-label="Main">
          <NavLink to="/" end>Products</NavLink>
          <NavLink to="/orders/new">Create order</NavLink>
        </nav>
      </header>
      <main className="app-main">{children}</main>
    </div>
  )
}
