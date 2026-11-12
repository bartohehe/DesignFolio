/** Formatowanie wartości do wyświetlenia. Wszystko po polsku. */

const dateFormatter = new Intl.DateTimeFormat('pl-PL', {
  day: 'numeric',
  month: 'long',
  year: 'numeric',
})

const monthFormatter = new Intl.DateTimeFormat('pl-PL', {
  month: 'long',
  year: 'numeric',
})

export function formatDate(iso: string | null): string {
  if (!iso) return '—'
  return dateFormatter.format(new Date(iso))
}

export function formatMonth(iso: string): string {
  return monthFormatter.format(new Date(iso))
}

/** "2025-11-02T18:00:00Z" → "2025-11-02" dla <input type="date">. */
export function toDateInput(iso: string | null): string {
  if (!iso) return ''
  return new Date(iso).toISOString().slice(0, 10)
}

/**
 * "2025-11-02" → pełny znacznik czasu w UTC.
 * Bierzemy południe zamiast północy, żeby przesunięcie strefy czasowej
 * nie przerzuciło daty na sąsiedni dzień.
 */
export function fromDateInput(value: string): string | null {
  if (!value) return null
  return new Date(`${value}T12:00:00Z`).toISOString()
}

/**
 * Skraca tekst do pełnego słowa. Markdown zostaje spłaszczony pobieżnie —
 * na kafelku i tak chodzi tylko o zorientowanie się, co to za rzecz.
 */
export function excerpt(markdown: string, maxChars: number): string {
  const flat = markdown
    .replace(/```[\s\S]*?```/g, ' ') // bloki kodu
    .replace(/!\[[^\]]*\]\([^)]*\)/g, ' ') // obrazki
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1') // linki → sam tekst
    .replace(/[#>*_`~-]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()

  if (flat.length <= maxChars) return flat

  const cut = flat.slice(0, maxChars)
  const lastSpace = cut.lastIndexOf(' ')
  return `${cut.slice(0, lastSpace > 0 ? lastSpace : maxChars).trimEnd()}…`
}

/**
 * Odmiana liczebnika po polsku.
 *
 * Polski ma trzy formy, a nie dwie jak angielski:
 *   1 rzecz · 2 rzeczy · 5 rzeczy
 *   1 rok   · 2 lata   · 5 lat
 *
 * Forma "few" (2–4) nie dotyczy końcówek 12–14: jest "22 lata", ale "12 lat".
 */
export function plural(count: number, one: string, few: string, many: string): string {
  const abs = Math.abs(count)
  const lastDigit = abs % 10
  const lastTwo = abs % 100

  if (abs === 1) return `${count} ${one}`
  if (lastDigit >= 2 && lastDigit <= 4 && !(lastTwo >= 12 && lastTwo <= 14)) {
    return `${count} ${few}`
  }
  return `${count} ${many}`
}

/** Rozmiar pliku po ludzku: 1536 → "1,5 kB". */
export function formatBytes(bytes: number): string {
  const units = ['B', 'kB', 'MB', 'GB']
  let value = bytes
  let unit = 0

  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit++
  }
  // Bajty bez części dziesiętnej — "512,0 B" wygląda dziwnie.
  const decimals = unit === 0 ? 0 : value < 10 ? 1 : 0
  return `${value.toLocaleString('pl-PL', { maximumFractionDigits: decimals })} ${units[unit]}`
}

/** Rok z happened_at, a gdy go nie ma — z created_at. Tak samo sortuje backend. */
export function effectiveYear(happenedAt: string | null, createdAt: string): number {
  return new Date(happenedAt ?? createdAt).getFullYear()
}
