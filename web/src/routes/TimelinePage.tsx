import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'

import { apiFetch } from '../api/client'
import type { Item } from '../api/types'
import { EmptyState, ErrorState } from '../components/states'
import { KindIcon, IconMark } from '../components/icons'
import { ButtonLink } from '../components/ui'
import { plural } from '../lib/format'
import { KIND_LABEL } from '../lib/kinds'
import { assetUrl } from '../lib/urls'

interface TimelineResponse {
  items: Item[]
  total: number
}

/** Miesiąc w mianowniku — Intl odmienia go w dopełniaczu ("listopada"). */
const MONTHS = [
  'styczeń', 'luty', 'marzec', 'kwiecień', 'maj', 'czerwiec',
  'lipiec', 'sierpień', 'wrzesień', 'październik', 'listopad', 'grudzień',
]

interface MonthGroup {
  key: string
  month: number
  items: Item[]
}

interface YearGroup {
  year: number
  count: number
  months: MonthGroup[]
}

/**
 * Oś czasu: wszystko chronologicznie, pogrupowane po latach i miesiącach.
 *
 * Data brana pod uwagę to happened_at, a gdy go nie ma — created_at.
 * Dokładnie tak samo sortuje backend, więc kolejność się nie rozjeżdża.
 */
export function TimelinePage() {
  const timeline = useQuery({
    queryKey: ['timeline'],
    queryFn: () => apiFetch<TimelineResponse>('/api/timeline'),
  })

  if (timeline.isError) {
    return <ErrorState error={timeline.error} onRetry={() => void timeline.refetch()} />
  }

  const years = groupByYear(timeline.data?.items ?? [])

  return (
    <div>
      <h1 className="text-3xl text-ink">Oś czasu</h1>
      {timeline.data && timeline.data.total > 0 && (
        <p className="mt-3 font-mono text-xs tabular-nums text-faint">
          {plural(timeline.data.total, 'rzecz', 'rzeczy', 'rzeczy')} ·{' '}
          {plural(years.length, 'rok', 'lata', 'lat')}
        </p>
      )}

      {timeline.isPending && (
        <div className="mt-14 space-y-10" role="status" aria-label="Wczytuję oś czasu">
          {[0, 1, 2].map((i) => (
            <div key={i} className="flex gap-8">
              <div className="h-8 w-16 animate-pulse rounded-sm bg-surface" />
              <div className="flex-1 space-y-3">
                <div className="h-5 w-2/3 animate-pulse rounded-sm bg-surface" />
                <div className="h-5 w-1/2 animate-pulse rounded-sm bg-surface" />
              </div>
            </div>
          ))}
        </div>
      )}

      {timeline.data && years.length === 0 && (
        <div className="mt-4">
          <EmptyState
            title="Oś czasu jest pusta"
            description="Nie ma jeszcze czego układać chronologicznie. Data „kiedy powstało” jest opcjonalna — bez niej liczy się dzień dodania."
            action={<ButtonLink to="/nowa">Dodaj pierwszą rzecz</ButtonLink>}
          />
        </div>
      )}

      <div className="mt-14 space-y-16">
        {years.map((year) => (
          <section key={year.year}>
            <div className="flex items-baseline gap-4 border-t border-line pt-4">
              <h2 className="font-mono text-2xl tabular-nums text-ink">{year.year}</h2>
              <span className="font-mono text-2xs tabular-nums text-faint">
                {year.count}
              </span>
            </div>

            <div className="mt-6 space-y-8">
              {year.months.map((group) => (
                <div key={group.key} className="sm:grid sm:grid-cols-[7rem_minmax(0,1fr)] sm:gap-8">
                  <h3 className="font-mono text-2xs uppercase text-faint sm:pt-1.5">
                    {MONTHS[group.month]}
                  </h3>
                  <ul className="mt-2 space-y-0.5 sm:mt-0">
                    {group.items.map((item) => (
                      <li key={item.id}>
                        <Entry item={item} />
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  )
}

function Entry({ item }: { item: Item }) {
  return (
    <Link
      to={`/rzecz/${item.slug}`}
      className="transition-tint group flex items-center gap-3 rounded-sm py-1.5 hover:bg-surface"
    >
      {/* Miniatura tylko jako drobny znacznik — oś czasu ma być listą
          do przebiegnięcia wzrokiem, nie drugą siatką. */}
      {item.cover ? (
        <img
          src={assetUrl(item.cover.id, 'thumb')}
          alt=""
          loading="lazy"
          className="size-8 shrink-0 rounded-sm object-cover"
        />
      ) : (
        <span className="flex size-8 shrink-0 items-center justify-center rounded-sm bg-surface text-faint">
          <KindIcon kind={item.kind} className="size-3.5" />
        </span>
      )}

      <span className="min-w-0 flex-1 truncate text-base text-muted group-hover:text-ink">
        {item.title}
      </span>

      {item.favorite && <IconMark className="size-4 shrink-0 -rotate-12 text-accent" />}

      <span className="hidden shrink-0 font-mono text-2xs uppercase text-faint sm:block">
        {KIND_LABEL[item.kind]}
      </span>
    </Link>
  )
}

function groupByYear(items: Item[]): YearGroup[] {
  const years = new Map<number, Map<number, Item[]>>()

  for (const item of items) {
    const date = new Date(item.happened_at ?? item.created_at)
    const year = date.getFullYear()
    const month = date.getMonth()

    let months = years.get(year)
    if (!months) {
      months = new Map()
      years.set(year, months)
    }
    months.set(month, [...(months.get(month) ?? []), item])
  }

  return [...years.entries()]
    .sort(([a], [b]) => b - a)
    .map(([year, months]) => ({
      year,
      count: [...months.values()].reduce((sum, list) => sum + list.length, 0),
      months: [...months.entries()]
        .sort(([a], [b]) => b - a)
        .map(([month, list]) => ({ key: `${year}-${month}`, month, items: list })),
    }))
}
