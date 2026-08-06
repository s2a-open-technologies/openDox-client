import {FormEvent, useEffect, useState} from 'react';
import {Loader2, Lock, LogIn, User} from 'lucide-react';
import {useAuth} from '@/contexts/AuthContext';
import {Button} from '@/components/ui/Button';
import {BASE_URL} from '@/api';
import {LoginWithOIDC as GoLoginWithOIDC} from '../../wailsjs/go/main/App';
import {EventsOn} from '../../wailsjs/runtime/runtime';
import './LoginScreen.css';

export const LoginScreen = () => {
    const {login, loginWithTokens, isLoading} = useAuth();
    const [name, setName] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState<string | null>(null);
    const [submitting, setSubmitting] = useState(false);
    const [oidcEnabled, setOidcEnabled] = useState(false);
    const [ssoWaiting, setSsoWaiting] = useState(false);

    useEffect(() => {
        fetch(`${BASE_URL}/api/v1/auth/oidc/status`)
            .then((res) => (res.ok ? res.json() : {enabled: false}))
            .then((data) => setOidcEnabled(!!data.enabled))
            .catch(() => setOidcEnabled(false));
    }, []);

    useEffect(() => {
        const offComplete = EventsOn('oidc:login-complete', (accessToken: string, refreshToken: string) => {
            setSsoWaiting(false);
            loginWithTokens(accessToken, refreshToken);
        });
        const offError = EventsOn('oidc:login-error', (message?: string) => {
            setSsoWaiting(false);
            setError(message || 'SSO-Login fehlgeschlagen.');
        });
        return () => {
            offComplete();
            offError();
        };
    }, []);

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault();
        setError(null);
        setSubmitting(true);
        const ok = await login(name.trim(), password);
        if (!ok) {
            setError('Anmeldung fehlgeschlagen. Bitte Benutzername und Passwort prüfen.');
        }
        setSubmitting(false);
    };

    const handleSsoLogin = async () => {
        setError(null);
        setSsoWaiting(true);
        try {
            await GoLoginWithOIDC();
        } catch (err) {
            console.error('SSO login error:', err);
            setSsoWaiting(false);
            setError('SSO-Login konnte nicht gestartet werden.');
        }
    };

    return (
        <div className="login">
            {/* Drag-Region am oberen Rand */}
            <div className="login-dragbar"/>

            <div className="login-inner">
                <div className="login-header">
                    <h1 className="login-title">openDox</h1>
                    <p className="login-subtitle">Bitte anmelden, um zu starten</p>
                </div>

                {oidcEnabled && (
                    <>
                        <Button
                            type="button"
                            variant="outline"
                            size="lg"
                            className="login-sso-btn"
                            onClick={handleSsoLogin}
                            disabled={ssoWaiting || submitting}
                        >
                            {ssoWaiting ? (
                                <>
                                    <Loader2 size={18} className="icon-spin-inline"/>
                                    Warte auf Browser...
                                </>
                            ) : (
                                <>
                                    <LogIn size={18}/>
                                    Mit SSO anmelden
                                </>
                            )}
                        </Button>
                        <div className="login-divider"><span>oder</span></div>
                    </>
                )}

                <form className="login-form" onSubmit={handleSubmit}>
                    <label className="login-field">
                        <span className="login-label">Benutzername</span>
                        <div className="login-input-wrap">
                            <User className="login-input-icon" size={16}/>
                            <input
                                type="text"
                                className="login-input"
                                value={name}
                                onChange={(e) => setName(e.target.value)}
                                autoComplete="username"
                                autoFocus
                                disabled={submitting || ssoWaiting}
                                placeholder="Benutzername"
                            />
                        </div>
                    </label>

                    <label className="login-field">
                        <span className="login-label">Passwort</span>
                        <div className="login-input-wrap">
                            <Lock className="login-input-icon" size={16}/>
                            <input
                                type="password"
                                className="login-input"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                autoComplete="current-password"
                                disabled={submitting || ssoWaiting}
                                placeholder="Passwort"
                            />
                        </div>
                    </label>

                    {error && <div className="login-error">{error}</div>}

                    <Button
                        type="submit"
                        variant="primary"
                        size="lg"
                        className="login-submit"
                        disabled={submitting || isLoading || ssoWaiting || !name.trim() || !password}
                    >
                        {submitting ? (
                            <>
                                <Loader2 size={18} className="icon-spin-inline"/>
                                Anmelden...
                            </>
                        ) : (
                            'Anmelden'
                        )}
                    </Button>
                </form>
            </div>
        </div>
    );
};

export default LoginScreen;
