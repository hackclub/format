'use client'

import { useState, useEffect } from 'react'
import { authAPI } from '@/lib/api'
import { User } from '@/types'

export function useAuth() {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadUser()
  }, [])

  const loadUser = async () => {
    try {
      setLoading(true)
      setError(null)
      const userData = await authAPI.getUser()
      setUser(userData)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load user')
    } finally {
      setLoading(false)
    }
  }

  const login = () => {
    window.location.href = authAPI.getLoginURL()
  }

  const logout = async () => {
    try {
      await authAPI.logout()
      setUser(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to logout')
    }
  }

  return {
    user,
    loading,
    error,
    login,
    logout,
    isAuthenticated: !!user,
  }
}
