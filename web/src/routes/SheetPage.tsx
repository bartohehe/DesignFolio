import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'

import { useItems } from '../api/queries'
import type { ItemKind, ItemQuery, ItemStatus } from '../api/types'
import { Sheet } from '../components/Sheet'
import { EmptyState, ErrorState, SheetSkeleton } from '../components/states'
import { Button, ButtonLink } from '../components/ui'
import { Select } from '../components/form'
import { IconPlus } from '../components/icons'
import { plural } from '../lib/format'
import { KIND_LABEL, STATUS_LABEL } from '../lib/kinds'

const PAGE_SIZE = 60

export function SheetPage() {
  const [params, setParams] = useSearchParams()
  const [limit, setLimit] = useState(PAGE_SIZE)

  const kinds = readList(params.get('kind')) as ItemKind[]
  const statuses = readList(params.get('status')) as ItemStatus[]
  const tags = readList(params.get('tag'))
  const favorite = params.get('favorite') === 'true'
  const search = params.get('q') ?? ''
  const sort = params.get('sort') as ItemQuery['sort'] | null
  const order = (params.get('order') ?? 'desc') as NonNullable<ItemQuery['order']>

  const query: ItemQuery = {
    ...(kinds.length > 0 && { kind: kinds }),
    ...(statuses.length > 0 && { status: statuses }),
    ...(tags.length > 0 && { tag: tags }),
    ...(favorite && { favorite: true }),
    ...(search.trim() !== '' && { q: search }),
    // Przy wyszukiwaniu domyślne sortowanie ustala backend (trafność),
    // więc nie narzucamy tu żadnego.
    ...(sort ? { sort } : search.trim() === '' ? { sort: 'happened_at' as const } : {}),
    order,
    limit,
  }

  const items = useItems(query)
  const filtered =
    kinds.length > 0 || statuses.length > 0 || tags.length > 0 || favorite || search.trim() !== ''

  return (
    <div>
      <div className="flex flex-wrap items-end justify-between gap-6">
        <h1 className="min-w-0 text-3xl text-ink">
          {search.trim() !== '' ? (
            <>
              <span className="text-faint">Szukam: </span>
              {search}
            </>
          ) : (
            describeSelection(kinds, statuses, tags, favorite)
          )}
        </h1>

        {items.data && items.data.total > 0 && (
          <div className="flex items-center gap-4">
            <p className="font-mono text-xs tabular-nums text-faint">
              {plural(items.data.total, 'rzecz', 'rzeczy', 'rzeczy')}
            </p>
            <SortControl
              value={`${sort ?? (search.trim() !== '' ? 'relevance' : 'happened_at')}:${order}`}
              searching={search.trim() !== ''}
              onChange={(next) => {
                const [nextSort, nextOrder] = next.split(':')
                const updated = new URLSearchParams(params)
                updated.set('sort', nextSort!)
                updated.set('order', nextOrder!)
                setParams(updated, { replace: true })
              }}
            />
          </div>
        )}
      </div>

      <div className="mt-10">
        {items.isPending && <SheetSkeleton />}

        {items.isError && <ErrorState error={items.error} onRetry={() => void items.refetch()} />}

        {items.data && items.data.items.length === 0 && !filtered && (
          <EmptyState
            title="Arkusz jest pusty"
            description="Nic tu jeszcze nie ma. Wrzuć pierwszą rzecz — wystarczy tytuł i rodzaj, resztę można uzupełnić później."
            action={
              <ButtonLink to="/nowa" variant="primary">
                <IconPlus />
                Dodaj pierwszą rzecz
              </ButtonLink>
            }
          />
        )}

        {items.data && items.data.items.length === 0 && filtered && (
          <EmptyState
            title="Nic nie pasuje"
            description={
              search.trim() !== ''
                ? 'Żadna rzecz nie zawiera wszystkich szukanych słów w tytule, opisie ani tagach. Spróbuj krótszej frazy — wystarczy początek słowa.'
                : 'Żadna rzecz nie spełnia wszystkich zaznaczonych warunków. Odznacz część filtrów w indeksie.'
            }
            action={
              <Button onClick={() => setParams(new URLSearchParams(), { replace: true })}>
                Wyczyść wszystko
              </Button>
            }
          />
        )}

        {items.data && items.data.items.length > 0 && (
          <>
            {/* Podczas dociągania nowych wyników arkusz przygasa,
                zamiast znikać i wracać jako szkielet. */}
            <div className={items.isFetching ? 'opacity-60 transition-ui' : 'transition-ui'}>
              <Sheet items={items.data.items} query={search} />
            </div>

            {items.data.total > items.data.items.length && (
              <div className="mt-12 flex justify-center">
                <Button onClick={() => setLimit((current) => current + PAGE_SIZE)}>
                  Wczytaj więcej
                </Button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}

function SortControl({
  value,
  searching,
  onChange,
}: {
  value: string
  searching: boolean
  onChange: (next: string) => void
}) {
  return (
    <>
      <label htmlFor="sort" className="sr-only">
        Kolejność
      </label>
      <Select
        id="sort"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-auto py-1.5 text-sm"
      >
        {/* Trafność ma sens tylko wtedy, gdy jest czego szukać. */}
        {searching && <option value="relevance:desc">Najtrafniejsze</option>}
        <option value="happened_at:desc">Od najnowszych</option>
        <option value="happened_at:asc">Od najstarszych</option>
        <option value="updated_at:desc">Ostatnio zmieniane</option>
        <option value="title:asc">Alfabetycznie</option>
      </Select>
    </>
  )
}

/** Nagłówek nazywa to, co widać: aktywne zawężenie albo po prostu wszystko. */
function describeSelection(
  kinds: ItemKind[],
  statuses: ItemStatus[],
  tags: string[],
  favorite: boolean,
): string {
  const parts: string[] = []

  if (kinds.length > 0) parts.push(kinds.map((k) => KIND_LABEL[k]).join(', '))
  if (statuses.length > 0) parts.push(statuses.map((s) => STATUS_LABEL[s]).join(', '))
  if (tags.length > 0) parts.push(tags.map((t) => `#${t}`).join(' '))
  if (favorite) parts.push('Ulubione')

  return parts.length === 0 ? 'Wszystko' : parts.join(' · ')
}

function readList(raw: string | null): string[] {
  if (!raw) return []
  return raw.split(',').filter(Boolean)
}
