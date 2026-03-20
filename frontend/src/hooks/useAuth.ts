// frontend/src/hooks/useAuth.ts
import { create } from 'zustand'

interface AuthState {
  token: string | null
  user: { id: string; email: string; username: string; role: string } | null
  login: (token: string, user: AuthState['user']) => void
  logout: () => void
}

export const useAuth = create<AuthState>((set) => ({
  token: localStorage.getItem('token'),
  user: JSON.parse(localStorage.getItem('user') || 'null'),
  login: (token, user) => {
    localStorage.setItem('token', token)
    localStorage.setItem('user', JSON.stringify(user))
    set({ token, user })
  },
  logout: () => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    set({ token: null, user: null })
  },
}))
