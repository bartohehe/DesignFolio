import { useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'

import { IconClose, IconSearch } from './icons'

/** Ile milisekund czekamy po ostatnim wciśnięciu klawisza. */
const DEBOUNCE_MS = 220

/**
 * Pole wyszukiwania. Fraza trafia do adresu URL, tak samo jak filtry —
 * dzięki temu wynik da się odświeżyć, cofnąć i zapisać w zakładkach.
 *
 * Wpisywany tekst trzymamy w lokalnym stanie i przepisujemy do URL-a
 * z opóźnieniem: bez tego każda litera powodowałaby wpis w historii
 * przeglądarki i osobne zapytanie do API.
 */
export function SearchBox() {
  const [params, setParams] = useSearchParams()
  const urlQuery = params.get('q') ?? ''

  const [draft, setDraft] = useState(urlQuery)
  const inputRef = useRef<HTMLInputElement>(null)

  // Synchronizacja w drugą stronę: gdy adres zmieni się z zewnątrz
  // (przycisk Wstecz, kliknięcie w tag), pole ma to odzwierciedlić.
  useEffect(() => {
    setDraft(urlQuery)
  }, [urlQuery])

  useEffect(() => {
    if (draft === urlQuery) return

    const timer = window.setTimeout(() => {
      const updated = new URLSearchParams(params)
      if (draft.trim() === '') updated.delete('q')
      else updated.set('q', draft)
      setParams(updated, { replace: true })
    }, DEBOUNCE_MS)

    return () => window.clearTimeout(timer)
  }, [draft, urlQuery, params, setParams])

  // Ukośnik ustawia kursor w polu — skrót, który każdy zna z innych narzędzi.
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key !== '/' || event.ctrlKey || event.metaKey || event.altKey) return

      const active = document.activeElement
      const typing =
        active instanceof HTMLInputElement ||
        active instanceof HTMLTextAreaElement ||
        (active instanceof HTMLElement && active.isContentEditable)
      if (typing) return

      event.preventDefault()
      inputRef.current?.focus()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  return (
    <div className="relative">
      <label htmlFor="szukaj" className="sr-only">
        Szukaj w katalogu
      </label>
      <IconSearch className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-faint" />
      <input
        ref={inputRef}
        id="szukaj"
        type="search"
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === 'Escape') setDraft('')
        }}
        placeholder="Szukaj…"
        // Natywny krzyżyk w polu typu search wygląda inaczej w każdej
        // przeglądarce, więc go chowamy i rysujemy własny.
        className="transition-tint w-full rounded-md bg-surface py-2 pl-9 pr-9 text-sm text-ink placeholder:text-faint [&::-webkit-search-cancel-button]:hidden"
      />
      {draft !== '' && (
        <button
          type="button"
          onClick={() => setDraft('')}
          className="transition-tint absolute right-2 top-1/2 -translate-y-1/2 rounded-sm p-1 text-faint hover:text-ink"
        >
          <IconClose className="size-3.5" />
          <span className="sr-only">Wyczyść wyszukiwanie</span>
        </button>
      )}
    </div>
  )
}
