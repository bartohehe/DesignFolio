/**
 * Jedyne miejsce z typami odpowiedzi API. Rośnie razem z backendem —
 * w projekcie nie ma `any`, więc każdy nowy endpoint dostaje tu swój typ.
 */

/** Kształt błędu zwracany przez każdy endpoint API. */
export interface ApiErrorBody {
  error: {
    code: ApiErrorCode
    message: string
  }
}

export type ApiErrorCode =
  | 'bad_request'
  | 'not_found'
  | 'conflict'
  | 'unauthorized'
  | 'payload_too_large'
  | 'unsupported_media_type'
  | 'internal'

/** GET /api/health */
export interface Health {
  status: 'ok' | 'degraded'
  version: string
  go: string
  db: string
  time: string
}

export const ITEM_KINDS = [
  'project',
  'drawing',
  'print3d',
  'photo',
  'note',
  'link',
  'other',
] as const
export type ItemKind = (typeof ITEM_KINDS)[number]

export const ITEM_STATUSES = ['wip', 'done', 'archived'] as const
export type ItemStatus = (typeof ITEM_STATUSES)[number]

export type AssetKind = 'image' | 'model' | 'document' | 'archive' | 'other'

export interface Asset {
  id: string
  item_id: string
  filename: string
  mime: string
  size: number
  kind: AssetKind
  width: number | null
  height: number | null
  checksum: string
  sort_order: number
  is_cover: boolean
  has_thumb: boolean
  created_at: string
}

export interface Tag {
  id: string
  name: string
  slug: string
  count?: number
}

/**
 * Metadane są celowo luźno typowane: ich kształt zależy od `kind`,
 * a schemat pól mieszka w src/lib/kinds.ts. `unknown` zamiast `any`
 * wymusza zawężenie typu przed użyciem wartości.
 */
export type Metadata = Record<string, unknown>

export interface Item {
  id: string
  title: string
  slug: string
  description: string
  kind: ItemKind
  status: ItemStatus
  favorite: boolean
  external_url: string | null
  metadata: Metadata
  happened_at: string | null
  created_at: string
  updated_at: string
  tags: Tag[]
  cover: Asset | null
  /** Wypełniane tylko w widoku pojedynczego itemu. */
  assets?: Asset[]
}

/** POST /api/items/{id}/assets */
export interface UploadResult {
  uploaded: { asset: Asset; duplicate: boolean }[]
  skipped: { filename: string; reason: string }[]
}

/** PATCH /api/items/{id}/assets */
export interface ArrangeResult {
  assets: Asset[]
}

/** GET /api/items */
export interface ItemList {
  items: Item[]
  total: number
  limit: number
  offset: number
}

/** POST /api/items */
export interface ItemInput {
  title: string
  description: string
  kind: ItemKind
  status: ItemStatus
  favorite: boolean
  external_url: string | null
  metadata: Metadata
  happened_at: string | null
  tags: string[]
}

/**
 * PATCH /api/items/{id} — każde pole jest opcjonalne.
 * Pominięcie klucza zostawia pole nietknięte, jawny `null` je czyści.
 * Odpowiada typowi model.Optional[T] po stronie Go.
 */
export type ItemPatch = Partial<{
  title: string
  description: string
  kind: ItemKind
  status: ItemStatus
  favorite: boolean
  external_url: string | null
  metadata: Metadata
  happened_at: string | null
  tags: string[]
}>

/** GET /api/tags */
export interface TagList {
  tags: Required<Tag>[]
}

/** GET /api/facets */
export interface Facet {
  value: string
  count: number
}

export interface Facets {
  total: number
  favorites: number
  kinds: Facet[]
  statuses: Facet[]
}

/** Parametry listy — trafiają wprost do query stringa. */
export interface ItemQuery {
  kind?: ItemKind[]
  status?: ItemStatus[]
  tag?: string[]
  favorite?: boolean
  /** Fraza wyszukiwania pełnotekstowego. */
  q?: string
  sort?: 'relevance' | 'happened_at' | 'created_at' | 'updated_at' | 'title'
  order?: 'asc' | 'desc'
  limit?: number
  offset?: number
}
