import { Fragment } from 'react'

import { splitMatches } from '../lib/text'

/**
 * Podświetla w tekście fragmenty pasujące do frazy wyszukiwania.
 *
 * Używamy <mark>, a nie <span> z kolorem — to element o znaczeniu semantycznym,
 * który czytniki ekranu potrafią zapowiedzieć jako wyróżnienie.
 */
export function Highlight({ text, query }: { text: string; query: string }) {
  if (!query.trim()) return <>{text}</>

  return (
    <>
      {splitMatches(text, query).map((part, index) => (
        <Fragment key={index}>
          {part.match ? (
            <mark className="bg-accent/22 text-inherit">{part.text}</mark>
          ) : (
            part.text
          )}
        </Fragment>
      ))}
    </>
  )
}
