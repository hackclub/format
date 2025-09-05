import { User, Asset, TransformResult, BatchInput, BatchResult } from '@/types'

const API_BASE = '/api'

export class APIError extends Error {
  constructor(message: string, public status: number) {
    super(message)
    this.name = 'APIError'
  }
}

async function apiRequest<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const url = `${API_BASE}${endpoint}`
  
  const response = await fetch(url, {
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    ...options,
  })

  if (!response.ok) {
    const errorText = await response.text()
    throw new APIError(errorText || `HTTP ${response.status}`, response.status)
  }

  return response.json()
}

// Auth API
export const authAPI = {
  async getUser(): Promise<User | null> {
    try {
      return await apiRequest<User>('/auth/me')
    } catch (error) {
      if (error instanceof APIError && error.status === 401) {
        return null
      }
      throw error
    }
  },

  async logout(): Promise<void> {
    await apiRequest('/auth/logout', { method: 'POST' })
  },

  getLoginURL(): string {
    return `${API_BASE}/auth/login`
  },
}

// Assets API
export const assetsAPI = {
  async uploadFromURL(url: string): Promise<Asset> {
    return apiRequest<Asset>('/assets', {
      method: 'POST',
      body: JSON.stringify({ url }),
    })
  },

  async uploadFromDataURI(dataUri: string): Promise<Asset> {
    return apiRequest<Asset>('/assets', {
      method: 'POST',
      body: JSON.stringify({ dataUri }),
    })
  },

  async uploadFile(file: File): Promise<Asset> {
    const formData = new FormData()
    formData.append('file', file)
    
    return fetch(`${API_BASE}/assets`, {
      method: 'POST',
      credentials: 'include',
      body: formData,
    }).then(async (response) => {
      if (!response.ok) {
        const errorText = await response.text()
        throw new APIError(errorText || `HTTP ${response.status}`, response.status)
      }
      return response.json()
    })
  },

  async uploadBatch(items: BatchInput[]): Promise<BatchResult> {
    return apiRequest<BatchResult>('/assets/batch', {
      method: 'POST',
      body: JSON.stringify({ items }),
    })
  },
}

// HTML API
export const htmlAPI = {
  async transform(html: string): Promise<TransformResult> {
    return apiRequest<TransformResult>('/html/transform', {
      method: 'POST',
      body: JSON.stringify({ html }),
    })
  },
}

// Config API
export const configAPI = {
  async getConfig(): Promise<{ cdnBaseUrl: string }> {
    return apiRequest<{ cdnBaseUrl: string }>('/config')
  },
}

