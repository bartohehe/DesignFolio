import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

/**
 * Opis itemu renderowany z markdowna.
 *
 * Bez rehype-raw: react-markdown domyślnie nie przepuszcza surowego HTML-a,
 * więc wklejony gdzieś <script> zostanie zwykłym tekstem. Przy jednym
 * użytkowniku to nadmiar ostrożności, ale nie kosztuje nic.
 *
 * Style opisane są wariantami na kontenerze zamiast wtyczki typography —
 * jedna zależność mniej, a wszystkie wartości i tak pochodzą z tokenów.
 */
export function Markdown({ children }: { children: string }) {
  return (
    <div
      className={[
        'max-w-prose text-base text-muted',
        '[&>*+*]:mt-5',
        '[&_h1]:text-2xl [&_h1]:text-ink [&_h2]:text-xl [&_h2]:text-ink [&_h3]:text-lg [&_h3]:text-ink',
        // Więcej powietrza nad nagłówkiem niż pod nim — nagłówek należy
        // do tego, co po nim następuje.
        '[&>h1]:mt-12 [&>h2]:mt-12 [&>h3]:mt-10',
        '[&_strong]:font-semibold [&_strong]:text-ink',
        '[&_a]:text-accent [&_a]:underline [&_a]:underline-offset-2',
        '[&_ul]:list-disc [&_ol]:list-decimal [&_ul]:pl-5 [&_ol]:pl-5 [&_li]:mt-1.5',
        '[&_code]:rounded-sm [&_code]:bg-raised [&_code]:px-1.5 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-sm [&_code]:text-ink',
        '[&_pre]:overflow-x-auto [&_pre]:rounded-md [&_pre]:bg-raised [&_pre]:p-4',
        '[&_pre_code]:bg-transparent [&_pre_code]:p-0',
        '[&_blockquote]:border-l [&_blockquote]:border-line [&_blockquote]:pl-4 [&_blockquote]:text-faint',
        '[&_hr]:border-line',
        '[&_table]:w-full [&_table]:text-sm [&_th]:pb-2 [&_th]:text-left [&_th]:font-mono [&_th]:text-2xs [&_th]:uppercase [&_th]:text-faint',
        '[&_td]:border-t [&_td]:border-line [&_td]:py-2',
      ].join(' ')}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          // Linki z opisu prowadzą na zewnątrz — otwieramy je w nowej karcie,
          // żeby nie wybijać użytkownika z katalogu.
          a: ({ href, children }) => (
            <a href={href} target="_blank" rel="noreferrer noopener">
              {children}
            </a>
          ),
        }}
      >
        {children}
      </ReactMarkdown>
    </div>
  )
}
