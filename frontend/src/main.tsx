import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import '@fontsource-variable/newsreader'
import '@fontsource/ibm-plex-sans/latin-400.css'
import '@fontsource/ibm-plex-sans/latin-500.css'
import '@fontsource/ibm-plex-sans/latin-600.css'
import '@fontsource/ibm-plex-mono/latin-400.css'
import '@fontsource/ibm-plex-mono/latin-500.css'
import { App } from './App'
import { PrintableProposal } from './PrintableProposal'
import { VersionComparison } from './VersionComparison'
import './styles.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    {/\/compare(?:\/report)?\/?$/.test(window.location.pathname) ? <VersionComparison /> : window.location.pathname.endsWith('/print') || window.location.pathname.endsWith('/print/') ? <PrintableProposal /> : <App />}
  </StrictMode>,
)
