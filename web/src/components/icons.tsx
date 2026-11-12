import type { ReactNode } from 'react'
import type { ItemKind } from '../api/types'

/**
 * Własny zestaw ikon, rysowany ręcznie w jednej siatce 24×24, jedną grubością
 * kreski i jednym zakończeniem. Bez biblioteki i bez emoji — emoji renderuje
 * się inaczej na każdym systemie i nie da się go dopasować do kroju ani koloru.
 *
 * Ikony są dekoracją przy tekstowej etykiecie, więc mają aria-hidden.
 * Gdy ikona jest jedyną treścią przycisku, przycisk dostaje aria-label.
 */
// `className?: string | undefined` zamiast samego `?` — w tsconfigu mamy
// exactOptionalPropertyTypes, więc jawne przekazanie undefined trzeba dopuścić
// w typie. Ten sam zapis powtarza się przy wszystkich opcjonalnych propsach.
function Glyph({
  children,
  className = 'size-4',
}: {
  children: ReactNode
  className?: string | undefined
}) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.5}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      className={className}
    >
      {children}
    </svg>
  )
}

type IconProps = { className?: string | undefined }

export function IconPlus({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M12 5v14M5 12h14" />
    </Glyph>
  )
}

export function IconArrowLeft({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M19 12H5M11 18l-6-6 6-6" />
    </Glyph>
  )
}

export function IconPencil({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M5 19.5l3.2-.7L19 8a1.9 1.9 0 0 0-2.7-2.7L5.7 16.1 5 19.5z" />
      <path d="M15 7.3l2.7 2.7" />
    </Glyph>
  )
}

export function IconTrash({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M4 7h16M9.5 7V4.8h5V7M6.5 7l.9 12.4h9.2L17.5 7M10.3 11v5M13.7 11v5" />
    </Glyph>
  )
}

/**
 * Znak ulubionego: otwarty pierścień jak zakreślenie kredką na arkuszu
 * stykowym. Celowo nie gwiazdka — gwiazdka to ocena, a to jest zaznaczenie.
 */
export function IconMark({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M15.8 4.6C11.2 2.9 6.1 5 4.7 9c-1.4 4 1.4 8.3 6.1 9.5 4.7 1.2 9.4-1.2 10.3-5.3.8-3.8-1.9-7.5-6.3-8.8" />
    </Glyph>
  )
}

export function IconExternal({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M14 4.5h5.5V10M19.5 4.5L11 13" />
      <path d="M18 14.5V19a1 1 0 0 1-1 1H5.5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1H10" />
    </Glyph>
  )
}

export function IconChevronDown({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M6 9.5l6 6 6-6" />
    </Glyph>
  )
}

export function IconArrowRight({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M5 12h14M13 6l6 6-6 6" />
    </Glyph>
  )
}

export function IconUpload({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M12 16V4M7.5 8.5L12 4l4.5 4.5" />
      <path d="M4.5 15v3.5a1 1 0 0 0 1 1h13a1 1 0 0 0 1-1V15" />
    </Glyph>
  )
}

export function IconDownload({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M12 4v12M7.5 11.5L12 16l4.5-4.5" />
      <path d="M4.5 15v3.5a1 1 0 0 0 1 1h13a1 1 0 0 0 1-1V15" />
    </Glyph>
  )
}

/** Znacznik okładki: kadr z zaznaczonym rogiem. */
export function IconCover({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M4.5 5.5h15a1 1 0 0 1 1 1v11a1 1 0 0 1-1 1h-15a1 1 0 0 1-1-1v-11a1 1 0 0 1 1-1z" />
      <path d="M3.5 15.5l4.5-3.5 3.5 2.7" />
      <path d="M14.5 9.5h4M16.5 7.5v4" />
    </Glyph>
  )
}

export function IconFile({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M13.5 3.5H7a1 1 0 0 0-1 1v15a1 1 0 0 0 1 1h10a1 1 0 0 0 1-1V8l-4.5-4.5z" />
      <path d="M13.5 3.5V8H18" />
    </Glyph>
  )
}

export function IconSearch({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M15.6 15.6a6 6 0 1 0-8.5-8.5 6 6 0 0 0 8.5 8.5zM15.8 15.8L20 20" />
    </Glyph>
  )
}

export function IconTag({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M11.6 3.5H19a1.5 1.5 0 0 1 1.5 1.5v7.4a1.5 1.5 0 0 1-.44 1.06l-6.1 6.1a1.5 1.5 0 0 1-2.12 0l-7.4-7.4a1.5 1.5 0 0 1 0-2.12l6.1-6.1a1.5 1.5 0 0 1 1.06-.44z" />
      <path d="M16.4 8.1a.6.6 0 1 1-1.2 0 .6.6 0 0 1 1.2 0z" />
    </Glyph>
  )
}

export function IconKey({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M10.2 13.8a4.2 4.2 0 1 1 3.1-3.1L20 17.4V20h-2.6v-1.8h-1.8v-1.8h-1.8z" />
      <path d="M8.3 9.1a.8.8 0 1 1-1.6 0 .8.8 0 0 1 1.6 0z" />
    </Glyph>
  )
}

export function IconClose({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M6 6l12 12M18 6L6 18" />
    </Glyph>
  )
}

// --- ikony rodzajów ---------------------------------------------------------
// Każda jest metaforą narzędzia, a nie literalnym obrazkiem rzeczy:
// nawiasy = kod, stalówka = rysunek, sześcian warstwowy = druk, przysłona = zdjęcie.

function IconProject({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M9 7.5L4.5 12 9 16.5M15 7.5l4.5 4.5-4.5 4.5" />
    </Glyph>
  )
}

function IconDrawing({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M12 3l6 11-6 7-6-7 6-11z" />
      <path d="M12 9.5v6" />
    </Glyph>
  )
}

// Eksportowana, bo poza rodzajem itemu oznacza też plik modelu 3D na półce.
export function IconModel({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M12 3.2l8 4.4v8.8l-8 4.4-8-4.4V7.6l8-4.4z" />
      <path d="M12 12.2l8-4.4M12 12.2v8.6M12 12.2L4 7.8" />
    </Glyph>
  )
}

function IconPhoto({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M4.5 5.5h15a1 1 0 0 1 1 1v11a1 1 0 0 1-1 1h-15a1 1 0 0 1-1-1v-11a1 1 0 0 1 1-1z" />
      <path d="M9 10.2a1.3 1.3 0 1 1-2.6 0 1.3 1.3 0 0 1 2.6 0z" />
      <path d="M3.5 16.2l4.8-3.7 3.7 2.8 2.9-2 5.6 4.2" />
    </Glyph>
  )
}

function IconNote({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M6.5 4h11a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1h-11a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1z" />
      <path d="M9 8.5h6M9 12h6M9 15.5h3.5" />
    </Glyph>
  )
}

function IconLink({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M10.2 13.8a3.8 3.8 0 0 0 5.4 0l2.5-2.5a3.8 3.8 0 1 0-5.4-5.4l-1.2 1.2" />
      <path d="M13.8 10.2a3.8 3.8 0 0 0-5.4 0l-2.5 2.5a3.8 3.8 0 1 0 5.4 5.4l1.2-1.2" />
    </Glyph>
  )
}

function IconOther({ className }: IconProps) {
  return (
    <Glyph className={className}>
      <path d="M12 4.5v15M4.5 12h15M6.7 6.7l10.6 10.6M17.3 6.7L6.7 17.3" />
    </Glyph>
  )
}

const KIND_ICONS: Record<ItemKind, (props: IconProps) => ReactNode> = {
  project: IconProject,
  drawing: IconDrawing,
  print3d: IconModel,
  photo: IconPhoto,
  note: IconNote,
  link: IconLink,
  other: IconOther,
}

export function KindIcon({
  kind,
  className,
}: {
  kind: ItemKind
  className?: string | undefined
}) {
  const Icon = KIND_ICONS[kind]
  return <Icon className={className} />
}
