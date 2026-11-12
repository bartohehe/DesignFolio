import { useRef, useState } from 'react'

import { useArrangeAssets, useDeleteAsset, useUploadAssets } from '../api/queries'
import type { Asset } from '../api/types'
import { formatBytes } from '../lib/format'
import { assetUrl } from '../lib/urls'
import { Lightbox } from './Lightbox'
import { Button } from './ui'
import {
  IconArrowLeft,
  IconArrowRight,
  IconCover,
  IconFile,
  IconModel,
  IconTrash,
  IconUpload,
} from './icons'

/**
 * Półka z plikami itemu: wrzucanie, kolejność, okładka, usuwanie i podgląd.
 *
 * Wszystko dzieje się na stronie rzeczy, a nie w formularzu edycji —
 * nowy item nie ma jeszcze identyfikatora, więc formularz i tak nie miałby
 * gdzie tych plików wysłać.
 */
export function AssetShelf({ itemId, assets }: { itemId: string; assets: Asset[] }) {
  const [dragActive, setDragActive] = useState(false)
  const [progress, setProgress] = useState<number | null>(null)
  const [lightboxIndex, setLightboxIndex] = useState<number | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const upload = useUploadAssets(itemId)
  const arrange = useArrangeAssets(itemId)
  const remove = useDeleteAsset()

  const images = assets.filter((a) => a.kind === 'image')

  function send(files: FileList | File[] | null) {
    const list = Array.from(files ?? [])
    if (list.length === 0) return

    setProgress(0)
    upload.mutate(
      { files: list, onProgress: setProgress },
      { onSettled: () => setProgress(null) },
    )
  }

  function move(assetId: string, direction: -1 | 1) {
    const current = assets.map((a) => a.id)
    const from = current.indexOf(assetId)
    const to = from + direction
    if (from < 0 || to < 0 || to >= current.length) return

    // Zamiana miejscami — czytelniejsza niż splice w dwóch krokach.
    ;[current[from], current[to]] = [current[to]!, current[from]!]

    const cover = assets.find((a) => a.is_cover)?.id ?? null
    arrange.mutate({ order: current, cover })
  }

  /** Przeciągnięcie: plik `draggedId` ląduje na miejscu pliku `targetId`. */
  function reorder(draggedId: string, targetId: string) {
    if (draggedId === targetId) return

    const order = assets.map((a) => a.id)
    const from = order.indexOf(draggedId)
    const to = order.indexOf(targetId)
    if (from < 0 || to < 0) return

    order.splice(from, 1)
    order.splice(to, 0, draggedId)

    const cover = assets.find((a) => a.is_cover)?.id ?? null
    arrange.mutate({ order, cover })
  }

  function setCover(assetId: string) {
    const isAlreadyCover = assets.find((a) => a.id === assetId)?.is_cover
    arrange.mutate({
      order: assets.map((a) => a.id),
      // Ponowne kliknięcie zdejmuje okładkę — wtedy kafelek wraca
      // do pierwszego zdjęcia z listy.
      cover: isAlreadyCover ? null : assetId,
    })
  }

  const busy = upload.isPending || arrange.isPending || remove.isPending

  return (
    <section
      // Sekcja przyjmuje wyłącznie pliki z dysku. Przeciąganie kafelków
      // w obrębie półki obsługują same kafelki — rozróżniamy je po typie
      // danych w dataTransfer, inaczej każde przestawienie podświetlałoby
      // całą sekcję jak przy uploadzie.
      onDragOver={(event) => {
        if (!event.dataTransfer.types.includes('Files')) return
        event.preventDefault()
        setDragActive(true)
      }}
      onDragLeave={(event) => {
        // Zdarzenie leci też przy przechodzeniu nad dziećmi — sprawdzamy,
        // czy kursor naprawdę opuścił obszar sekcji.
        if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
          setDragActive(false)
        }
      }}
      onDrop={(event) => {
        if (!event.dataTransfer.types.includes('Files')) return
        event.preventDefault()
        setDragActive(false)
        send(event.dataTransfer.files)
      }}
      className={`transition-tint rounded-lg ${dragActive ? 'bg-surface' : ''}`}
    >
      <header className="flex items-center justify-between gap-4">
        <h2 className="font-mono text-2xs uppercase text-faint">
          Pliki{assets.length > 0 && <span className="tabular-nums"> · {assets.length}</span>}
        </h2>

        <Button variant="ghost" onClick={() => inputRef.current?.click()} disabled={busy}>
          <IconUpload />
          Dodaj pliki
        </Button>

        <input
          ref={inputRef}
          type="file"
          multiple
          // sr-only chowa pole wizualnie, ale zostaje ono w kolejności Tab —
          // bez etykiety czytnik ekranu zapowiedziałby "pole pliku" i nic więcej.
          aria-label="Wybierz pliki do wysłania"
          className="sr-only"
          onChange={(event) => {
            send(event.target.files)
            // Wyczyszczenie pozwala wybrać ten sam plik ponownie —
            // bez tego onChange nie odpaliłby się drugi raz.
            event.target.value = ''
          }}
        />
      </header>

      {progress !== null && (
        <div className="mt-4" role="status" aria-live="polite">
          <div className="h-0.5 w-full overflow-hidden rounded-sm bg-raised">
            {/* scaleX zamiast width: animowanie szerokości przelicza układ
                przy każdej klatce, transform idzie na GPU. Zasada projektu:
                ruch wyłącznie przez transform i opacity. */}
            <div
              className="transition-ui h-full origin-left bg-accent"
              style={{ transform: `scaleX(${progress / 100})` }}
            />
          </div>
          <p className="mt-2 font-mono text-2xs tabular-nums text-faint">
            wysyłam… {progress}%
          </p>
        </div>
      )}

      {upload.isError && (
        <p className="mt-4 text-sm text-danger" role="alert">
          Nie udało się wysłać: {upload.error.message}
        </p>
      )}

      {upload.data && upload.data.skipped.length > 0 && (
        <ul className="mt-4 space-y-1 text-sm text-danger" role="alert">
          {upload.data.skipped.map((s) => (
            <li key={s.filename}>
              <span className="font-mono text-xs">{s.filename}</span> — {s.reason}
            </li>
          ))}
        </ul>
      )}

      {assets.length === 0 ? (
        <DropHint active={dragActive} onPick={() => inputRef.current?.click()} />
      ) : (
        <>
          <ul className="mt-5 grid grid-cols-2 gap-4 sm:grid-cols-3 xl:grid-cols-4">
            {assets.map((asset, index) => (
              <AssetTile
                key={asset.id}
                asset={asset}
                first={index === 0}
                last={index === assets.length - 1}
                busy={busy}
                onOpen={
                  asset.kind === 'image'
                    ? () => setLightboxIndex(images.findIndex((a) => a.id === asset.id))
                    : undefined
                }
                onMove={(direction) => move(asset.id, direction)}
                onDropAsset={(draggedId) => reorder(draggedId, asset.id)}
                onCover={() => setCover(asset.id)}
                onDelete={() => remove.mutate(asset.id)}
              />
            ))}
          </ul>
          {assets.length > 1 && (
            <p className="mt-4 font-mono text-2xs text-faint">
              przeciągnij kafelek, żeby zmienić kolejność · strzałki robią to samo z klawiatury
            </p>
          )}
        </>
      )}

      {lightboxIndex !== null && images.length > 0 && (
        <Lightbox
          assets={images}
          index={lightboxIndex}
          onClose={() => setLightboxIndex(null)}
          onNavigate={setLightboxIndex}
        />
      )}
    </section>
  )
}

/**
 * Własny typ danych przeciągania. Po nim kafelek odróżnia przestawianie
 * plików na półce od pliku przeciąganego z pulpitu ("Files").
 * Małe litery, bo przeglądarki i tak je tak zapisują w dataTransfer.types.
 */
const ASSET_DRAG_TYPE = 'application/x-designfolio-asset'

function DropHint({ active, onPick }: { active: boolean; onPick: () => void }) {
  return (
    <button
      type="button"
      onClick={onPick}
      className={`transition-tint mt-5 flex w-full flex-col items-center gap-3 rounded-lg border border-dashed px-6 py-12 ${
        active ? 'border-accent text-accent' : 'border-line text-faint hover:text-muted'
      }`}
    >
      <IconUpload className="size-6" />
      <span className="text-sm">
        Przeciągnij tu zdjęcia, modele albo dokumenty — albo kliknij, żeby wybrać
      </span>
    </button>
  )
}

function AssetTile({
  asset,
  first,
  last,
  busy,
  onOpen,
  onMove,
  onDropAsset,
  onCover,
  onDelete,
}: {
  asset: Asset
  first: boolean
  last: boolean
  busy: boolean
  onOpen?: (() => void) | undefined
  onMove: (direction: -1 | 1) => void
  onDropAsset: (draggedId: string) => void
  onCover: () => void
  onDelete: () => void
}) {
  const [confirming, setConfirming] = useState(false)
  const [dragging, setDragging] = useState(false)
  const [dropTarget, setDropTarget] = useState(false)

  return (
    <li
      // Natywne HTML5 drag & drop — bez biblioteki. Przeciąganie myszą nie
      // jest dostępne z klawiatury, dlatego strzałki niżej zostają.
      draggable={!busy}
      onDragStart={(event) => {
        event.dataTransfer.setData(ASSET_DRAG_TYPE, asset.id)
        event.dataTransfer.effectAllowed = 'move'
        setDragging(true)
      }}
      onDragEnd={() => setDragging(false)}
      onDragOver={(event) => {
        if (!event.dataTransfer.types.includes(ASSET_DRAG_TYPE)) return
        event.preventDefault() // bez tego przeglądarka nie pozwoli upuścić
        event.dataTransfer.dropEffect = 'move'
        setDropTarget(true)
      }}
      onDragLeave={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
          setDropTarget(false)
        }
      }}
      onDrop={(event) => {
        if (!event.dataTransfer.types.includes(ASSET_DRAG_TYPE)) return
        event.preventDefault()
        // stopPropagation: sekcja nad nami też nasłuchuje upuszczenia
        // (dla plików z dysku) i nie powinna tego zdarzenia dostać.
        event.stopPropagation()
        setDropTarget(false)
        onDropAsset(event.dataTransfer.getData(ASSET_DRAG_TYPE))
      }}
      className={[
        'transition-ui group relative cursor-grab active:cursor-grabbing',
        dragging ? 'opacity-40' : '',
        dropTarget ? '-translate-y-1' : '',
      ].join(' ')}
    >
      <div
        className={`overflow-hidden rounded-md bg-surface ${dropTarget ? 'outline-2 outline-offset-2 outline-accent' : ''}`}
      >
        {asset.has_thumb && onOpen ? (
          <button
            type="button"
            onClick={onOpen}
            className="block aspect-square w-full"
          >
            {/* draggable={false}: obrazki i linki przeglądarka przeciąga
                po swojemu (jako adres URL). Wyłączamy to, żeby źródłem
                przeciągania był zawsze cały kafelek. */}
            <img
              src={assetUrl(asset.id, 'thumb')}
              alt={asset.filename}
              loading="lazy"
              draggable={false}
              className="transition-ui size-full object-cover group-hover:scale-[1.02]"
            />
            <span className="sr-only">Otwórz podgląd</span>
          </button>
        ) : (
          <a
            href={assetUrl(asset.id, 'original')}
            download={asset.filename}
            draggable={false}
            className="flex aspect-square w-full flex-col items-center justify-center gap-2 text-faint"
          >
            {asset.kind === 'model' ? (
              <IconModel className="size-8" />
            ) : (
              <IconFile className="size-8" />
            )}
            <span className="font-mono text-2xs uppercase">
              {asset.filename.split('.').pop()}
            </span>
          </a>
        )}
      </div>

      <p className="mt-2 truncate font-mono text-2xs text-faint" title={asset.filename}>
        {asset.is_cover && <span className="text-accent">okładka · </span>}
        {asset.filename}
      </p>
      <p className="font-mono text-2xs tabular-nums text-faint/80">{formatBytes(asset.size)}</p>

      {/* Pasek narzędzi jest zawsze w drzewie i widoczny przy focusie
          klawiatury — samo pokazywanie na hover odcięłoby go od klawiatury. */}
      <div className="mt-2 flex items-center gap-0.5 opacity-0 transition-tint focus-within:opacity-100 group-hover:opacity-100">
        {asset.kind === 'image' && (
          <TileButton
            label={asset.is_cover ? 'Zdejmij okładkę' : 'Ustaw jako okładkę'}
            active={asset.is_cover}
            disabled={busy}
            onClick={onCover}
          >
            <IconCover />
          </TileButton>
        )}
        <TileButton label="Przesuń wcześniej" disabled={busy || first} onClick={() => onMove(-1)}>
          <IconArrowLeft />
        </TileButton>
        <TileButton label="Przesuń później" disabled={busy || last} onClick={() => onMove(1)}>
          <IconArrowRight />
        </TileButton>

        {confirming ? (
          <span className="ml-auto flex items-center gap-1">
            <button
              type="button"
              onClick={onDelete}
              disabled={busy}
              className="transition-tint rounded-sm px-2 py-1 text-2xs text-danger hover:bg-danger/12"
            >
              Usuń
            </button>
            <button
              type="button"
              onClick={() => setConfirming(false)}
              className="transition-tint rounded-sm px-2 py-1 text-2xs text-faint hover:text-ink"
            >
              Nie
            </button>
          </span>
        ) : (
          <span className="ml-auto">
            <TileButton label="Usuń plik" disabled={busy} onClick={() => setConfirming(true)}>
              <IconTrash />
            </TileButton>
          </span>
        )}
      </div>
    </li>
  )
}

function TileButton({
  label,
  children,
  onClick,
  disabled,
  active = false,
}: {
  label: string
  children: React.ReactNode
  onClick: () => void
  disabled: boolean
  active?: boolean
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      title={label}
      className={`transition-tint rounded-sm p-1.5 disabled:opacity-30 ${
        active ? 'text-accent' : 'text-faint hover:text-ink'
      }`}
    >
      {children}
      <span className="sr-only">{label}</span>
    </button>
  )
}
