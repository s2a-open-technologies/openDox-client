import {useEffect, useState} from 'react';
import {X} from 'lucide-react';
import {AuthProvider, useAuth} from '@/contexts/AuthContext';
import {LoginScreen} from '@/components/LoginScreen';
import {DisclaimerGate} from '@/components/DisclaimerGate';
import Recorder from '@/components/Recorder';
import {GetActiveSession as GoGetActiveSession, Quit as GoQuit} from '../wailsjs/go/main/App';
import {Toaster} from 'sonner';
import './App.css';

function Overlay() {
    const {isAuthenticated, isLoading, user, logout, getToken} = useAuth();
    const [activeSessionId, setActiveSessionId] = useState<string | undefined>(undefined);

    // Aktive (unfertige) Session beim Start abfragen, damit der Recorder sie fortsetzen kann.
    useEffect(() => {
        if (!isAuthenticated || !user) return;
        let cancelled = false;
        (async () => {
            try {
                const data = await GoGetActiveSession(user.id, getToken());
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

function App() {
    return (
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
                position="top-center"
                theme="dark"
                toastOptions={{
                    style: {
                        fontSize: '12px',
                    },
                }}
            />
        </AuthProvider>
    );
}

export default App;
