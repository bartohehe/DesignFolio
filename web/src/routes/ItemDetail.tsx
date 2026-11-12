import { lazy, Suspense, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

// three.js waży kilkaset kilobajtów — ładujemy go dopiero wtedy,
// gdy oglądana rzecz naprawdę ma model 3D.
const ModelViewer = lazy(() => import('../components/ModelViewer'))

import { useDeleteItem, useItem, useUpdateItem } from '../api/queries'
import type { Item } from '../api/types'
import { AssetShelf } from '../components/AssetShelf'
import { Markdown } from '../components/Markdown'
import { ErrorState } from '../components/states'
import { Button, ButtonLink } from '../components/ui'
import { IconArrowLeft, IconExternal, IconMark, IconPencil, IconTrash, KindIcon } from '../components/icons'
import { formatDate } from '../lib/format'
import { KIND_LABEL, META_FIELDS, STATUS_LABEL, unknownMetaKeys } from '../lib/kinds'

export function ItemDetail() {
  const { slug = '' } = useParams()
  const item = useItem(slug)

  if (item.isPending) return <DetailSkeleton />
  if (item.isError) return <ErrorState error={item.error} onRetry={() => void item.refetch()} />
  if (!item.data) return null

  return <Detail item={item.data} />
}

/** Formaty, które umie pokazać przeglądarkowy podgląd 3D. */
const VIEWABLE_MODELS = ['.stl', '.3mf']

function Detail({ item }: { item: Item }) {
  const navigate = useNavigate()
  const models = (item.assets ?? []).filter(
    (a) =>
      a.kind === 'model' &&
      VIEWABLE_MODELS.some((ext) => a.filename.toLowerCase().endsWith(ext)),
  )
  const update = useUpdateItem(item.id)
  const remove = useDeleteItem()
  const [confirmingDelete, setConfirmingDelete] = useState(false)

  return (
    <article>
      <Link
        to="/"
        className="transition-tint inline-flex items-center gap-2 rounded-sm text-sm text-faint hover:text-ink"
      >
        <IconArrowLeft />
        Arkusz
      </Link>

      <header className="mt-8 flex flex-wrap items-start justify-between gap-x-8 gap-y-6">
        <div className="min-w-0">
          {/* Tytuł prowadzi; rodzaj i stan idą pod nim jako opis tego tytułu,
              a nie jako etykieta zapowiadająca nagłówek. */}
          <h1 className="max-w-4xl text-3xl text-ink">{item.title}</h1>
          <p className="mt-3 flex items-center gap-2 font-mono text-2xs uppercase text-faint">
            <KindIcon kind={item.kind} className="size-3.5" />
            {KIND_LABEL[item.kind]}
            <span aria-hidden="true">·</span>
            {STATUS_LABEL[item.status]}
          </p>
        </div>

        <div className="flex items-center gap-1">
          <Button
            variant={item.favorite ? 'mark' : 'ghost'}
            onClick={() => update.mutate({ favorite: !item.favorite })}
            aria-pressed={item.favorite}
          >
            <IconMark className={item.favorite ? 'size-5 -rotate-12' : 'size-5'} />
            {item.favorite ? 'Ulubione' : 'Do ulubionych'}
          </Button>

          <ButtonLink to={`/rzecz/${item.slug}/edycja`} variant="ghost">
            <IconPencil />
            Edytuj
          </ButtonLink>

          {confirmingDelete ? (
            // Dwustopniowe potwierdzenie zamiast okna modalnego — nie przerywa
            // pracy, a i tak wymaga świadomego drugiego kliknięcia.
            <span className="inline-flex items-center gap-1 rounded-md bg-surface pl-3">
              <span className="text-sm text-muted">Usunąć na zawsze?</span>
              <Button
                variant="danger"
                disabled={remove.isPending}
                onClick={() =>
                  remove.mutate(item.id, { onSuccess: () => void navigate('/') })
                }
              >
                {remove.isPending ? 'Usuwam…' : 'Tak, usuń'}
              </Button>
              <Button variant="ghost" onClick={() => setConfirmingDelete(false)}>
                Anuluj
              </Button>
            </span>
          ) : (
            <Button variant="ghost" onClick={() => setConfirmingDelete(true)}>
              <IconTrash />
              Usuń
            </Button>
          )}
        </div>
      </header>

      {remove.isError && (
        <p className="mt-6 text-sm text-danger" role="alert">
          Nie udało się usunąć: {remove.error.message}
        </p>
      )}

      {/* Dwie kolumny dopiero od xl: przy 1024 px kolumna opisu zwężała się
          do ~47 znaków w wierszu, a wygodna miara to 65–75. */}
      <div className="mt-14 grid gap-x-16 gap-y-14 xl:grid-cols-[minmax(0,1fr)_16rem]">
        <div className="min-w-0 space-y-14">
          {item.description ? (
            <Markdown>{item.description}</Markdown>
          ) : (
            <p className="max-w-prose text-base text-faint">
              Bez opisu. Możesz go dopisać w każdej chwili — markdown działa.
            </p>
          )}

          {models.length > 0 && (
            <section>
              <h2 className="font-mono text-2xs uppercase text-faint">Model 3D</h2>
              <div className="mt-4">
                {/* Suspense + lazy: three.js schodzi do osobnej paczki
                    i pobiera się dopiero tutaj, przy rzeczy z modelem. */}
                <Suspense fallback={<ViewerSkeleton />}>
                  <ModelViewer asset={models[0]!} />
                </Suspense>
              </div>
              {models.length > 1 && (
                <p className="mt-2 font-mono text-2xs text-faint">
                  Podgląd pokazuje pierwszy z {models.length} modeli. Kolejność zmienisz
                  na półce plików niżej.
                </p>
              )}
            </section>
          )}

          <AssetShelf itemId={item.id} assets={item.assets ?? []} />
        </div>

        <aside className="space-y-10">
          <MetaBlock item={item} />

          {item.tags.length > 0 && (
            <Block title="Tagi">
              <ul className="flex flex-wrap gap-1.5">
                {item.tags.map((tag) => (
                  <li key={tag.id}>
                    <Link
                      to={`/?tag=${encodeURIComponent(tag.slug)}`}
                      className="transition-tint inline-block rounded-sm bg-surface px-2.5 py-1 font-mono text-xs text-muted hover:text-accent"
                    >
                      {tag.name}
                    </Link>
                  </li>
                ))}
              </ul>
            </Block>
          )}

          {item.external_url && (
            <Block title="Link">
              <a
                href={item.external_url}
                target="_blank"
                rel="noreferrer noopener"
                className="transition-tint inline-flex items-start gap-2 break-all text-sm text-accent hover:underline"
              >
                <IconExternal className="mt-0.5 size-4 shrink-0" />
                {item.external_url}
              </a>
            </Block>
          )}

          <Block title="Daty">
            <dl className="space-y-1.5 font-mono text-xs">
              <DateRow label="Powstało" value={formatDate(item.happened_at)} />
              <DateRow label="Dodano" value={formatDate(item.created_at)} />
              <DateRow label="Zmieniono" value={formatDate(item.updated_at)} />
            </dl>
          </Block>
        </aside>
      </div>
    </article>
  )
}

/**
 * Metadane zależne od rodzaju. Pokazujemy wyłącznie wypełnione pola —
 * pusta lista etykiet z myślnikami nie niesie żadnej informacji.
 *
 * Klucze spoza schematu bieżącego rodzaju też trafiają na ekran: item mógł
 * kiedyś być drukiem 3D, a dziś być "inne". Ukrycie ich wyglądałoby
 * jak utrata danych.
 */
function MetaBlock({ item }: { item: Item }) {
  const known = META_FIELDS[item.kind]
    .map((field) => ({ label: field.label, value: item.metadata[field.key], suffix: field.suffix }))
    .filter((entry) => hasValue(entry.value))

  const extra = unknownMetaKeys(item.kind, item.metadata)
    .map((key) => ({ label: key, value: item.metadata[key], suffix: undefined }))
    .filter((entry) => hasValue(entry.value))

  if (known.length === 0 && extra.length === 0) return null

  return (
    <Block title="Metadane">
      <dl className="space-y-2">
        {[...known, ...extra].map((entry) => (
          <div key={entry.label}>
            <dt className="font-mono text-2xs uppercase text-faint">{entry.label}</dt>
            <dd className="mt-0.5 font-mono text-sm text-ink">
              {formatMetaValue(entry.value)}
              {entry.suffix && <span className="text-faint"> {entry.suffix}</span>}
            </dd>
          </div>
        ))}
      </dl>
    </Block>
  )
}

function Block({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section>
      <h2 className="font-mono text-2xs uppercase text-faint">{title}</h2>
      <div className="mt-3">{children}</div>
    </section>
  )
}

function DateRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-4">
      <dt className="text-faint">{label}</dt>
      <dd className="text-muted">{value}</dd>
    </div>
  )
}

function hasValue(value: unknown): boolean {
  if (value === null || value === undefined || value === '') return false
  if (Array.isArray(value)) return value.length > 0
  return true
}

function formatMetaValue(value: unknown): string {
  if (Array.isArray(value)) return value.join(', ')
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  if (typeof value === 'string') return value
  return JSON.stringify(value)
}

function ViewerSkeleton() {
  return (
    <div
      className="aspect-4/3 w-full animate-pulse rounded-md bg-surface"
      role="status"
      aria-label="Wczytuję podgląd 3D"
    />
  )
}

function DetailSkeleton() {
  return (
    <div role="status" aria-label="Wczytuję">
      <div className="h-3 w-16 animate-pulse rounded-sm bg-surface" />
      <div className="mt-10 h-3 w-24 animate-pulse rounded-sm bg-surface" />
      <div className="mt-4 h-9 w-2/3 animate-pulse rounded-sm bg-surface" />
      <div className="mt-14 grid gap-16 lg:grid-cols-[minmax(0,1fr)_16rem]">
        <div className="space-y-3">
          <div className="h-4 w-full animate-pulse rounded-sm bg-surface" />
          <div className="h-4 w-11/12 animate-pulse rounded-sm bg-surface" />
          <div className="h-4 w-4/5 animate-pulse rounded-sm bg-surface" />
        </div>
        <div className="h-32 animate-pulse rounded-md bg-surface" />
      </div>
    </div>
  )
}
