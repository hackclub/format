'use client'

import { useAuth } from '@/hooks/useAuth'
import { LoadingSpinner } from './LoadingSpinner'

interface AuthGuardProps {
  children: React.ReactNode
}

export function AuthGuard({ children }: AuthGuardProps) {
  const { user, loading, error, login } = useAuth()

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-xl font-bold text-red-600 mb-4">Authentication Error</h1>
          <p className="text-gray-600 mb-4">{error}</p>
          <button
            onClick={login}
            className="bg-hack-red text-white px-4 py-2 rounded hover:bg-red-600 transition-colors"
          >
            Try Again
          </button>
        </div>
      </div>
    )
  }

  if (!user) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center max-w-md">
          <div className="mb-8">
            <h1 className="text-3xl font-bold text-gray-900 mb-2">format.hackclub.com</h1>
          </div>
          
          <div className="bg-white rounded-lg shadow-lg p-8">
            <button
              onClick={login}
              className="w-full bg-hack-red text-white px-4 py-3 rounded-lg hover:bg-red-600 transition-colors font-semibold"
            >
              Sign in with Google
            </button>
            
            <p className="text-xs text-gray-500 mt-4">
              Only @hackclub.com accounts are allowed
            </p>
          </div>
        </div>
      </div>
    )
  }

  return <>{children}</>
}
