import type { ReactNode } from 'react'
import { ApiError } from '../api/client'

/**
 * Szkielet ładowania powtarza rytm arkusza: kilka klatek o różnej wysokości,
 * a nie jeden szary prostokąt. Dzięki temu przejście do wczytanych danych
 * nie przeskakuje layoutu.
 */
export function SheetSkeleton({ count = 9 }: { count?: number }) {
  // Stała, nielosowa sekwencja wysokości — losowanie przy każdym renderze
  // powodowałoby migotanie przy każdym odświeżeniu.
  const heights = ['h-64', 'h-40', 'h-80', 'h-44', 'h-64', 'h-36', 'h-72', 'h-40', 'h-56']

  return (
    <div
      className="columns-1 gap-8 sm:columns-2 xl:columns-3"
      role="status"
      aria-label="Wczytuję arkusz"
    >
      {Array.from({ length: count }, (_, i) => (
        <div key={i} className="mb-8 break-inside-avoid">
          <div className={`${heights[i % heights.length]} animate-pulse rounded-md bg-surface`} />
          <div className="mt-3 h-3 w-2/3 animate-pulse rounded-sm bg-surface" />
        </div>
      ))}
    </div>
  )
}

/** Rysunek pustych klatek — motyw arkusza, na którym nic jeszcze nie ma. */
function EmptyFrames() {
  return (
    <svg
      viewBox="0 0 132 48"
      fill="none"
      stroke="currentColor"
      strokeWidth={1}
      aria-hidden="true"
      className="h-12 w-33 text-line"
    >
      <rect x="0.5" y="0.5" width="40" height="47" rx="2" strokeDasharray="3 4" />
      <rect x="46.5" y="0.5" width="40" height="47" rx="2" strokeDasharray="3 4" />
      <rect x="92.5" y="0.5" width="39" height="47" rx="2" strokeDasharray="3 4" />
    </svg>
  )
}

export function EmptyState({
  title,
  description,
  action,
}: {
  title: string
  description: string
  action?: ReactNode
}) {
  return (
    <div className="max-w-md py-16">
      <EmptyFrames />
      <h2 className="mt-8 text-2xl text-ink">{title}</h2>
      <p className="mt-3 text-base text-muted">{description}</p>
      {action && <div className="mt-8">{action}</div>}
    </div>
  )
}

/**
 * Komunikat błędu nazywa problem i drogę wyjścia. Techniczne szczegóły
 * (kod, status) zostają, bo jedynym użytkownikiem jest osoba, która to pisze.
 */
export function ErrorState({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  const { headline, detail } = describeError(error)

  return (
    <div className="max-w-md py-16" role="alert">
      <p className="font-mono text-2xs uppercase text-danger">Błąd</p>
      <h2 className="mt-3 text-xl text-ink">{headline}</h2>
      <p className="mt-3 text-base text-muted">{detail}</p>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="transition-tint mt-8 rounded-md bg-raised px-4 py-2 font-mono text-xs text-ink hover:bg-line"
        >
          Spróbuj ponownie
        </button>
      )}
    </div>
  )
}

function describeError(error: unknown): { headline: string; detail: string } {
  if (error instanceof ApiError) {
    switch (error.code) {
      case 'not_found':
        return { headline: 'Nie ma takiej rzeczy', detail: error.message }
      case 'unauthorized':
        return {
          headline: 'Serwer wymaga tokenu',
          detail:
            'Na serwerze ustawiono APP_TOKEN, a ta przeglądarka go nie zna. Wpisz go przez „Token zapisu” na dole marginesu.',
        }
      case 'bad_request':
        return { headline: 'Serwer odrzucił dane', detail: error.message }
      default:
        return { headline: 'Serwer zwrócił błąd', detail: `${error.message} (${error.code})` }
    }
  }
  return {
    headline: 'Brak połączenia z API',
    detail: 'Sprawdź, czy serwer Go działa — w trybie dev powinien słuchać na porcie 8080.',
  }
}
