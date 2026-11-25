import {FormEvent, useEffect, useState} from 'react';
import {useTranslation} from 'react-i18next';
import {goMessage} from '@/i18n';
import {UiLanguageSelect} from '@/components/ui/UiLanguageSelect';
import {Loader2, Lock, LogIn, Moon, Sun, User} from 'lucide-react';
import {toggleTheme, useTheme} from '@/lib/preferences';
import {useAuth} from '@/contexts/AuthContext';
import {useSettings} from '@/contexts/SettingsContext';
import {Button} from '@/components/ui/Button';
import {LoginWithOIDC as GoLoginWithOIDC} from '../../wailsjs/go/main/App';
import {EventsOn} from '../../wailsjs/runtime/runtime';
import './LoginScreen.css';

export const LoginScreen = () => {
    const {login, loginWithTokens, isLoading} = useAuth();
    const {settings, isLoading: settingsLoading} = useSettings();
    const restURL = settings?.server_url || "http://localhost:10000";
    const [name, setName] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState<string | null>(null);
    const [submitting, setSubmitting] = useState(false);
    const [oidcEnabled, setOidcEnabled] = useState(false);
    const [ssoWaiting, setSsoWaiting] = useState(false);
    const theme = useTheme();
    const {t} = useTranslation();

    useEffect(() => {
        // Erst abfragen, wenn die Server-URL feststeht — sonst geht der Request
        // an den localhost-Fallback und der SSO-Button erscheint fälschlich nie.
        if (settingsLoading) return;
        fetch(`${restURL}/api/v1/auth/oidc/status`)
            .then((res) => (res.ok ? res.json() : {enabled: false}))
            .then((data) => setOidcEnabled(!!data.enabled))
            .catch(() => setOidcEnabled(false));
    }, [settingsLoading]);

    useEffect(() => {
        const offComplete = EventsOn('oidc:login-complete', (accessToken: string, refreshToken: string) => {
            setSsoWaiting(false);
            loginWithTokens(accessToken, refreshToken);
        });
        const offError = EventsOn('oidc:login-error', (message?: string) => {
            setSsoWaiting(false);
            setError(message ? goMessage(message) : t('login.ssoFailed'));
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
            setError(t('login.failed'));
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
            setError(t('login.ssoStartFailed'));
        }
    };

    return (
        <div className="login">
            {/* Drag-Region am oberen Rand */}
            <div className="login-dragbar"/>
            <button
                type="button"
                className="login-theme-btn"
                onClick={toggleTheme}
                title={theme === "dark" ? t("theme.light") : t("theme.dark")}
                aria-label={theme === "dark" ? t("theme.toLight") : t("theme.toDark")}
            >
                {theme === "dark" ? <Sun size={14}/> : <Moon size={14}/>}
            </button>

            <UiLanguageSelect className="login-lang-select"/>

            <div className="login-inner">
                <div className="login-header">
                    <h1 className="login-title">openDox</h1>
                    <p className="login-subtitle">{t('login.subtitle')}</p>
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
                                    {t('login.ssoWaiting')}
                                </>
                            ) : (
                                <>
                                    <LogIn size={18}/>
                                    {t('login.ssoButton')}
                                </>
                            )}
                        </Button>
                        <div className="login-divider"><span>{t('login.or')}</span></div>
                    </>
                )}

                <form className="login-form" onSubmit={handleSubmit}>
                    <label className="login-field">
                        <span className="login-label">{t('login.username')}</span>
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
                                placeholder={t('login.username')}
                            />
                        </div>
                    </label>

                    <label className="login-field">
                        <span className="login-label">{t('login.password')}</span>
                        <div className="login-input-wrap">
                            <Lock className="login-input-icon" size={16}/>
                            <input
                                type="password"
                                className="login-input"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                autoComplete="current-password"
                                disabled={submitting || ssoWaiting}
                                placeholder={t('login.password')}
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
                                {t('login.submitting')}
                            </>
                        ) : (
                            t('login.submit')
                        )}
                    </Button>
                </form>
            </div>
        </div>
    );
};

export default LoginScreen;
