import type { ReactNode } from 'react'
import { Link, useSearchParams } from 'react-router-dom'

import { useFacets, useTags } from '../api/queries'
import { ITEM_KINDS, ITEM_STATUSES, type ItemKind, type ItemStatus } from '../api/types'
import { KIND_LABEL, STATUS_LABEL } from '../lib/kinds'
import { IconTag, KindIcon } from './icons'
import { SearchBox } from './SearchBox'

/**
 * Indeks na marginesie arkusza: pełna lista rodzajów i stanów z licznikami.
 *
 * Stan filtrów mieszka w adresie URL, nie w useState — dzięki temu działa
 * przycisk Wstecz, a widok da się odświeżyć bez utraty zawężenia.
 */
export function IndexRail() {
  const [params, setParams] = useSearchParams()
  const facets = useFacets()

  const activeKinds = readList(params.get('kind'))
  const activeStatuses = readList(params.get('status'))
  const activeTags = readList(params.get('tag'))
  const favoriteOnly = params.get('favorite') === 'true'

  const countOf = (list: { value: string; count: number }[] | undefined, value: string) =>
    list?.find((f) => f.value === value)?.count ?? 0

  function toggle(key: 'kind' | 'status' | 'tag', value: string) {
    const current = readList(params.get(key))
    const next = current.includes(value)
      ? current.filter((v) => v !== value)
      : [...current, value]

    const updated = new URLSearchParams(params)
    if (next.length === 0) updated.delete(key)
    else updated.set(key, next.join(','))
    setParams(updated, { replace: true })
  }

  function toggleFavorite() {
    const updated = new URLSearchParams(params)
    if (favoriteOnly) updated.delete('favorite')
    else updated.set('favorite', 'true')
    setParams(updated, { replace: true })
  }

  function clearAll() {
    setParams(new URLSearchParams(), { replace: true })
  }

  const nothingActive =
    activeKinds.length === 0 &&
    activeStatuses.length === 0 &&
    activeTags.length === 0 &&
    !favoriteOnly &&
    !params.get('q')

  return (
    <nav aria-label="Indeks arkusza" className="space-y-8">
      <SearchBox />

      <Group title="Rodzaj">
        <Row label="Wszystko" count={facets.data?.total} active={nothingActive} onClick={clearAll} />
        {ITEM_KINDS.map((kind: ItemKind) => (
          <Row
            key={kind}
            label={KIND_LABEL[kind]}
            icon={<KindIcon kind={kind} className="size-3.5" />}
            count={countOf(facets.data?.kinds, kind)}
            active={activeKinds.includes(kind)}
            onClick={() => toggle('kind', kind)}
          />
        ))}
      </Group>

      <Group title="Stan">
        {ITEM_STATUSES.map((status: ItemStatus) => (
          <Row
            key={status}
            label={STATUS_LABEL[status]}
            count={countOf(facets.data?.statuses, status)}
            active={activeStatuses.includes(status)}
            onClick={() => toggle('status', status)}
          />
        ))}
        <Row
          label="Ulubione"
          count={facets.data?.favorites}
          active={favoriteOnly}
          onClick={toggleFavorite}
        />
      </Group>

      <TagGroup active={activeTags} onToggle={(slug) => toggle('tag', slug)} />
    </nav>
  )
}

/** Ile tagów mieści się w marginesie, zanim lista przestaje być indeksem. */
const RAIL_TAG_LIMIT = 12

function TagGroup({
  active,
  onToggle,
}: {
  active: string[]
  onToggle: (slug: string) => void
}) {
  const tags = useTags()
  const all = tags.data?.tags ?? []

  if (all.length === 0) return null

  // Najczęściej używane na wierzchu, ale aktywne zawsze widoczne —
  // inaczej nie dałoby się odznaczyć tagu z ogona listy.
  const activeSet = new Set(active)
  const shown = [
    ...all.filter((t) => activeSet.has(t.slug)),
    ...all.filter((t) => !activeSet.has(t.slug)).slice(0, RAIL_TAG_LIMIT),
  ]
  const hidden = all.length - shown.length

  return (
    <Group title="Tagi">
      {shown.map((tag) => (
        <Row
          key={tag.id}
          label={tag.name}
          icon={<IconTag className="size-3.5" />}
          count={tag.count}
          active={activeSet.has(tag.slug)}
          onClick={() => onToggle(tag.slug)}
        />
      ))}
      {hidden > 0 && (
        <li>
          <Link
            to="/tagi"
            className="transition-tint block py-1 text-sm text-faint hover:text-ink"
          >
            …i jeszcze {hidden}
          </Link>
        </li>
      )}
    </Group>
  )
}

function Group({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section>
      <h2 className="font-mono text-2xs uppercase text-faint">{title}</h2>
      <ul className="mt-3 space-y-0.5">{children}</ul>
    </section>
  )
}

function Row({
  label,
  count,
  active,
  onClick,
  icon,
}: {
  label: string
  count: number | undefined
  active: boolean
  onClick: () => void
  icon?: ReactNode
}) {
  // Filtr, który nie ma czego pokazać, jest wyłączony — chyba że akurat
  // jest aktywny, bo wtedy musi dać się wyłączyć.
  const empty = count === 0 && !active

  return (
    <li>
      <button
        type="button"
        onClick={onClick}
        disabled={empty}
        aria-pressed={active}
        className={[
          'transition-tint flex w-full items-center gap-2 rounded-sm py-1 text-left text-sm',
          active ? 'text-accent' : 'text-muted',
          empty ? 'cursor-default text-faint/60' : 'hover:text-ink',
        ].join(' ')}
      >
        {icon && <span className={active ? 'text-accent' : 'text-faint'}>{icon}</span>}
        <span className="min-w-0 flex-1 truncate">{label}</span>
        <span className="font-mono text-2xs tabular-nums">{count ?? '·'}</span>
      </button>
    </li>
  )
}

function readList(raw: string | null): string[] {
  if (!raw) return []
  return raw.split(',').filter(Boolean)
}
