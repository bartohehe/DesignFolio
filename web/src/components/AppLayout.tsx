import { useState } from 'react'
import { Link, NavLink, Outlet, useLocation } from 'react-router-dom'

import { openTokenDialog } from '../api/client'
import { IconChevronDown, IconDownload, IconKey, IconPlus } from './icons'
import { IndexRail } from './IndexRail'
import { TokenDialog } from './TokenDialog'
import { ButtonLink } from './ui'

/**
 * Układ całej aplikacji: wąski margines z nawigacją i indeksem po lewej,
 * arkusz po prawej. Celowo asymetryczny — treść nie jest wyśrodkowana
 * w kolumnie, tylko dosunięta do marginesu, jak na arkuszu roboczym.
 *
 * Na wąskich ekranach margines przenosi się nad treść i zwija do rozwijanego
 * indeksu, zamiast znikać — dostęp do filtrów zostaje pełny.
 */
export function AppLayout() {
  const [indexOpen, setIndexOpen] = useState(false)
  const location = useLocation()

  // Indeks filtruje arkusz, więc na innych widokach nie ma czego pokazywać.
  const onSheet = location.pathname === '/'

  return (
    <div className="min-h-dvh lg:grid lg:grid-cols-[16rem_minmax(0,1fr)]">
      {/* Skok do treści. Niewidoczny, dopóki nie dostanie focusa z klawiatury —
          bez niego trzeba przejść Tabem przez cały indeks (kilkadziesiąt
          pozycji), żeby dotrzeć do arkusza. */}
      <a
        href="#tresc"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-10 focus:rounded-md focus:bg-accent focus:px-4 focus:py-2 focus:text-sm focus:font-medium focus:text-canvas"
      >
        Przejdź do treści
      </a>

      <header className="flex flex-col px-6 pt-8 lg:sticky lg:top-0 lg:h-dvh lg:overflow-y-auto lg:px-8 lg:pb-8">
        <div className="flex items-center justify-between gap-4">
          <Wordmark />
          <div className="flex items-center gap-2 lg:hidden">
            <ButtonLink to="/nowa" variant="primary">
              <IconPlus />
              Dodaj
            </ButtonLink>
            {onSheet && (
              <button
                type="button"
                onClick={() => setIndexOpen((open) => !open)}
                aria-expanded={indexOpen}
                aria-controls="indeks"
                className="transition-tint rounded-md p-2 text-muted hover:bg-surface hover:text-ink"
              >
                <IconChevronDown className={indexOpen ? 'size-5 rotate-180' : 'size-5'} />
                <span className="sr-only">{indexOpen ? 'Zwiń indeks' : 'Rozwiń indeks'}</span>
              </button>
            )}
          </div>
        </div>

        <div className="mt-8 hidden lg:block">
          <ButtonLink to="/nowa" variant="primary" className="w-full">
            <IconPlus />
            Dodaj rzecz
          </ButtonLink>
        </div>

        <nav aria-label="Widoki" className="mt-6 flex gap-4 lg:mt-8 lg:flex-col lg:gap-0.5">
          <ViewLink to="/">Arkusz</ViewLink>
          <ViewLink to="/os-czasu">Oś czasu</ViewLink>
          <ViewLink to="/tagi">Tagi</ViewLink>
        </nav>

        {onSheet && (
          <div id="indeks" className={`mt-8 ${indexOpen ? 'block' : 'hidden'} lg:block`}>
            <IndexRail />
          </div>
        )}

        {/* mt-auto zsuwa narzędzia na sam dół marginesu, poza rytm nawigacji.
            Na telefonie ich nie ma: eksport robi się z komputera, a okno
            tokenu i tak otworzy się samo, gdy serwer odrzuci zapis. */}
        <div className="mt-auto hidden flex-col items-start gap-1.5 pt-10 lg:flex">
          <a
            href="/api/export"
            download
            className="transition-tint inline-flex items-center gap-2 rounded-sm text-xs text-faint hover:text-ink"
          >
            <IconDownload className="size-3.5" />
            Eksport JSON
          </a>
          <button
            type="button"
            onClick={openTokenDialog}
            className="transition-tint inline-flex items-center gap-2 rounded-sm text-xs text-faint hover:text-ink"
          >
            <IconKey className="size-3.5" />
            Token zapisu
          </button>
        </div>
      </header>

      <TokenDialog />

      {/* tabIndex={-1} pozwala nadać focus przez skok kotwicą, ale nie wstawia
          <main> do zwykłej kolejności tabulacji. */}
      <main id="tresc" tabIndex={-1} className="px-6 pb-24 pt-10 lg:pl-0 lg:pr-16 lg:pt-16">
        <Outlet />
      </main>
    </div>
  )
}

function ViewLink({ to, children }: { to: string; children: React.ReactNode }) {
  return (
    <NavLink
      to={to}
      // NavLink sam podaje, czy trafiliśmy w bieżącą trasę. `end` sprawia,
      // że "/" nie podświetla się na każdej podstronie.
      end={to === '/'}
      className={({ isActive }) =>
        `transition-tint rounded-sm py-1 text-sm ${isActive ? 'text-accent' : 'text-muted hover:text-ink'}`
      }
    >
      {children}
    </NavLink>
  )
}

function Wordmark() {
  return (
    <Link to="/" className="rounded-sm text-xl text-ink">
      <span className="font-display">Design</span>
      <span className="font-display text-accent">Folio</span>
    </Link>
  )
}
