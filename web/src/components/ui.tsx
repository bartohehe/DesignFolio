import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { Link } from 'react-router-dom'

/**
 * Warianty przycisku w jednym miejscu, żeby ten sam zestaw klas dało się
 * nałożyć na <button> i na <Link> z routera bez kopiowania.
 *
 * Etykiety są zapisane zwykłym krojem tekstowym, nie monospace'em —
 * mono jest w tym projekcie zarezerwowane dla metadanych, tagów i liczb.
 */
export type ButtonVariant = 'primary' | 'quiet' | 'ghost' | 'mark' | 'danger'

const BASE =
  'transition-tint inline-flex items-center justify-center gap-2 rounded-md px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-50'

const VARIANTS: Record<ButtonVariant, string> = {
  primary: 'bg-accent text-canvas hover:bg-accent-dim',
  quiet: 'bg-raised text-ink hover:bg-line',
  ghost: 'text-muted hover:bg-surface hover:text-ink',
  // Stan włączony (np. ulubione). Musi być osobnym wariantem, a nie klasą
  // doklejaną do ghost: dwie klasy koloru tekstu mają tę samą wagę,
  // więc o wyniku decydowałaby kolejność w arkuszu, nie w atrybucie class.
  mark: 'text-accent hover:bg-surface',
  danger: 'text-danger hover:bg-danger/12',
}

export function buttonClass(variant: ButtonVariant = 'quiet', extra = ''): string {
  return [BASE, VARIANTS[variant], extra].filter(Boolean).join(' ')
}

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant
  children: ReactNode
}

export function Button({ variant = 'quiet', className = '', children, ...rest }: ButtonProps) {
  return (
    <button className={buttonClass(variant, className)} {...rest}>
      {children}
    </button>
  )
}

export function ButtonLink({
  to,
  variant = 'quiet',
  className = '',
  children,
}: {
  to: string
  variant?: ButtonVariant
  className?: string
  children: ReactNode
}) {
  return (
    <Link to={to} className={buttonClass(variant, className)}>
      {children}
    </Link>
  )
}
