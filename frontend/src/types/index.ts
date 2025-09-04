export interface User {
  sub: string
  email: string
  name: string
  picture: string
  hd: string
}

export interface Asset {
  url: string
  mime: string
  width: number
  height: number
  bytes: number
  hash: string
  deduped: boolean
  key?: string
}

export interface TransformStats {
  images_processed: number
  images_rehosted: number
  styles_removed: number
  scripts_removed: number
}

export interface TransformResult {
  html: string
  messages: string[]
  stats: TransformStats
}

export interface BatchInput {
  url?: string
  dataUri?: string
}

export interface BatchResult {
  assets: Asset[]
  count: number
}
