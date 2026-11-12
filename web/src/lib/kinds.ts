import type { ItemKind, ItemStatus } from '../api/types'

export const KIND_LABEL: Record<ItemKind, string> = {
  project: 'Projekt',
  drawing: 'Rysunek',
  print3d: 'Druk 3D',
  photo: 'Zdjęcie',
  note: 'Notatka',
  link: 'Link',
  other: 'Inne',
}

export const STATUS_LABEL: Record<ItemStatus, string> = {
  wip: 'W toku',
  done: 'Skończone',
  archived: 'Archiwum',
}

/**
 * Opis jednego pola metadanych. Schemat jest po stronie frontendu,
 * bo backend celowo trzyma metadata jako dowolny JSON — dzięki temu
 * dołożenie pola nie wymaga migracji bazy.
 */
export interface MetaField {
  key: string
  label: string
  type: 'text' | 'number' | 'url' | 'list'
  placeholder?: string
  /** Jednostka pokazywana przy wartości, np. "h", "mm", "%". */
  suffix?: string
}

/**
 * Nazwane pola metadanych zależne od rodzaju. Użytkownik wybrał gotowe pola
 * zamiast generycznego edytora klucz-wartość — spójność danych jest ważniejsza
 * niż pełna swoboda. Klucze są bez polskich znaków, bo trafiają do JSON-a
 * i mogą się kiedyś znaleźć w zapytaniach.
 */
export const META_FIELDS: Record<ItemKind, MetaField[]> = {
  project: [
    { key: 'jezyk', label: 'Język', type: 'text', placeholder: 'Go, TypeScript…' },
    { key: 'repo', label: 'Repozytorium', type: 'url', placeholder: 'https://github.com/…' },
    { key: 'stack', label: 'Stack', type: 'list', placeholder: 'chi, SQLite, Vite' },
    { key: 'licencja', label: 'Licencja', type: 'text', placeholder: 'MIT' },
  ],
  drawing: [
    { key: 'medium', label: 'Narzędzie / medium', type: 'text', placeholder: 'ołówek 2B, akwarela…' },
    { key: 'podloze', label: 'Podłoże', type: 'text', placeholder: 'papier A4, brystol…' },
    { key: 'czas_h', label: 'Czas pracy', type: 'number', suffix: 'h' },
  ],
  print3d: [
    { key: 'drukarka', label: 'Drukarka', type: 'text', placeholder: 'Prusa MK4' },
    { key: 'material', label: 'Materiał', type: 'text', placeholder: 'PETG' },
    { key: 'czas_h', label: 'Czas druku', type: 'number', suffix: 'h' },
    { key: 'warstwa_mm', label: 'Wysokość warstwy', type: 'number', suffix: 'mm' },
    { key: 'wypelnienie', label: 'Wypełnienie', type: 'number', suffix: '%' },
    { key: 'slicer', label: 'Slicer i ustawienia', type: 'text', placeholder: 'PrusaSlicer, 0.2 SPEED' },
  ],
  photo: [
    { key: 'aparat', label: 'Aparat', type: 'text', placeholder: 'Fujifilm X-T30' },
    { key: 'obiektyw', label: 'Obiektyw', type: 'text', placeholder: '35 mm f/1.4' },
    { key: 'przyslona', label: 'Przysłona', type: 'text', placeholder: 'f/2.8' },
    { key: 'naswietlanie', label: 'Czas naświetlania', type: 'text', placeholder: '1/250 s' },
    { key: 'iso', label: 'ISO', type: 'number' },
    { key: 'miejsce', label: 'Miejsce', type: 'text', placeholder: 'Kraków, Kazimierz' },
  ],
  note: [{ key: 'zrodlo', label: 'Źródło', type: 'text', placeholder: 'skąd to jest' }],
  link: [{ key: 'autor', label: 'Autor', type: 'text' }],
  other: [],
}

/**
 * Klucze metadanych, których nie obsługuje schemat danego rodzaju.
 *
 * Formularz musi je zachować przy zapisie: item mógł powstać jako `print3d`
 * z polem "drukarka", a potem zostać przełączony na `other`. Ciche skasowanie
 * takich danych przy edycji byłoby utratą informacji.
 */
export function unknownMetaKeys(kind: ItemKind, metadata: Record<string, unknown>): string[] {
  const known = new Set(META_FIELDS[kind].map((f) => f.key))
  return Object.keys(metadata).filter((key) => !known.has(key))
}
