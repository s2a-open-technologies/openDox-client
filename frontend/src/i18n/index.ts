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

import i18n from "i18next";
import {initReactI18next} from "react-i18next";

// Jede locales/<sprache>/translation.json wird automatisch eingelesen. Eine neue Sprache:
// Datei anlegen, hier in SUPPORTED_LANGUAGES eintragen und in internal/clientpolicy
// (UILanguages) ergänzen.
const modules = import.meta.glob<Record<string, unknown>>("./locales/*/translation.json", {
    eager: true,
    import: "default",
});
const resources: Record<string, {translation: Record<string, unknown>}> = {};
for (const [path, content] of Object.entries(modules)) {
    const lang = path.match(/\.\/locales\/([^/]+)\//)?.[1];
    if (lang) resources[lang] = {translation: content};
}

export const SUPPORTED_LANGUAGES = [
    {code: "de", label: "Deutsch"},
    {code: "en", label: "English"},
    {code: "es", label: "Español"},
] as const;
export type UiLanguage = (typeof SUPPORTED_LANGUAGES)[number]["code"];
export const DEFAULT_LANGUAGE: UiLanguage = "de";

export const isSupportedLanguage = (code: unknown): code is UiLanguage =>
    SUPPORTED_LANGUAGES.some((l) => l.code === code);

// Ohne Wahl des Nutzers (und ohne ui_language in settings.json) gilt die Sprache des
// Betriebssystems bzw. der WebView, sonst Deutsch.
export function detectSystemLanguage(): UiLanguage {
    for (const l of navigator.languages ?? [navigator.language]) {
        const code = l.slice(0, 2).toLowerCase();
        if (isSupportedLanguage(code)) return code;
    }
    return DEFAULT_LANGUAGE;
}

void i18n.use(initReactI18next).init({
    resources,
    lng: detectSystemLanguage(),
    fallbackLng: DEFAULT_LANGUAGE,
    interpolation: {escapeValue: false}, // React escaped selbst
    returnNull: false,
});

// Effektive Sprache setzen ("" = Systemsprache).
export function applyUiLanguage(code: string) {
    const lang = isSupportedLanguage(code) ? code : detectSystemLanguage();
    if (i18n.language !== lang) void i18n.changeLanguage(lang);
    document.documentElement.lang = lang;
}

// Texte aus dem Go-Teil: Fehler und Ereignisse tragen einen Code ("odx:<code>"), der hier in die
// aktuelle Sprache übersetzt wird. Alles andere (z. B. Servertexte) bleibt unverändert.
export function goMessage(err: unknown): string {
    const text = err instanceof Error ? err.message : String(err);
    const code = text.match(/^odx:([a-z_]+)$/)?.[1];
    return code && i18n.exists(`goerr.${code}`) ? i18n.t(`goerr.${code}`) : text;
}

export default i18n;
