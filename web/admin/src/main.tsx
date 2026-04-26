import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { translate } from '@/i18n'

// Replace the build-time placeholders the Go server injects when it
// renders the admin shell. The title fallback runs only when the server
// failed to substitute __KITE_ADMIN_TITLE__ — usually a sign the SPA is
// being served from `vite dev` directly. translate() reads the same
// kite_locale signal the server-rendered surfaces use, so the tab title
// matches whichever language the user picked last.
if (document.title === '__KITE_ADMIN_TITLE__') {
  document.title = translate('auth.adminDocumentTitle')
}

const faviconLink = document.querySelector<HTMLLinkElement>("link[rel='icon']")
if (faviconLink?.getAttribute('href') === '__KITE_FAVICON_URL__') {
  faviconLink.setAttribute('href', '/favicon.svg')
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>
)

// Fade out the pre-mount splash once React has painted.
requestAnimationFrame(() => {
  requestAnimationFrame(() => {
    document.getElementById('splash')?.classList.add('ready')
  })
})
