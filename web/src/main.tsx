import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { MutationCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Route, Routes } from 'react-router-dom'

// Fonty self-hostowane: paczki npm z plikami woff2, bundlowane przez Vite.
// Zero requestów do Google Fonts — aplikacja ma działać bez internetu.
import '@fontsource-variable/space-grotesk'
import '@fontsource-variable/inter'
import '@fontsource-variable/jetbrains-mono'

import './styles/tokens.css'
import { ApiError, TOKEN_REQUIRED_EVENT } from './api/client'
import { AppLayout } from './components/AppLayout'
import { ItemDetail } from './routes/ItemDetail'
import { EditItemPage, NewItemPage } from './routes/ItemForm'
import { NotFound } from './routes/NotFound'
import { SheetPage } from './routes/SheetPage'
import { TagsPage } from './routes/TagsPage'
import { TimelinePage } from './routes/TimelinePage'

const queryClient = new QueryClient({
  // Jedno miejsce, które widzi błąd KAŻDEGO zapisu (formularz, ulubione,
  // upload, usuwanie). Gdy serwer odpowie "unauthorized", otwieramy okno
  // z tokenem — zamiast obsługiwać ten przypadek w każdym komponencie osobno.
  mutationCache: new MutationCache({
    onError: (error) => {
      if (error instanceof ApiError && error.code === 'unauthorized') {
        window.dispatchEvent(new Event(TOKEN_REQUIRED_EVENT))
      }
    },
  }),
  defaultOptions: {
    queries: {
      // Aplikacja chodzi lokalnie i ma jednego użytkownika, więc dane nie
      // zmieniają się "same z siebie" — odpytywanie przy każdym powrocie
      // do karty byłoby hałasem bez wartości.
      refetchOnWindowFocus: false,
      staleTime: 30_000,
      retry: 1,
    },
  },
})

const container = document.getElementById('root')
if (!container) {
  throw new Error('Nie znaleziono #root w index.html')
}

createRoot(container).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route element={<AppLayout />}>
            <Route path="/" element={<SheetPage />} />
            <Route path="/nowa" element={<NewItemPage />} />
            <Route path="/rzecz/:slug" element={<ItemDetail />} />
            <Route path="/rzecz/:slug/edycja" element={<EditItemPage />} />
            <Route path="/tagi" element={<TagsPage />} />
            <Route path="/os-czasu" element={<TimelinePage />} />
            <Route path="*" element={<NotFound />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
)
