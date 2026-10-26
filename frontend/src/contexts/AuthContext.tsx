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

import {createContext, useState, useEffect, useContext, useRef, ReactNode} from 'react';
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
    getFreshToken: () => Promise<string>;
    acceptDisclaimer: () => Promise<boolean>;
};

const AuthContext = createContext<AuthContextType | undefined>(undefined);

type RefreshOutcome = 'ok' | 'rejected' | 'error';

// Erneuern, sobald das Access-Token weniger als das noch gilt; der Timer schaut
// alle REFRESH_CHECK_MS nach. So bleibt der Client angemeldet, solange er läuft,
// und das Refresh-Token rotiert mit (läuft erst nach REFRESH_TOKEN_EXPIRE_DAYS
// ohne laufenden Client ab).
const REFRESH_MARGIN_MS = 2 * 60_000;
const REFRESH_CHECK_MS = 60_000;

// Ablaufzeit eines JWT in ms; 0, wenn sie nicht lesbar ist.
const tokenExpiry = (token: string): number => {
    try {
        const payload = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')));
        return typeof payload.exp === 'number' ? payload.exp * 1000 : 0;
    } catch {
        return 0;
    }
};

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

    const clearSession = (): void => {
        localStorage.removeItem("access_token");
        localStorage.removeItem("refresh_token");
        localStorage.removeItem("currentSession");
        setUser(null);
        setIsAuthenticated(false);
    };

    // Nur eine Erneuerung gleichzeitig: parallele Aufrufe warten auf dieselbe,
    // statt das (rotierende) Refresh-Token mehrfach zu verbrauchen.
    const refreshInFlight = useRef<Promise<RefreshOutcome> | null>(null);

    // Tauscht den Refresh-Token gegen ein frisches Token-Paar. Der Server prüft
    // dabei LDAP-/OIDC-Benutzer erneut: "rejected" heißt, die Sitzung ist wirklich
    // vorbei (Konto gesperrt, Gruppe entzogen, Refresh-Token abgelaufen). "error"
    // (Server/WLAN weg) lässt die Tokens stehen, damit der Client danach weiterläuft.
    const refreshAccessToken = (): Promise<RefreshOutcome> => {
        if (refreshInFlight.current) return refreshInFlight.current;
        const refreshToken = getRefreshToken();
        if (!refreshToken) return Promise.resolve('rejected');
        refreshInFlight.current = (async (): Promise<RefreshOutcome> => {
            try {
                // Im Body, nicht in der URL: URLs landen in Proxy-Logs.
                const res = await fetch(`${restURL}/api/v1/auth/refresh`, {
                    method: "POST",
                    headers: {"Content-Type": "application/json"},
                    body: JSON.stringify({refresh_token: refreshToken}),
                });
                if (res.status === 401) return 'rejected';
                if (!res.ok) return 'error';
                const data = await res.json();
                if (!data.access_token || !data.refresh_token) return 'error';
                applyTokens(data.access_token, data.refresh_token);
                return 'ok';
            } catch (error) {
                console.error("Token refresh error:", error);
                return 'error';
            } finally {
                refreshInFlight.current = null;
            }
        })();
        return refreshInFlight.current;
    };

    // Access-Token für die Go-Aufrufe und den WebSocket, die selbst nicht auf 401
    // reagieren: läuft es bald ab, wird es vorher erneuert.
    const getFreshToken = async (): Promise<string> => {
        const token = getToken();
        const expiry = token ? tokenExpiry(token) : 0;
        if (expiry && expiry - Date.now() < REFRESH_MARGIN_MS) {
            const outcome = await refreshAccessToken();
            if (outcome === 'rejected') {
                clearSession();
                return "";
            }
        }
        return getToken();
    };

    const plainFetch = (url: string, options: RequestInit): Promise<Response> => {
        const token = getToken();
        const headers = {
            ...options.headers,
            Authorization: token ? `Bearer ${token}` : "",
        };
        const fullUrl = /^https?:\/\//i.test(url) ? url : `${restURL}${url}`;
        return fetch(fullUrl, {...options, headers});
    };

    // Bei 401 einmal still erneuern und die Anfrage wiederholen.
    const authFetch = async (url: string, options: RequestInit = {}): Promise<Response> => {
        const res = await plainFetch(url, options);
        if (res.status !== 401) return res;
        const outcome = await refreshAccessToken();
        if (outcome === 'ok') return plainFetch(url, options);
        if (outcome === 'rejected' && getToken()) clearSession();
        return res;
    };

    const fetchUser = async (): Promise<void> => {
        try {
            // authFetch erneuert ein abgelaufenes Access-Token (typischerweise nach
            // einem Neustart) still per Refresh-Token.
            const res = await authFetch("/api/v1/auth/me", {method: "GET"});
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
                // Bei 401 hat authFetch die Tokens schon verworfen; bei einem
                // Serverfehler bleiben sie für den nächsten Start erhalten.
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
            clearSession();
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

    // Solange der Client angemeldet ist, das Token vor Ablauf erneuern. Läuft
    // der Rechner im Ruhezustand, holt der erste Tick danach das nach.
    useEffect(() => {
        if (!isAuthenticated) return;
        const timer = window.setInterval(() => {
            getFreshToken();
        }, REFRESH_CHECK_MS);
        return () => window.clearInterval(timer);
    }, [isAuthenticated, restURL]);

    const value = {user, isLoading, isAuthenticated, login, loginWithTokens, logout, authFetch, getToken, getFreshToken, acceptDisclaimer};

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
