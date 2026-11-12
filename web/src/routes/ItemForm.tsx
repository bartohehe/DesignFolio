import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'

import { ApiError } from '../api/client'
import { useCreateItem, useItem, useUpdateItem } from '../api/queries'
import {
  ITEM_KINDS,
  ITEM_STATUSES,
  type Item,
  type ItemInput,
  type ItemKind,
  type ItemStatus,
  type Metadata,
} from '../api/types'
import { Checkbox, Field, NumberInput, Select, TagInput, TextArea, TextInput } from '../components/form'
import { ErrorState } from '../components/states'
import { Button } from '../components/ui'
import { IconArrowLeft } from '../components/icons'
import { fromDateInput, toDateInput } from '../lib/format'
import { KIND_LABEL, META_FIELDS, STATUS_LABEL, type MetaField } from '../lib/kinds'

/** Stan formularza — wszystko jako tekst, konwersje robimy przy zapisie. */
interface FormState {
  title: string
  description: string
  kind: ItemKind
  status: ItemStatus
  favorite: boolean
  externalUrl: string
  happenedAt: string
  tags: string[]
  metadata: Metadata
}

const EMPTY: FormState = {
  title: '',
  description: '',
  kind: 'project',
  status: 'wip',
  favorite: false,
  externalUrl: '',
  happenedAt: '',
  tags: [],
  metadata: {},
}

export function NewItemPage() {
  return <ItemFormView mode="create" initial={EMPTY} />
}

export function EditItemPage() {
  const { slug = '' } = useParams()
  const item = useItem(slug)

  if (item.isPending) return <p className="text-sm text-faint">Wczytuję…</p>
  if (item.isError) return <ErrorState error={item.error} onRetry={() => void item.refetch()} />
  if (!item.data) return null

  return <ItemFormView mode="edit" item={item.data} initial={toFormState(item.data)} />
}

function toFormState(item: Item): FormState {
  return {
    title: item.title,
    description: item.description,
    kind: item.kind,
    status: item.status,
    favorite: item.favorite,
    externalUrl: item.external_url ?? '',
    happenedAt: toDateInput(item.happened_at),
    tags: item.tags.map((t) => t.name),
    metadata: item.metadata,
  }
}

function ItemFormView({
  mode,
  item,
  initial,
}: {
  mode: 'create' | 'edit'
  item?: Item
  initial: FormState
}) {
  const navigate = useNavigate()
  const [form, setForm] = useState<FormState>(initial)

  const create = useCreateItem()
  const update = useUpdateItem(item?.id ?? '')
  const pending = create.isPending || update.isPending
  const serverError = create.error ?? update.error

  // Zmiana pojedynczego pola bez rozwlekłego setForm w każdym handlerze.
  function set<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((current) => ({ ...current, [key]: value }))
  }

  function setMeta(key: string, value: unknown) {
    setForm((current) => {
      const metadata = { ...current.metadata }
      // Puste pole usuwamy z JSON-a zamiast zapisywać "" — inaczej metadane
      // zapełniłyby się pustymi kluczami po każdej edycji.
      if (value === undefined || value === '' || (Array.isArray(value) && value.length === 0)) {
        delete metadata[key]
      } else {
        metadata[key] = value
      }
      return { ...current, metadata }
    })
  }

  function handleSubmit(event: React.FormEvent) {
    event.preventDefault()

    const payload: ItemInput = {
      title: form.title.trim(),
      description: form.description,
      kind: form.kind,
      status: form.status,
      favorite: form.favorite,
      external_url: form.externalUrl.trim() === '' ? null : form.externalUrl.trim(),
      metadata: form.metadata,
      happened_at: fromDateInput(form.happenedAt),
      tags: form.tags,
    }

    if (mode === 'create') {
      create.mutate(payload, {
        onSuccess: (created) => void navigate(`/rzecz/${created.slug}`, { replace: true }),
      })
    } else {
      update.mutate(payload, {
        onSuccess: (saved) => void navigate(`/rzecz/${saved.slug}`, { replace: true }),
      })
    }
  }

  const fieldError = extractFieldError(serverError)

  return (
    <form onSubmit={handleSubmit} className="max-w-2xl">
      <button
        type="button"
        onClick={() => void navigate(-1)}
        className="transition-tint inline-flex items-center gap-2 rounded-sm text-sm text-faint hover:text-ink"
      >
        <IconArrowLeft />
        Wróć
      </button>

      <h1 className="mt-8 text-3xl text-ink">
        {mode === 'create' ? 'Nowa rzecz' : 'Edycja'}
      </h1>
      <p className="mt-3 max-w-prose text-base text-muted">
        {mode === 'create'
          ? 'Wystarczy tytuł i rodzaj. Pliki dorzucisz zaraz po zapisaniu, na stronie rzeczy.'
          : 'Pliki, okładkę i ich kolejność ustawia się na stronie rzeczy, nie tutaj.'}
      </p>

      <div className="mt-12 space-y-8">
        <Field label="Tytuł" error={fieldError('title')}>
          {(props) => (
            <TextInput
              {...props}
              value={form.title}
              onChange={(e) => set('title', e.target.value)}
              placeholder="Uchwyt na słuchawki"
              autoFocus={mode === 'create'}
              required
            />
          )}
        </Field>

        <div className="grid gap-8 sm:grid-cols-2">
          <Field label="Rodzaj" error={fieldError('kind')}>
            {(props) => (
              <Select
                {...props}
                value={form.kind}
                onChange={(e) => set('kind', e.target.value as ItemKind)}
              >
                {ITEM_KINDS.map((kind) => (
                  <option key={kind} value={kind}>
                    {KIND_LABEL[kind]}
                  </option>
                ))}
              </Select>
            )}
          </Field>

          <Field label="Stan" error={fieldError('status')}>
            {(props) => (
              <Select
                {...props}
                value={form.status}
                onChange={(e) => set('status', e.target.value as ItemStatus)}
              >
                {ITEM_STATUSES.map((status) => (
                  <option key={status} value={status}>
                    {STATUS_LABEL[status]}
                  </option>
                ))}
              </Select>
            )}
          </Field>
        </div>

        <Field
          label="Kiedy powstało"
          hint="Zostaw puste, jeśli nie pamiętasz — wtedy liczy się data dodania."
          error={fieldError('happened_at')}
        >
          {(props) => (
            <TextInput
              {...props}
              type="date"
              value={form.happenedAt}
              onChange={(e) => set('happenedAt', e.target.value)}
              className="w-auto font-mono"
            />
          )}
        </Field>

        <Field label="Opis" hint="Markdown działa: nagłówki, listy, linki, kod." error={fieldError('description')}>
          {(props) => (
            <TextArea
              {...props}
              value={form.description}
              onChange={(e) => set('description', e.target.value)}
              rows={8}
              placeholder="Co to jest, jak powstało, co poszło nie tak…"
            />
          )}
        </Field>

        <Field label="Tagi" hint="Enter albo przecinek dodaje tag." error={fieldError('tags')}>
          {(props) => <TagInput {...props} value={form.tags} onChange={(tags) => set('tags', tags)} />}
        </Field>

        <Field label="Link" hint="Repozytorium, wpis na blogu, model na Printables…" error={fieldError('external_url')}>
          {(props) => (
            <TextInput
              {...props}
              type="url"
              inputMode="url"
              value={form.externalUrl}
              onChange={(e) => set('externalUrl', e.target.value)}
              placeholder="https://"
            />
          )}
        </Field>

        {META_FIELDS[form.kind].length > 0 && (
          <fieldset className="border-t border-line pt-8">
            <legend className="sr-only">Metadane rodzaju {KIND_LABEL[form.kind]}</legend>
            <p className="font-mono text-2xs uppercase text-faint">
              {KIND_LABEL[form.kind]} — szczegóły
            </p>
            <div className="mt-6 grid gap-8 sm:grid-cols-2">
              {META_FIELDS[form.kind].map((field) => (
                <MetaInput
                  key={field.key}
                  field={field}
                  value={form.metadata[field.key]}
                  onChange={(value) => setMeta(field.key, value)}
                />
              ))}
            </div>
          </fieldset>
        )}

        <div className="border-t border-line pt-8">
          <Checkbox
            checked={form.favorite}
            onChange={(checked) => set('favorite', checked)}
            label="Ulubione"
          />
        </div>
      </div>

      {serverError && !fieldError.matchedAny && (
        <p className="mt-8 text-sm text-danger" role="alert">
          {serverError.message}
        </p>
      )}

      <div className="mt-12 flex items-center gap-3">
        <Button type="submit" variant="primary" disabled={pending || form.title.trim() === ''}>
          {pending ? 'Zapisuję…' : mode === 'create' ? 'Zapisz rzecz' : 'Zapisz zmiany'}
        </Button>
        <Button type="button" variant="ghost" onClick={() => void navigate(-1)}>
          Anuluj
        </Button>
      </div>
    </form>
  )
}

function MetaInput({
  field,
  value,
  onChange,
}: {
  field: MetaField
  value: unknown
  onChange: (value: unknown) => void
}) {
  return (
    <Field label={field.label}>
      {(props) => {
        switch (field.type) {
          case 'number':
            return (
              <NumberInput
                {...props}
                {...(field.suffix ? { suffix: field.suffix } : {})}
                value={typeof value === 'number' ? value : ''}
                step="any"
                onChange={(e) =>
                  onChange(e.target.value === '' ? undefined : Number(e.target.value))
                }
              />
            )
          case 'list':
            return (
              <TextInput
                {...props}
                value={Array.isArray(value) ? value.join(', ') : ''}
                placeholder={field.placeholder ?? ''}
                onChange={(e) =>
                  onChange(
                    e.target.value
                      .split(',')
                      .map((part) => part.trim())
                      .filter(Boolean),
                  )
                }
              />
            )
          default:
            return (
              <TextInput
                {...props}
                type={field.type === 'url' ? 'url' : 'text'}
                value={typeof value === 'string' ? value : ''}
                placeholder={field.placeholder ?? ''}
                onChange={(e) => onChange(e.target.value)}
              />
            )
        }
      }}
    </Field>
  )
}

/**
 * Backend zwraca błędy walidacji w postaci "pole: komunikat".
 * Rozbijamy je, żeby komunikat wylądował przy właściwym polu,
 * a nie w ogólnym pasku na dole.
 */
function extractFieldError(error: Error | null) {
  let field = ''
  let message = ''

  if (error instanceof ApiError && error.code === 'bad_request') {
    const [head, ...rest] = error.message.split(':')
    if (rest.length > 0) {
      field = head!.trim()
      message = rest.join(':').trim()
    }
  }

  const lookup = (name: string): string | undefined =>
    field === name ? message : undefined

  // Flaga mówi widokowi, czy błąd został już pokazany przy polu.
  return Object.assign(lookup, { matchedAny: field !== '' })
}
