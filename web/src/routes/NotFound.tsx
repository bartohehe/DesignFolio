import { EmptyState } from '../components/states'
import { ButtonLink } from '../components/ui'

export function NotFound() {
  return (
    <EmptyState
      title="Nie ma takiej strony"
      description="Adres nie pasuje do niczego w katalogu. Może rzecz została usunięta albo zmienił się jej tytuł — a razem z nim adres."
      action={<ButtonLink to="/">Wróć na arkusz</ButtonLink>}
    />
  )
}
