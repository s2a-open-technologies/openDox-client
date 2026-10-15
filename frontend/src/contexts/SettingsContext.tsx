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

import {createContext, useContext, useState, useEffect, ReactNode} from 'react';
import {GetSettings as GoGetSettings} from '../../wailsjs/go/main/App';

type SettingsContextType = {
    settings: {server_url: string} | null;
    isLoading: boolean;
};

const SettingsContext = createContext<SettingsContextType | undefined>(undefined);

export const SettingsProvider = ({children}: { children: ReactNode }) => {
    const [settings, setSettings] = useState<{server_url: string} | null>(null);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        GoGetSettings()
            .then((s) => {
                setSettings({server_url: s.server_url});
            })
            .catch((err) => {
                console.error("Settings laden fehlgeschlagen:", err);
            })
            .finally(() => {
                setIsLoading(false);
            });
    }, []);

    return (
        <SettingsContext.Provider value={{settings, isLoading}}>
            {children}
        </SettingsContext.Provider>
    );
};

export const useSettings = (): SettingsContextType => {
    const context = useContext(SettingsContext);
    if (context === undefined) {
        throw new Error("useSettings must be used within a SettingsProvider");
    }
    return context;
};
