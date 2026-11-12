import { useEffect, useId, useRef, useState } from 'react'

import { getToken, setToken, TOKEN_OPEN_EVENT, TOKEN_REQUIRED_EVENT } from '../api/client'
import { Button } from './ui'

type Reason = 'manual' | 'rejected'

/**
 * Okno do wpisania APP_TOKEN.
 *
 * Otwiera się samo, gdy serwer odrzuci zapis kodem `unauthorized`, albo na
 * żądanie z marginesu. Token zostaje w localStorage tej przeglądarki —
 * apiFetch i uploadFiles dokleją go do każdego zapisu.
 *
 * Stan otwarcia trzymamy jako zwykły boolean i synchronizujemy z <dialog>
 * w efekcie zależnym właśnie od niego. To ważne: efekt zależny od funkcji
 * tworzonej przy każdym renderze otwierałby okno z powrotem tuż po zamknięciu
 * (ten błąd był już raz w Lightboxie).
 */
export function TokenDialog() {
  const dialogRef = useRef<HTMLDialogElement>(null)
  const [open, setOpen] = useState(false)
  const [reason, setReason] = useState<Reason>('manual')
  const [value, setValue] = useState('')
  const [hasToken, setHasToken] = useState(false)
  const inputId = useId()

  useEffect(() => {
    const onRequired = () => {
      setReason('rejected')
      setOpen(true)
    }
    const onManual = () => {
      setReason('manual')
      setOpen(true)
    }
    window.addEventListener(TOKEN_REQUIRED_EVENT, onRequired)
    window.addEventListener(TOKEN_OPEN_EVENT, onManual)
    return () => {
      window.removeEventListener(TOKEN_REQUIRED_EVENT, onRequired)
      window.removeEventListener(TOKEN_OPEN_EVENT, onManual)
    }
  }, [])

  useEffect(() => {
    const dialog = dialogRef.current
    if (!dialog) return

    if (open) {
      const current = getToken()
      setValue(current)
      setHasToken(current !== '')
      if (!dialog.open) dialog.showModal()
    } else if (dialog.open) {
      dialog.close()
    }
  }, [open])

  function save(next: string) {
    setToken(next.trim())
    setOpen(false)
  }

  return (
    <dialog
      ref={dialogRef}
      // Zamknięcie natywne (np. Escape tam, gdzie przeglądarka je obsługuje)
      // musi zsynchronizować stan — inaczej następne otwarcie nic by nie zrobiło.
      onClose={() => setOpen(false)}
      onKeyDown={(event) => {
        // Escape obsługujemy sami: nie każdy silnik zgłasza natywne `cancel`.
        if (event.key === 'Escape') {
          event.preventDefault()
          setOpen(false)
        }
      }}
      aria-labelledby={`${inputId}-title`}
      // m-auto przywraca wyśrodkowanie: przeglądarka centruje modal przez
      // margin: auto, a preflight Tailwinda zeruje margines każdemu elementowi.
      className="m-auto w-[calc(100%-2rem)] max-w-md rounded-lg bg-surface p-0 text-ink backdrop:bg-canvas/80"
    >
      <form
        className="p-6"
        onSubmit={(event) => {
          event.preventDefault()
          save(value)
        }}
      >
        <h2 id={`${inputId}-title`} className="text-xl text-ink">
          Token zapisu
        </h2>

        <p className="mt-3 text-sm text-muted">
          {reason === 'rejected'
            ? 'Serwer odrzucił zapis: ma ustawiony APP_TOKEN, a ta przeglądarka nie zna właściwej wartości. Wpisz go i powtórz operację.'
            : 'Potrzebny tylko wtedy, gdy na serwerze ustawiono APP_TOKEN. Odczyt działa bez niego.'}
        </p>

        <label htmlFor={inputId} className="mt-6 block font-mono text-2xs uppercase text-faint">
          APP_TOKEN
        </label>
        <input
          id={inputId}
          type="password"
          autoComplete="off"
          spellCheck={false}
          value={value}
          onChange={(event) => setValue(event.target.value)}
          autoFocus
          className="transition-tint mt-2 w-full rounded-md bg-raised px-3 py-2 font-mono text-base text-ink placeholder:text-faint"
          placeholder="wartość z pliku .env"
        />
        <p className="mt-1.5 text-xs text-faint">
          Zostaje w tej przeglądarce (localStorage) i trafia tylko do własnego serwera.
        </p>

        <div className="mt-8 flex flex-wrap items-center gap-2">
          <Button type="submit" variant="primary" disabled={value.trim() === ''}>
            Zapisz token
          </Button>
          {hasToken && (
            <Button type="button" variant="danger" onClick={() => save('')}>
              Usuń zapisany
            </Button>
          )}
          <Button type="button" variant="ghost" onClick={() => setOpen(false)} className="ml-auto">
            Anuluj
          </Button>
        </div>
      </form>
    </dialog>
  )
}
