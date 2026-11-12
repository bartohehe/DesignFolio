import { useId, useState, type ReactNode } from 'react'

import { IconChevronDown, IconClose } from './icons'

/** Wspólne tło i kształt wszystkich kontrolek. */
const CONTROL =
  'transition-tint w-full rounded-md bg-raised px-3 py-2 text-base text-ink placeholder:text-faint'

/**
 * Field opakowuje etykietę, kontrolkę, podpowiedź i błąd.
 *
 * useId daje identyfikator unikalny w obrębie strony, więc powiązanie
 * label ↔ input działa nawet wtedy, gdy to samo pole pojawi się dwa razy.
 */
export function Field({
  label,
  hint,
  error,
  children,
}: {
  label: string
  hint?: string | undefined
  error?: string | undefined
  children: (props: { id: string; 'aria-describedby': string | undefined }) => ReactNode
}) {
  const id = useId()
  const hintId = hint ? `${id}-hint` : undefined
  const errorId = error ? `${id}-error` : undefined
  const describedBy = [hintId, errorId].filter(Boolean).join(' ') || undefined

  return (
    <div>
      <label htmlFor={id} className="block font-mono text-2xs uppercase text-faint">
        {label}
      </label>
      <div className="mt-2">{children({ id, 'aria-describedby': describedBy })}</div>
      {hint && !error && (
        <p id={hintId} className="mt-1.5 text-xs text-faint">
          {hint}
        </p>
      )}
      {error && (
        <p id={errorId} className="mt-1.5 text-xs text-danger">
          {error}
        </p>
      )}
    </div>
  )
}

export function TextInput({
  className = '',
  ...rest
}: React.InputHTMLAttributes<HTMLInputElement>) {
  return <input className={`${CONTROL} ${className}`} {...rest} />
}

export function TextArea({
  className = '',
  ...rest
}: React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={`${CONTROL} resize-y leading-relaxed ${className}`} {...rest} />
}

export function Select({
  className = '',
  children,
  ...rest
}: React.SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    // Natywny <select> zamiast własnego rozwijania: dostaje za darmo
    // obsługę klawiatury, czytniki ekranu i systemowy picker na telefonie.
    <div className="relative">
      <select className={`${CONTROL} appearance-none pr-10 ${className}`} {...rest}>
        {children}
      </select>
      <IconChevronDown className="pointer-events-none absolute right-3 top-1/2 size-4 -translate-y-1/2 text-faint" />
    </div>
  )
}

/** Pole liczbowe z jednostką dopisaną po prawej (h, mm, %). */
export function NumberInput({
  suffix,
  className = '',
  ...rest
}: React.InputHTMLAttributes<HTMLInputElement> & { suffix?: string | undefined }) {
  return (
    <div className="relative">
      <input
        type="number"
        className={`${CONTROL} font-mono tabular-nums ${suffix ? 'pr-12' : ''} ${className}`}
        {...rest}
      />
      {suffix && (
        <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 font-mono text-xs text-faint">
          {suffix}
        </span>
      )}
    </div>
  )
}

/**
 * Wprowadzanie tagów: Enter albo przecinek zatwierdza, Backspace na pustym
 * polu zdejmuje ostatni. Tagi żyją jako lista stringów — dopasowaniem
 * do istniejących zajmuje się backend, po slugu.
 */
export function TagInput({
  value,
  onChange,
  id,
  'aria-describedby': describedBy,
}: {
  value: string[]
  onChange: (tags: string[]) => void
  id?: string | undefined
  'aria-describedby'?: string | undefined
}) {
  const [draft, setDraft] = useState('')

  function commit(raw: string) {
    const tag = raw.trim().replace(/,$/, '').trim()
    if (tag && !value.some((t) => t.toLowerCase() === tag.toLowerCase())) {
      onChange([...value, tag])
    }
    setDraft('')
  }

  return (
    <div className="rounded-md bg-raised px-2 py-2">
      {value.length > 0 && (
        <ul className="mb-2 flex flex-wrap gap-1.5">
          {value.map((tag) => (
            <li key={tag}>
              <span className="inline-flex items-center gap-1 rounded-sm bg-surface py-1 pl-2.5 pr-1 font-mono text-xs text-muted">
                {tag}
                <button
                  type="button"
                  onClick={() => onChange(value.filter((t) => t !== tag))}
                  className="transition-tint rounded-sm p-0.5 text-faint hover:text-danger"
                >
                  <IconClose className="size-3" />
                  <span className="sr-only">Usuń tag {tag}</span>
                </button>
              </span>
            </li>
          ))}
        </ul>
      )}

      <input
        id={id}
        aria-describedby={describedBy}
        value={draft}
        onChange={(e) => {
          // Wklejenie "a, b, c" ma dodać trzy tagi, nie jeden.
          if (e.target.value.includes(',')) commit(e.target.value)
          else setDraft(e.target.value)
        }}
        onKeyDown={(e) => {
          if (e.key === 'Enter') {
            e.preventDefault() // inaczej Enter wysłałby cały formularz
            commit(draft)
          } else if (e.key === 'Backspace' && draft === '' && value.length > 0) {
            onChange(value.slice(0, -1))
          }
        }}
        onBlur={() => commit(draft)}
        placeholder={value.length === 0 ? 'druk 3d, biurko…' : 'dopisz kolejny'}
        className="w-full bg-transparent px-1 py-1 text-base text-ink placeholder:text-faint focus:outline-none"
      />
    </div>
  )
}

/** Przełącznik ulubionego — pole wyboru, ale wyglądające jak zaznaczenie. */
export function Checkbox({
  checked,
  onChange,
  label,
}: {
  checked: boolean
  onChange: (checked: boolean) => void
  label: string
}) {
  return (
    <label className="inline-flex cursor-pointer items-center gap-3">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="size-4 accent-[var(--color-accent)]"
      />
      <span className="text-sm text-muted">{label}</span>
    </label>
  )
}
