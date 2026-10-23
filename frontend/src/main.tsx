import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import App from './App'
import {initPreferences} from './lib/preferences'

const container = document.getElementById('root')

const root = createRoot(container!)

// Erst die Nutzer-Einstellungen holen (Design), damit nicht kurz das falsche aufblitzt —
// höchstens 1,5 s, falls Go nicht antwortet.
Promise.race([initPreferences(), new Promise((r) => setTimeout(r, 1500))]).finally(() => {
    root.render(
        <React.StrictMode>
            <App/>
        </React.StrictMode>
    )
})
