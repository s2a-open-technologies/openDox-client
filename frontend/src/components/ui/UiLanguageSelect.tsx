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

import {useTranslation} from 'react-i18next';
import {SUPPORTED_LANGUAGES} from '@/i18n';
import {savePreferences, usePreferences} from '@/lib/preferences';

// Auswahl der Oberflächensprache; die Wahl landet in preferences.json (ui_language).
export function UiLanguageSelect({className}: {className?: string}) {
    const {t, i18n} = useTranslation();
    const prefs = usePreferences();
    return (
        <select
            className={className}
            value={prefs.ui_language || i18n.language.slice(0, 2)}
            onChange={(e) => savePreferences({ui_language: e.target.value})}
            title={t('language.label')}
            aria-label={t('language.label')}
        >
            {SUPPORTED_LANGUAGES.map((l) => (
                <option key={l.code} value={l.code}>{l.label}</option>
            ))}
        </select>
    );
}
