/*
  openDox - Medical transcription platform
  Copyright (C) 2025  Christopher Abanilla <christopher.abanilla@gmx.de>
  SPDX-FileCopyrightText: 2025 Christopher Abanilla <christopher.abanilla@gmx.de>
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

import {createContext, useState, useEffect, useContext, ReactNode} from 'react';
import {useSettings} from '@/contexts/SettingsContext';

type User = {
    id: number;
    name: string;
    is_active: boolean;
    authenticated: boolean;
    role: 'admin' | 'moderator' | 'user';
    disclaimer_accepted_at?: string | null;
};

type AuthContextType = {
    user: User | null;
    isLoading: boolean;
    isAuthenticated: boolean;
    login: (name: string, password: string) => Promise<boolean>;
    loginWithTokens: (accessToken: string, refreshToken: string) => Promise<void>;
    logout: () => Promise<void>;
    authFetch: (url: string, options?: RequestInit) => Promise<Response>;
    getToken: () => string;
    acceptDisclaimer: () => Promise<boolean>;
};

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({children}: { children: ReactNode }) => {
    const {settings, isLoading: settingsLoading} = useSettings();
    const restURL = settings?.server_url || "http://localhost:10000";
    const [user, setUser] = useState<User | null>(null);
    const [isLoading, setIsLoading] = useState<boolean>(true);
    const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);

    const getToken = (): string => localStorage.getItem("access_token") || "";
    const getRefreshToken = (): string => localStorage.getItem("refresh_token") || "";

    const applyTokens = (accessToken: string, refreshToken: string): void => {
        localStorage.setItem("access_token", accessToken);
        localStorage.setItem("refresh_token", refreshToken);
    };

    // Tauscht den Refresh-Token gegen ein frisches Token-Paar - vermeidet, dass jeder
    // Neustart nach Ablauf des kurzlebigen Access-Tokens (Default 30min) eine erneute
    // manuelle Anmeldung erzwingt, solange der Refresh-Token (Default 7 Tage) noch gültig ist.
    const refreshAccessToken = async (): Promise<boolean> => {
        const refreshToken = getRefreshToken();
        if (!refreshToken) return false;
        try {
            const res = await fetch(
                `${restURL}/api/v1/auth/refresh?refresh_token=${encodeURIComponent(refreshToken)}`,
                {method: "POST"}
            );
            if (!res.ok) return false;
            const data = await res.json();
            if (!data.access_token || !data.refresh_token) return false;
            applyTokens(data.access_token, data.refresh_token);
            return true;
        } catch (error) {
            console.error("Token refresh error:", error);
            return false;
        }
    };

    const authFetch = async (url: string, options: RequestInit = {}): Promise<Response> => {
        const token = getToken();
        const headers = {
            ...options.headers,
            Authorization: token ? `Bearer ${token}` : "",
        };
        const fullUrl = /^https?:\/\//i.test(url) ? url : `${restURL}${url}`;
        return fetch(fullUrl, {...options, headers});
    };

    const fetchUser = async (): Promise<void> => {
        try {
            let res = await authFetch("/api/v1/auth/me", {method: "GET"});
            if (res.status === 401) {
                // Access-Token abgelaufen (typischerweise nach einem Neustart) - still per
                // Refresh-Token erneuern statt den Nutzer zur erneuten Anmeldung zu zwingen.
                const refreshed = await refreshAccessToken();
                if (refreshed) {
                    res = await authFetch("/api/v1/auth/me", {method: "GET"});
                }
            }
            if (res.ok) {
                const data = await res.json();
                setUser({
                    id: data.id,
                    name: data.name,
                    is_active: data.is_active,
                    authenticated: true,
                    role: data.role,
                    disclaimer_accepted_at: data.disclaimer_accepted_at ?? null,
                });
                setIsAuthenticated(true);
            } else {
                localStorage.removeItem("access_token");
                localStorage.removeItem("refresh_token");
                setUser(null);
                setIsAuthenticated(false);
            }
        } catch (error) {
            console.error("Error fetching user:", error);
            setUser(null);
            setIsAuthenticated(false);
        } finally {
            setIsLoading(false);
        }
    };

    const login = async (name: string, password: string): Promise<boolean> => {
        setIsLoading(true);
        try {
            const res = await fetch(`${restURL}/api/v1/auth/login`, {
                method: "POST",
                headers: {"Content-Type": "application/json"},
                body: JSON.stringify({name, password}),
            });

            if (!res.ok) return false;

            const data = await res.json();
            if (!data.access_token) return false;

            applyTokens(data.access_token, data.refresh_token || "");
            await fetchUser();
            return true;
        } catch (error) {
            console.error("Login error:", error);
            return false;
        } finally {
            setIsLoading(false);
        }
    };

    // Schließt einen OIDC/SSO-Login ab (siehe LoginScreen), bei dem die Tokens bereits
    // vom Go-seitigen Loopback-Callback geliefert wurden statt über das Passwort-Formular.
    const loginWithTokens = async (accessToken: string, refreshToken: string): Promise<void> => {
        setIsLoading(true);
        applyTokens(accessToken, refreshToken);
        await fetchUser();
    };

    const logout = async (): Promise<void> => {
        try {
            await fetch(`${restURL}/api/v1/auth/logout`, {method: "POST"});
        } catch (error) {
            console.error("Logout error:", error);
        } finally {
            localStorage.removeItem("access_token");
            localStorage.removeItem("refresh_token");
            localStorage.removeItem("currentSession");
            setUser(null);
            setIsAuthenticated(false);
        }
    };

    const acceptDisclaimer = async (): Promise<boolean> => {
        try {
            const res = await authFetch("/api/v1/auth/accept-disclaimer", {method: "POST"});
            if (!res.ok) return false;
            const data = await res.json();
            setUser((prev) =>
                prev ? {...prev, disclaimer_accepted_at: data.disclaimer_accepted_at} : prev
            );
            return true;
        } catch (error) {
            console.error("Error accepting disclaimer:", error);
            return false;
        }
    };

    useEffect(() => {
        // Wichtig: erst starten, wenn die Server-URL feststeht (Settings geladen).
        // Sonst läuft der Token-Restore gegen den localhost-Fallback und scheitert,
        // wenn die App auf einem anderen Server konfiguriert ist — der Nutzer müsste
        // sich dann bei jedem App-Start neu einloggen, obwohl Tokens gespeichert sind.
        if (settingsLoading) return;
        if (getToken()) {
            fetchUser();
        } else {
            setIsLoading(false);
        }
    }, [settingsLoading]);

    const value = {user, isLoading, isAuthenticated, login, loginWithTokens, logout, authFetch, getToken, acceptDisclaimer};

    return (
        <AuthContext.Provider value={value}>
            {children}
        </AuthContext.Provider>
    );
};

export const useAuth = (): AuthContextType => {
    const context = useContext(AuthContext);
    if (context === undefined) {
        throw new Error("useAuth must be used within an AuthProvider");
    }
    return context;
};
