import {useEffect, useState, type CSSProperties} from 'react';
import {X} from 'lucide-react';
import {SettingsProvider, useSettings} from '@/contexts/SettingsContext';
import {AuthProvider, useAuth} from '@/contexts/AuthContext';
import {LoginScreen} from '@/components/LoginScreen';
import {DisclaimerGate} from '@/components/DisclaimerGate';
import Recorder from '@/components/Recorder';
import {GetActiveSession as GoGetActiveSession, Quit as GoQuit} from '../wailsjs/go/main/App';
import {Toaster} from 'sonner';
import {useTheme} from '@/lib/preferences';
import './App.css';

function Overlay() {
    const {isAuthenticated, isLoading, user, logout, getFreshToken} = useAuth();
    const [activeSessionId, setActiveSessionId] = useState<string | undefined>(undefined);

    // Aktive (unfertige) Session beim Start abfragen, damit der Recorder sie fortsetzen kann.
    useEffect(() => {
        if (!isAuthenticated || !user) return;
        let cancelled = false;
        (async () => {
            try {
                const data = await GoGetActiveSession(user.id, await getFreshToken());
                if (!cancelled && data?.session_id) {
                    setActiveSessionId(data.session_id);
                }
            } catch (e) {
                console.warn('Active session check failed:', e);
            }
        })();
        return () => {
            cancelled = true;
        };
    }, [isAuthenticated, user]);

    if (isLoading) {
        return (
            <div className="app-loading">
                <div className="app-loading-spinner"/>
            </div>
        );
    }

    if (!isAuthenticated || !user) {
        return <LoginScreen/>;
    }

    if (!user.disclaimer_accepted_at) {
        return <DisclaimerGate/>;
    }

    return (
        <div className="app-overlay">
            <Recorder
                userId={String(user.id)}
                resumeSessionId={activeSessionId}
                onLogout={logout}
            />
        </div>
    );
}

// Abstand der Toasts zum Rand der Transkript-Card und ihre Maximalbreite.
const TOAST_INSET = 8;
const TOAST_MAX_WIDTH = 240;

// Position der Transkript-Card (.recorder-transcript) im Fenster, damit die Toasts
// unten rechts in der Card erscheinen und nicht die Schaltflächen darunter verdecken.
// null, solange keine Card da ist (Login, Disclaimer) — dann gilt die Standardposition.
function useToastAnchor() {
    const [rect, setRect] = useState<DOMRect | null>(null);
    useEffect(() => {
        let observed: Element | null = null;
        const update = () => {
            const el = document.querySelector('.recorder-transcript');
            if (el !== observed) {
                if (observed) resizeObserver.unobserve(observed);
                if (el) resizeObserver.observe(el);
                observed = el;
            }
            setRect((prev) => {
                const next = el?.getBoundingClientRect() ?? null;
                if (!next || !prev) return next;
                const same = next.left === prev.left && next.right === prev.right && next.bottom === prev.bottom;
                return same ? prev : next;
            });
        };
        const resizeObserver = new ResizeObserver(update);
        // Die Card wird erst nach Login/Disclaimer gemountet und wandert, wenn sich
        // die Leisten darunter ändern (Aktions-Buttons, Visualizer).
        const mutationObserver = new MutationObserver(update);
        mutationObserver.observe(document.body, {childList: true, subtree: true});
        window.addEventListener('resize', update);
        update();
        return () => {
            resizeObserver.disconnect();
            mutationObserver.disconnect();
            window.removeEventListener('resize', update);
        };
    }, []);
    return rect;
}

function App() {
    const theme = useTheme();
    const anchor = useToastAnchor();
    // Das Overlay ist schmaler als 600px, dort gilt bei sonner mobileOffset und die
    // Toasts sind volle Breite — daher Breite und linken Rand hier selbst setzen.
    const anchored = anchor && {
        right: window.innerWidth - anchor.right + TOAST_INSET,
        bottom: window.innerHeight - anchor.bottom + TOAST_INSET,
        width: Math.min(TOAST_MAX_WIDTH, anchor.width - 2 * TOAST_INSET),
    };
    return (
        <SettingsProvider>
            <AuthProvider>
                {/* Frameless-Fenster hat keinen OS-Rahmen — dieser Button ist der einzige
                    Weg, die App zu schließen, daher unabhängig vom Auth-/Ladezustand sichtbar. */}
                <button
                    className="app-close"
                    onClick={() => GoQuit()}
                    title="Schließen"
                    aria-label="Schließen"
                >
                    <X size={16}/>
                </button>
                <Overlay/>
                <Toaster
                    // Unten rechts in der Transkript-Card (ohne Card: unten mittig über der
                    // Tastenkürzel-Leiste); kurz sichtbar (2,5 s statt sonners 4 s).
                    // Fehler und Berechtigungshinweise setzen ihre eigene, längere duration.
                    position={anchored ? 'bottom-right' : 'bottom-center'}
                    offset={anchored ? {bottom: anchored.bottom, right: anchored.right} : {bottom: 40}}
                    mobileOffset={anchored ? {bottom: anchored.bottom, right: anchored.right, left: 0} : {bottom: 40}}
                    style={anchored ? {left: 'auto', width: anchored.width, '--width': `${anchored.width}px`} as CSSProperties : undefined}
                    theme={theme}
                    toastOptions={{
                        duration: 2500,
                        style: {
                            fontSize: '12px',
                        },
                    }}
                />
            </AuthProvider>
        </SettingsProvider>
    );
}

export default App;
