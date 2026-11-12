import { useEffect, useRef } from 'react'

import type { Asset } from '../api/types'
import { formatBytes } from '../lib/format'
import { assetUrl } from '../lib/urls'
import { IconArrowLeft, IconArrowRight, IconClose, IconDownload } from './icons'

/**
 * Podgląd pełnoekranowy.
 *
 * Oparty na natywnym <dialog> z showModal(): przeglądarka daje za darmo
 * pułapkę focusa, zamykanie Escape'em i wyłączenie tła dla czytników ekranu.
 * Własna implementacja modala musiałaby to wszystko odtworzyć — i zwykle
 * odtwarza źle.
 */
export function Lightbox({
  assets,
  index,
  onClose,
  onNavigate,
}: {
  assets: Asset[]
  index: number
  onClose: () => void
  onNavigate: (index: number) => void
}) {
  const dialogRef = useRef<HTMLDialogElement>(null)
  const asset = assets[index]

  // Callback trzymamy w ref, żeby efekt niżej mógł mieć PUSTĄ listę zależności.
  //
  // To nie jest ozdobnik: onClose przychodzi jako nowa funkcja przy każdym
  // renderze rodzica. Gdyby efekt od niej zależał, wyglądałoby to tak:
  // Escape zamyka dialog → stan się zmienia → render → efekt leci od nowa
  // → widzi zamknięty dialog → otwiera go z powrotem. Modal nie dałby się zamknąć.
  const onCloseRef = useRef(onClose)
  onCloseRef.current = onClose

  useEffect(() => {
    const dialog = dialogRef.current
    if (!dialog) return

    dialog.showModal()

    // Natywne zamknięcie (Escape) musi zsynchronizować stan Reacta.
    const handleClose = () => onCloseRef.current()
    dialog.addEventListener('close', handleClose)

    return () => {
      dialog.removeEventListener('close', handleClose)
      if (dialog.open) dialog.close()
    }
  }, [])

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'ArrowRight') onNavigate((index + 1) % assets.length)
      if (event.key === 'ArrowLeft') onNavigate((index - 1 + assets.length) % assets.length)
      // Escape obsługujemy sami, mimo że <dialog> ma to robić natywnie.
      // Nie wszystkie silniki uruchamiają close watcher (sprawdzone —
      // zdarzenie `cancel` potrafi się nie pojawić), a zamknięcie podglądu
      // Escape'em to rzecz, która musi działać zawsze.
      if (event.key === 'Escape') {
        event.preventDefault()
        dialogRef.current?.close()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [index, assets.length, onNavigate])

  if (!asset) return null

  return (
    <dialog ref={dialogRef} className="max-h-dvh max-w-full bg-transparent p-0 backdrop:bg-canvas/92">
      {/* Kliknięcie w puste miejsce zamyka, kliknięcie w treść nie.
          Sprawdzamy currentTarget, bo dialog jest rozciągnięty na cały ekran
          i sam nigdy nie bywa celem kliknięcia. */}
      <div
        onClick={(event) => {
          if (event.target === event.currentTarget) dialogRef.current?.close()
        }}
        className="flex h-dvh w-dvw flex-col"
      >
        <header className="flex items-center justify-between gap-4 px-6 py-4">
          <p className="min-w-0 truncate font-mono text-xs text-muted">
            {asset.filename}
            <span className="text-faint">
              {' · '}
              {formatBytes(asset.size)}
              {asset.width && asset.height ? ` · ${asset.width}×${asset.height}` : ''}
            </span>
          </p>

          <div className="flex shrink-0 items-center gap-1">
            <a
              href={assetUrl(asset.id, 'original')}
              download={asset.filename}
              className="transition-tint rounded-md p-2 text-muted hover:bg-surface hover:text-ink"
            >
              <IconDownload className="size-5" />
              <span className="sr-only">Pobierz oryginał</span>
            </a>
            <button
              type="button"
              onClick={() => dialogRef.current?.close()}
              className="transition-tint rounded-md p-2 text-muted hover:bg-surface hover:text-ink"
            >
              <IconClose className="size-5" />
              <span className="sr-only">Zamknij podgląd</span>
            </button>
          </div>
        </header>

        <div
          onClick={(event) => {
            if (event.target === event.currentTarget) dialogRef.current?.close()
          }}
          className="flex min-h-0 flex-1 items-center gap-2 px-4 pb-6"
        >
          {assets.length > 1 && (
            <NavButton
              direction="left"
              onClick={() => onNavigate((index - 1 + assets.length) % assets.length)}
            />
          )}

          <img
            src={assetUrl(asset.id, 'preview')}
            alt={asset.filename}
            className="mx-auto max-h-full min-h-0 max-w-full object-contain"
          />

          {assets.length > 1 && (
            <NavButton direction="right" onClick={() => onNavigate((index + 1) % assets.length)} />
          )}
        </div>

        {assets.length > 1 && (
          <p className="pb-6 text-center font-mono text-2xs tabular-nums text-faint">
            {index + 1} / {assets.length}
          </p>
        )}
      </div>
    </dialog>
  )
}

function NavButton({ direction, onClick }: { direction: 'left' | 'right'; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="transition-tint shrink-0 rounded-md p-3 text-muted hover:bg-surface hover:text-ink"
    >
      {direction === 'left' ? (
        <IconArrowLeft className="size-6" />
      ) : (
        <IconArrowRight className="size-6" />
      )}
      <span className="sr-only">{direction === 'left' ? 'Poprzedni plik' : 'Następny plik'}</span>
    </button>
  )
}
