import { Link } from 'react-router-dom'

import { useTags } from '../api/queries'
import { plural } from '../lib/format'
import { EmptyState, ErrorState } from '../components/states'
import { ButtonLink } from '../components/ui'

/**
 * Wszystkie tagi naraz. Rozmiar napisu niesie liczbę użyć — tag użyty
 * dwadzieścia razy ma być widoczny z drugiego końca pokoju, a użyty raz
 * ma nie zabierać uwagi. Liczba stoi obok, bo sam rozmiar to za mało,
 * żeby coś dokładnie odczytać.
 */
/** Szerokości szkieletu — stała sekwencja, żeby nic nie migotało przy renderze. */
const SKELETON_WIDTHS = ['w-24', 'w-36', 'w-20', 'w-44', 'w-28', 'w-32', 'w-40']

/**
 * Wielkość napisu niesie liczbę użyć — ale skokowo, czterema stopniami
 * z projektowej skali typograficznej, a nie płynnie wyliczonym rem-em.
 * Skala jest zamknięta; rozmiar spoza niej byłby wartością wyjętą z powietrza.
 */
function sizeFor(count: number, max: number): string {
  const share = count / Math.max(max, 1)
  if (share > 0.66) return 'text-xl'
  if (share > 0.33) return 'text-lg'
  if (share > 0.1) return 'text-base'
  return 'text-sm'
}

export function TagsPage() {
  const tags = useTags()

  if (tags.isError) return <ErrorState error={tags.error} onRetry={() => void tags.refetch()} />

  const all = tags.data?.tags ?? []
  const max = all.reduce((top, tag) => Math.max(top, tag.count), 0)

  return (
    <div>
      <h1 className="text-3xl text-ink">Tagi</h1>
      {all.length > 0 && (
        <p className="mt-3 font-mono text-xs tabular-nums text-faint">
          {plural(all.length, 'tag', 'tagi', 'tagów')}
        </p>
      )}

      {tags.isPending && (
        <div className="mt-12 flex flex-wrap gap-x-6 gap-y-4" role="status" aria-label="Wczytuję tagi">
          {Array.from({ length: 14 }, (_, i) => (
            <div
              key={i}
              className={`h-6 animate-pulse rounded-sm bg-surface ${SKELETON_WIDTHS[i % SKELETON_WIDTHS.length]}`}
            />
          ))}
        </div>
      )}

      {tags.data && all.length === 0 && (
        <div className="mt-4">
          <EmptyState
            title="Nie ma jeszcze żadnego tagu"
            description="Tagi powstają same, kiedy wpisujesz je przy dodawaniu rzeczy. Nie trzeba ich nigdzie zakładać."
            action={<ButtonLink to="/nowa">Dodaj rzecz</ButtonLink>}
          />
        </div>
      )}

      {all.length > 0 && (
        <ul className="mt-12 flex max-w-4xl flex-wrap items-baseline gap-x-7 gap-y-5">
          {all.map((tag) => (
            <li key={tag.id}>
              <Link
                to={`/?tag=${encodeURIComponent(tag.slug)}`}
                className={`transition-tint text-muted hover:text-accent ${sizeFor(tag.count, max)}`}
              >
                {tag.name}
                <span className="ml-1.5 font-mono text-2xs tabular-nums text-faint">
                  {tag.count}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
