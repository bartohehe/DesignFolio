import type { ApiErrorBody, ApiErrorCode } from './types'

/** Klucz w localStorage na opcjonalny APP_TOKEN (używany tylko przy zapisach). */
const TOKEN_KEY = 'designfolio.token'

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) ?? ''
}

export function setToken(token: string): void {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

/**
 * Zdarzenia okna dialogowego z tokenem. Zwykłe zdarzenia przeglądarki zamiast
 * globalnego stanu Reacta: zgłasza je MutationCache (poza drzewem komponentów),
 * a odbiera TokenDialog — nie muszą o sobie nic wiedzieć.
 */
export const TOKEN_REQUIRED_EVENT = 'designfolio:token-required'
export const TOKEN_OPEN_EVENT = 'designfolio:token-open'

/** Otwiera okno tokenu na żądanie użytkownika (np. z marginesu). */
export function openTokenDialog(): void {
  window.dispatchEvent(new Event(TOKEN_OPEN_EVENT))
}

/**
 * Błąd API z zachowanym kodem i statusem HTTP — dzięki temu UI może
 * rozróżnić "nie znaleziono" od "serwer padł" bez parsowania komunikatów.
 */
export class ApiError extends Error {
  readonly code: ApiErrorCode
  readonly status: number

  constructor(status: number, code: ApiErrorCode, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

/**
 * Cienka nakładka na fetch: dokleja token przy zapisach, parsuje JSON
 * i zamienia błędne odpowiedzi na ApiError.
 *
 * Ścieżki są zawsze względne ("/api/..."), więc ten sam kod działa
 * i za proxy Vite w dev, i pod binarką Go w prod.
 */
export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const method = (init.method ?? 'GET').toUpperCase()
  const headers = new Headers(init.headers)

  const token = getToken()
  if (token && method !== 'GET' && method !== 'HEAD') {
    headers.set('X-App-Token', token)
  }

  const res = await fetch(path, { ...init, headers })

  if (!res.ok) {
    throw await toApiError(res)
  }

  if (res.status === 204) {
    return undefined as T
  }
  return (await res.json()) as T
}

/**
 * Wysyłka plików z podglądem postępu.
 *
 * Tu jako jedyne miejsce w projekcie używamy XMLHttpRequest zamiast fetch:
 * fetch nie raportuje postępu WYSYŁANIA (tylko pobierania), a przy modelu 3D
 * albo serii zdjęć pasek postępu jest różnicą między "działa" a "zawiesiło się".
 */
export function uploadFiles<T>(
  path: string,
  files: File[],
  onProgress: (percent: number) => void,
): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const form = new FormData()
    for (const file of files) {
      form.append('files', file, file.name)
    }

    const xhr = new XMLHttpRequest()
    xhr.open('POST', path)

    const token = getToken()
    if (token) xhr.setRequestHeader('X-App-Token', token)

    xhr.upload.addEventListener('progress', (event) => {
      if (event.lengthComputable) {
        onProgress(Math.round((event.loaded / event.total) * 100))
      }
    })

    xhr.addEventListener('load', () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          resolve(JSON.parse(xhr.responseText) as T)
        } catch {
          reject(new ApiError(xhr.status, 'internal', 'serwer zwrócił niepoprawny JSON'))
        }
        return
      }
      try {
        const body = JSON.parse(xhr.responseText) as ApiErrorBody
        reject(new ApiError(xhr.status, body.error.code, body.error.message))
      } catch {
        reject(new ApiError(xhr.status, 'internal', `HTTP ${xhr.status}`))
      }
    })

    xhr.addEventListener('error', () =>
      reject(new ApiError(0, 'internal', 'przerwane połączenie z serwerem')),
    )
    xhr.addEventListener('abort', () =>
      reject(new ApiError(0, 'internal', 'wysyłka anulowana')),
    )

    xhr.send(form)
  })
}

async function toApiError(res: Response): Promise<ApiError> {
  try {
    const body = (await res.json()) as ApiErrorBody
    if (body.error?.code) {
      return new ApiError(res.status, body.error.code, body.error.message)
    }
  } catch {
    // Odpowiedź nie była JSON-em (np. proxy zwróciło HTML) — lecimy dalej
    // i budujemy błąd z samego statusu.
  }
  return new ApiError(res.status, 'internal', `HTTP ${res.status} ${res.statusText}`)
}
