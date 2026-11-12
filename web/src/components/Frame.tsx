import { Link } from 'react-router-dom'

import type { Item } from '../api/types'
import { excerpt } from '../lib/format'
import { KIND_LABEL } from '../lib/kinds'
import { assetUrl } from '../lib/urls'
import { Highlight } from './Highlight'
import { IconMark, KindIcon } from './icons'

/**
 * Klatka na arkuszu — jeden item w siatce.
 *
 * Dwa warianty, zgodnie z decyzją użytkownika:
 *   • z okładką — zdjęcie prowadzi, podpis leży pod nim wprost na tle arkusza,
 *   • bez okładki — mniejsza klatka tekstowa na własnym, jaśniejszym tle.
 *
 * Rozróżnienie robi tło i odstęp, nie ramka ani cień. Item bez zdjęcia
 * nie dostaje placeholdera: notatka i link to pełnoprawne rzeczy,
 * a nie uszkodzone obrazki.
 */
export function Frame({ item, query = '' }: { item: Item; query?: string }) {
  return item.cover ? (
    <CoverFrame item={item} query={query} />
  ) : (
    <TextFrame item={item} query={query} />
  )
}

function CoverFrame({ item, query }: { item: Item; query: string }) {
  const cover = item.cover!

  return (
    <article className="group relative">
      <Link to={`/rzecz/${item.slug}`} className="block rounded-md">
        <div
          className="overflow-hidden rounded-md bg-surface"
          // Proporcja bierze się z prawdziwych wymiarów zdjęcia, ale jest
          // przycięta do zakresu, żeby jedna skrajnie wąska panorama
          // nie rozwaliła rytmu kolumny.
          style={{ aspectRatio: coverAspect(cover.width, cover.height) }}
        >
          <img
            src={assetUrl(cover.id, 'thumb')}
            alt=""
            loading="lazy"
            decoding="async"
            className="transition-ui size-full object-cover group-hover:scale-[1.02]"
          />
        </div>
        <Caption item={item} query={query} />
      </Link>
      {item.favorite && <Mark />}
    </article>
  )
}

function TextFrame({ item, query }: { item: Item; query: string }) {
  return (
    <article className="group relative">
      <Link
        to={`/rzecz/${item.slug}`}
        className="transition-ui block rounded-md bg-surface px-6 py-6 group-hover:-translate-y-0.5"
      >
        <Caption item={item} inset query={query} />
      </Link>
      {item.favorite && <Mark />}
    </article>
  )
}

function Caption({
  item,
  inset = false,
  query = '',
}: {
  item: Item
  inset?: boolean
  query?: string
}) {
  const summary = excerpt(item.description, 140)

  return (
    <div className={inset ? '' : 'mt-3'}>
      {/* Tytuł na pierwszym miejscu — rodzaj jest jego opisem, nie zapowiedzią.
          Użytkownik chciał na klatce tytuł, rodzaj i fragment opisu; bez tagów
          i bez dat. */}
      <h2 className="transition-tint text-lg text-ink group-hover:text-accent">
        <Highlight text={item.title} query={query} />
      </h2>
      <p className="mt-1.5 flex items-center gap-1.5 font-mono text-2xs uppercase text-faint">
        <KindIcon kind={item.kind} className="size-3.5" />
        {KIND_LABEL[item.kind]}
      </p>
      {summary && (
        <p className="mt-3 text-sm text-muted">
          <Highlight text={summary} query={query} />
        </p>
      )}
    </div>
  )
}

/**
 * Zakreślenie ulubionej klatki — jak kółko kredką na arkuszu stykowym.
 * Lekki obrót sprawia, że nie wygląda na wydrukowany element interfejsu.
 */
function Mark() {
  return (
    <span
      className="pointer-events-none absolute -right-2 -top-2 text-accent"
      title="Ulubione"
    >
      <IconMark className="size-7 -rotate-12" />
      <span className="sr-only">Ulubione</span>
    </span>
  )
}

function coverAspect(width: number | null, height: number | null): string {
  if (!width || !height) return '4 / 3'

  const ratio = width / height
  const clamped = Math.min(Math.max(ratio, 0.62), 1.6)
  return String(clamped)
}
