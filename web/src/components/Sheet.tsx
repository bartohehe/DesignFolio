import { useEffect, useRef, useState } from 'react'

import type { Item } from '../api/types'
import { Frame } from './Frame'

/** Docelowa szerokość kolumny; z niej wynika, ile ich się zmieści. */
const TARGET_COLUMN_PX = 340
const MAX_COLUMNS = 4

/**
 * Arkusz — masonry o kolumnach wyliczanych z rzeczywistej szerokości
 * kontenera, a nie z szerokości okna. To istotne, bo po lewej stoi indeks,
 * który zabiera miejsce tylko na szerokich ekranach.
 *
 * Dlaczego nie CSS `columns`? Kolumny CSS układają treść od góry do dołu:
 * przy sortowaniu od najnowszych trzy pierwsze rzeczy wylądowałyby jedna
 * pod drugą w pierwszej kolumnie. Rozdanie po kolei (i % liczba kolumn)
 * daje poprawną kolejność czytania w pierwszym rzędzie.
 */
export function Sheet({ items, query = '' }: { items: Item[]; query?: string }) {
  const containerRef = useRef<HTMLDivElement>(null)
  const columnCount = useColumnCount(containerRef)

  const columns: Item[][] = Array.from({ length: columnCount }, () => [])
  items.forEach((item, index) => {
    columns[index % columnCount]!.push(item)
  })

  return (
    <div ref={containerRef} className="flex items-start gap-8">
      {columns.map((column, index) => (
        <div key={index} className="flex min-w-0 flex-1 flex-col gap-8">
          {column.map((item) => (
            <Frame key={item.id} item={item} query={query} />
          ))}
        </div>
      ))}
    </div>
  )
}

function useColumnCount(ref: React.RefObject<HTMLElement | null>): number {
  const [count, setCount] = useState(1)

  useEffect(() => {
    const element = ref.current
    if (!element) return

    // ResizeObserver reaguje też wtedy, gdy zmieni się sam kontener
    // (np. po zwinięciu indeksu), a nie tylko przy zmianie rozmiaru okna.
    const observer = new ResizeObserver(([entry]) => {
      if (!entry) return
      const width = entry.contentRect.width
      const fits = Math.floor(width / TARGET_COLUMN_PX)
      setCount(Math.min(Math.max(fits, 1), MAX_COLUMNS))
    })

    observer.observe(element)
    return () => observer.disconnect()
  }, [ref])

  return count
}
