import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from '@tanstack/react-query'

import { apiFetch, uploadFiles } from './client'
import type {
  ArrangeResult,
  Facets,
  Item,
  ItemInput,
  ItemList,
  ItemPatch,
  ItemQuery,
  TagList,
  UploadResult,
} from './types'

/**
 * Klucze cache'a w jednym miejscu. Dzięki temu unieważnianie po mutacji
 * jest jawne i nie polega na zgadywaniu, jak dokładnie brzmiał klucz.
 */
export const keys = {
  items: (query: ItemQuery = {}) => ['items', query] as const,
  item: (idOrSlug: string) => ['item', idOrSlug] as const,
  tags: ['tags'] as const,
  facets: ['facets'] as const,
}

function buildQueryString(query: ItemQuery): string {
  const params = new URLSearchParams()

  if (query.kind?.length) params.set('kind', query.kind.join(','))
  if (query.status?.length) params.set('status', query.status.join(','))
  if (query.tag?.length) params.set('tag', query.tag.join(','))
  if (query.favorite !== undefined) params.set('favorite', String(query.favorite))
  if (query.q) params.set('q', query.q)
  if (query.sort) params.set('sort', query.sort)
  if (query.order) params.set('order', query.order)
  if (query.limit !== undefined) params.set('limit', String(query.limit))
  if (query.offset !== undefined) params.set('offset', String(query.offset))

  const qs = params.toString()
  return qs ? `?${qs}` : ''
}

export function useItems(query: ItemQuery = {}): UseQueryResult<ItemList> {
  return useQuery({
    queryKey: keys.items(query),
    queryFn: () => apiFetch<ItemList>(`/api/items${buildQueryString(query)}`),
    // Poprzednie wyniki zostają widoczne przy zmianie filtra — siatka
    // przygasa, zamiast znikać i wracać jako skeleton przy każdym kliknięciu.
    placeholderData: (previous) => previous,
  })
}

export function useItem(idOrSlug: string): UseQueryResult<Item> {
  return useQuery({
    queryKey: keys.item(idOrSlug),
    queryFn: () => apiFetch<Item>(`/api/items/${encodeURIComponent(idOrSlug)}`),
    enabled: idOrSlug !== '',
  })
}

export function useTags(): UseQueryResult<TagList> {
  return useQuery({
    queryKey: keys.tags,
    queryFn: () => apiFetch<TagList>('/api/tags'),
  })
}

export function useFacets(): UseQueryResult<Facets> {
  return useQuery({
    queryKey: keys.facets,
    queryFn: () => apiFetch<Facets>('/api/facets'),
  })
}

/**
 * invalidateAll odświeża wszystko, co mogło się zmienić po zapisie:
 * listy, liczniki w indeksie i tagi. Przy jednoosobowej aplikacji
 * z kilkuset itemami to tańsze niż precyzyjne łatanie cache'a.
 */
function useInvalidateAll() {
  const client = useQueryClient()
  return async () => {
    await Promise.all([
      client.invalidateQueries({ queryKey: ['items'] }),
      client.invalidateQueries({ queryKey: keys.tags }),
      client.invalidateQueries({ queryKey: keys.facets }),
    ])
  }
}

export function useCreateItem(): UseMutationResult<Item, Error, ItemInput> {
  const invalidate = useInvalidateAll()

  return useMutation({
    mutationFn: (input: ItemInput) =>
      apiFetch<Item>('/api/items', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(input),
      }),
    onSuccess: invalidate,
  })
}

export function useUpdateItem(
  id: string,
): UseMutationResult<Item, Error, ItemPatch> {
  const client = useQueryClient()
  const invalidate = useInvalidateAll()

  return useMutation({
    mutationFn: (patch: ItemPatch) =>
      apiFetch<Item>(`/api/items/${encodeURIComponent(id)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(patch),
      }),
    onSuccess: async (item) => {
      // Świeży item wkładamy do cache'a od razu pod oboma kluczami,
      // bo widok szczegółów adresuje go slugiem, a edycja identyfikatorem.
      client.setQueryData(keys.item(item.id), item)
      client.setQueryData(keys.item(item.slug), item)
      await invalidate()
    },
  })
}

export function useUploadAssets(
  itemId: string,
): UseMutationResult<UploadResult, Error, { files: File[]; onProgress: (percent: number) => void }> {
  const invalidate = useInvalidateAll()
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ files, onProgress }) =>
      uploadFiles<UploadResult>(
        `/api/items/${encodeURIComponent(itemId)}/assets`,
        files,
        onProgress,
      ),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ['item'] })
      await invalidate()
    },
  })
}

export function useArrangeAssets(
  itemId: string,
): UseMutationResult<ArrangeResult, Error, { order: string[]; cover: string | null }> {
  const client = useQueryClient()
  const invalidate = useInvalidateAll()

  return useMutation({
    mutationFn: (body) =>
      apiFetch<ArrangeResult>(`/api/items/${encodeURIComponent(itemId)}/assets`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ['item'] })
      await invalidate()
    },
  })
}

export function useDeleteAsset(): UseMutationResult<void, Error, string> {
  const client = useQueryClient()
  const invalidate = useInvalidateAll()

  return useMutation({
    mutationFn: (assetId: string) =>
      apiFetch<void>(`/api/assets/${encodeURIComponent(assetId)}`, { method: 'DELETE' }),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ['item'] })
      await invalidate()
    },
  })
}

export function useDeleteItem(): UseMutationResult<void, Error, string> {
  const invalidate = useInvalidateAll()

  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(`/api/items/${encodeURIComponent(id)}`, { method: 'DELETE' }),
    onSuccess: invalidate,
  })
}
