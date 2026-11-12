/**
 * Zwijanie znaków diakrytycznych — odpowiednik model.FoldDiacritics z Go.
 *
 * Musi dawać ten sam wynik co backend, bo służy do podświetlania trafień:
 * serwer znalazł "słuchawki" po wpisaniu "sluchawki", więc przeglądarka
 * musi umieć wskazać to samo miejsce w tekście.
 */
const DIACRITICS: Record<string, string> = {
  ą: 'a', ć: 'c', ę: 'e', ł: 'l', ń: 'n', ó: 'o', ś: 's', ź: 'z', ż: 'z',
  á: 'a', à: 'a', ä: 'a', â: 'a', å: 'a', ã: 'a',
  é: 'e', è: 'e', ë: 'e', ê: 'e',
  í: 'i', ì: 'i', ï: 'i', î: 'i',
  ò: 'o', ö: 'o', ô: 'o', õ: 'o', ø: 'o',
  ú: 'u', ù: 'u', ü: 'u', û: 'u',
  ý: 'y', ÿ: 'y', ñ: 'n', ç: 'c',
  š: 's', č: 'c', ř: 'r', ž: 'z', ě: 'e', ů: 'u', ď: 'd', ť: 't',
  ß: 'ss', æ: 'ae', œ: 'oe', đ: 'd',
}

export function foldDiacritics(input: string): string {
  let out = ''
  for (const char of input.toLowerCase()) {
    out += DIACRITICS[char] ?? char
  }
  return out
}

/** Fragment tekstu z informacją, czy jest trafieniem wyszukiwania. */
export interface TextPart {
  text: string
  match: boolean
}

/**
 * Dzieli tekst na fragmenty pasujące i niepasujące do zapytania.
 *
 * Porównujemy na zwiniętych wersjach, ale wycinamy z ORYGINAŁU —
 * dzięki temu podświetlone "sło" w słowie "słuchawki" zachowa polskie znaki.
 * Wymaga, żeby zwijanie nie zmieniało długości tekstu, więc znaki mapowane
 * na dwie litery (ß → ss) są tu pomijane.
 */
export function splitMatches(text: string, query: string): TextPart[] {
  const words = foldDiacritics(query)
    .split(/[^\p{L}\p{N}]+/u)
    .filter((w) => w.length > 1)

  if (words.length === 0) return [{ text, match: false }]

  const folded = foldDiacritics(text)
  if (folded.length !== text.length) return [{ text, match: false }]

  // Zaznaczamy pozycje objęte dowolnym trafieniem, potem sklejamy je w bloki.
  const hit = new Array<boolean>(text.length).fill(false)
  for (const word of words) {
    let from = 0
    for (;;) {
      const at = folded.indexOf(word, from)
      if (at === -1) break
      for (let i = at; i < at + word.length; i++) hit[i] = true
      from = at + word.length
    }
  }

  const parts: TextPart[] = []
  let start = 0
  for (let i = 1; i <= text.length; i++) {
    if (i === text.length || hit[i] !== hit[start]) {
      parts.push({ text: text.slice(start, i), match: hit[start] === true })
      start = i
    }
  }
  return parts
}
