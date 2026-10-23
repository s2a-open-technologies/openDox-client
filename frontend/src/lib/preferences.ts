/*
  openDox - Medical transcription platform
  Copyright (C) 2025  S2A Open Technologies gGmbH
  SPDX-FileCopyrightText: 2025 S2A Open Technologies gGmbH
  SPDX-License-Identifier: AGPL-3.0-or-later

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU Affero General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU Affero General Public License for more details.

  You should have received a copy of the GNU Affero General Public License
  along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import {useSyncExternalStore} from 'react';
import {toast} from 'sonner';
import {
    GetPreferences as GoGetPreferences,
    ResetPreferences as GoResetPreferences,
    SavePreferences as GoSavePreferences,
} from '../../wailsjs/go/main/App';
import {clientpolicy, main} from '../../wailsjs/go/models';

// Nutzer-Einstellungen (Design, Live-Ansicht, Mikrofontaste). Go hält sie in preferences.json im
// Benutzer-Konfigordner; ohne Wahl des Nutzers liefert GetPreferences die Vorgabe aus
// settings.json bzw. Server-Richtlinie. Ein kleiner Modul-Store statt eines Contexts, damit
// Recorder, LoginScreen und der Toaster in App.tsx denselben Stand sehen.
export type Theme = "dark" | "light";
export type Prefs = {theme: Theme; live_view: boolean; dictation_mic_mode: "hold" | "toggle"};

const DEFAULTS: Prefs = {theme: "dark", live_view: true, dictation_mic_mode: "hold"};
const listeners = new Set<() => void>();
let current: Prefs = DEFAULTS;

function set(p: main.Preferences) {
    current = {
        theme: p.theme === "light" ? "light" : "dark",
        live_view: p.live_view,
        dictation_mic_mode: p.dictation_mic_mode === "toggle" ? "toggle" : "hold",
    };
    document.documentElement.dataset.theme = current.theme;
    listeners.forEach((l) => l());
}

// Einmal vor dem ersten Rendern abwarten (main.tsx), damit das falsche Design nicht aufblitzt.
// Übernimmt dabei einmalig die frühere Wahl der Live-Ansicht aus dem localStorage.
export async function initPreferences() {
    try {
        set(await GoGetPreferences());
        let legacy: string | null = null;
        try {legacy = localStorage.getItem("live_view");} catch (_) { /* localStorage nicht verfügbar */ }
        if (legacy === "0" || legacy === "1") {
            await savePreferences({live_view: legacy === "1"});
        }
        try {localStorage.removeItem("live_view");} catch (_) { /* egal */ }
    } catch (e) {
        console.warn("Einstellungen nicht ladbar:", e);
        document.documentElement.dataset.theme = current.theme;
    }
}

export async function savePreferences(update: Partial<Prefs>) {
    try {
        set(await GoSavePreferences(clientpolicy.Preferences.createFrom(update)));
    } catch (e) {
        console.warn("Einstellungen nicht speicherbar:", e);
        toast.error("Einstellung konnte nicht gespeichert werden");
    }
}

export async function resetPreferences() {
    try {
        set(await GoResetPreferences());
    } catch (e) {
        console.warn("Einstellungen nicht zurücksetzbar:", e);
        toast.error("Einstellungen konnten nicht zurückgesetzt werden");
    }
}

export function toggleTheme() {
    return savePreferences({theme: current.theme === "dark" ? "light" : "dark"});
}

function subscribe(listener: () => void) {
    listeners.add(listener);
    return () => listeners.delete(listener);
}

export function usePreferences(): Prefs {
    return useSyncExternalStore(subscribe, () => current);
}

export function useTheme(): Theme {
    return usePreferences().theme;
}
