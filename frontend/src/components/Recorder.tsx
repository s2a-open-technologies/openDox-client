/*
  openDox - Medical transcription platform
  Copyright (C) 2025  Christopher Abanilla <christopher.abanilla@gmx.de>

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

import {useState, useRef, useEffect, forwardRef, useImperativeHandle} from "react";
import {Mic, Square, Loader2, Pause, Play, X, ExternalLink, LogOut} from "lucide-react";
import {toast} from "sonner";
import {CreateRecord as GoCreateRecord, GetWSURL as GoGetWSURL, GetWhisperSettings as GoGetWhisperSettings, GetCategories as GoGetCategories} from "../../wailsjs/go/main/App";
import {InsertTranscript as GoInsertTranscript} from "../../wailsjs/go/main/App";
import {GetTaskStatus as GoGetTaskStatus, GetRecordBySession as GoGetRecordBySession} from "../../wailsjs/go/main/App";
import {ReprocessWithCategory as GoReprocessWithCategory} from "../../wailsjs/go/main/App";
import {OpenDashboard as GoOpenDashboard, CancelSession as GoCancelSession} from "../../wailsjs/go/main/App";
import {EventsOn, Environment} from "../../wailsjs/runtime/runtime";
import {BASE_URL} from "@/api";
import {useAuth} from "@/contexts/AuthContext";
import {ConfirmModal} from "@/components/ui/ConfirmModal";
import {VoiceVisualizer, BarVisualizer} from "@/components/ui/voice-visualizer";
import './Recorder.css';

interface RecorderProps {
    userId: string;
    title?: string;
    categoryId?: string | null;
    onFinalTranscription?: (sessionId: string, taskId?: string | number) => void;
    onRecordingStart?: (recordingData: any) => void;
    autoStart?: boolean;
    resumeSessionId?: string;
    onLogout?: () => void;
}

export interface RecorderHandle {
    waitForUploads: () => Promise<void>;
}

const Recorder = forwardRef<RecorderHandle, RecorderProps>(({userId, title, categoryId, onFinalTranscription, onRecordingStart, autoStart, resumeSessionId, onLogout}, ref) => {
    const {getToken} = useAuth();
    const [isRecording, setIsRecording] = useState(false);
    const [isProcessing, setIsProcessing] = useState(false);
    const [liveTranscript, setLiveTranscript] = useState("");
    const [stableTranscript, setStableTranscript] = useState("");
    const [isPaused, setIsPaused] = useState(false);
    const [isStopDialogOpen, setIsStopDialogOpen] = useState(false);
    const [isCancelDialogOpen, setIsCancelDialogOpen] = useState(false);
    const [sessionId, setSessionId] = useState<string | null>(null);
    const sessionIdRef = useRef<string | null>(null);
    const [analyser, setAnalyser] = useState<AnalyserNode | null>(null);
    // Auf macOS ist ⌘+⌥ (Cmd+Option) die Konvention für globale Shortcuts statt
    // Strg+Alt — siehe registerGlobalHotkeys/hotkeyMods in app.go.
    const [modKeyLabel, setModKeyLabel] = useState("Strg+Alt");
    // Erreichbarkeit des openDox-Servers (nur relevant im Leerlauf-Status, siehe
    // recorder-status unten — während einer Aufnahme zeigt der Punkt weiterhin
    // den Aufnahme-/Pause-Status).
    const [isBackendReachable, setIsBackendReachable] = useState(true);
    // ID des fertigen Records (sobald die Transkription vorliegt) — wird für den
    // nachträglichen Kategoriewechsel (Neuverarbeitung durch die KI) gebraucht.
    const [recordId, setRecordId] = useState<number | null>(null);

    const streamRef = useRef<MediaStream | null>(null);
    const socketRef = useRef<WebSocket | null>(null);

    const audioContextRef = useRef<AudioContext | null>(null);
    const workletRef = useRef<AudioWorkletNode | null>(null);
    const isPausedRef = useRef(false);
    const wsFinishedRef = useRef(false);
    const processingTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const pollTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    // Für den globalen Hotkey-Listener (unten): der spiegelt isRecording/isProcessing,
    // damit der einmalig registrierte Event-Handler immer den aktuellen Stand sieht statt
    // eines veralteten Closures.
    const isRecordingRef = useRef(false);
    const isProcessingRef = useRef(false);

    const [liveEnabled, setLiveEnabled] = useState(true);
    const [categories, setCategories] = useState<{id: number; name: string}[]>([]);
    const [selectedCategoryId, setSelectedCategoryId] = useState<number>(() => {
        const stored = localStorage.getItem(`selected_category_id_${userId}`);
        if (!stored) return 0;
        const parsed = parseInt(stored, 10);
        return isNaN(parsed) ? 0 : parsed;
    });
    const liveTranscriptRef = useRef<HTMLDivElement | null>(null);

    useEffect(() => {
        Environment().then((env) => {
            if (env.platform === "darwin") {
                setModKeyLabel("⌘+⌥");
            }
        });
    }, []);

    useEffect(() => {
        let cancelled = false;
        const checkReachability = () => {
            fetch(BASE_URL)
                .then(() => { if (!cancelled) setIsBackendReachable(true); })
                .catch(() => { if (!cancelled) setIsBackendReachable(false); });
        };
        checkReachability();
        const interval = setInterval(checkReachability, 15000);
        return () => {
            cancelled = true;
            clearInterval(interval);
        };
    }, []);

    useEffect(() => {
        const el = liveTranscriptRef.current;
        if (el) {
            el.scrollTop = el.scrollHeight;
        }
    }, [stableTranscript, liveTranscript]);

    useImperativeHandle(ref, () => ({
        waitForUploads: async () => {
            // No uploads needed — audio is sent via WebSocket PCM
        },
    }));

    const createRecord = async () => {
        const record = await GoCreateRecord(parseInt(userId), title || "Overlay-Aufnahme", selectedCategoryId, getToken());

        setSessionId(record.session_id);
        sessionIdRef.current = record.session_id;
        toast.success("Aufnahme gestartet");
        if (onRecordingStart) {
            onRecordingStart(record);
        }
        return record;
    };

    // Nach dem Stop läuft die finale Transkription + LLM-Zusammenfassung als
    // Hintergrund-Task auf dem Server; die WS-"finished"-Nachricht signalisiert nur den
    // Start davon, nicht das Ergebnis. Hier wird gepollt, bis der Task terminal ist, und
    // erst dann das fertige Record (inkl. Zusammenfassung) geladen.
    const pollTaskStatus = async (taskId: number, pollSessionId: string) => {
        if (sessionIdRef.current !== pollSessionId) return; // neue Session gestartet, verwerfen

        let status;
        try {
            status = await GoGetTaskStatus(taskId, getToken());
        } catch (err) {
            console.error("Task-Status-Abfrage fehlgeschlagen:", err);
            pollTimeoutRef.current = setTimeout(() => pollTaskStatus(taskId, pollSessionId), 2000);
            return;
        }
        if (sessionIdRef.current !== pollSessionId) return;

        if (status.status === "finished") {
            try {
                const record = await GoGetRecordBySession(pollSessionId, getToken());
                if (sessionIdRef.current !== pollSessionId) return;
                setStableTranscript(record.summarize || "");
                setLiveTranscript("");
                setRecordId(record.id);
                toast.success(`Transkription abgeschlossen – ${modKeyLabel}+V zum Einfügen`);
                if (onFinalTranscription) {
                    onFinalTranscription(pollSessionId, taskId);
                }
            } catch (err) {
                console.error(err);
                toast.error("Fehler beim Abrufen der fertigen Aufnahme");
            } finally {
                setIsProcessing(false);
                if (processingTimeoutRef.current) {
                    clearTimeout(processingTimeoutRef.current);
                    processingTimeoutRef.current = null;
                }
            }
            return;
        }

        if (status.status === "error") {
            setIsProcessing(false);
            if (processingTimeoutRef.current) {
                clearTimeout(processingTimeoutRef.current);
                processingTimeoutRef.current = null;
            }
            toast.error(status.error || "Verarbeitung fehlgeschlagen");
            return;
        }

        // pending / running / transcribing / processing_llm -> weiter pollen
        pollTimeoutRef.current = setTimeout(() => pollTaskStatus(taskId, pollSessionId), 2000);
    };

    // Kategoriewechsel bei bereits fertigem Transkript: Server ändert die Kategorie des
    // Records und stößt die KI-Zusammenfassung neu an (analog zum openDox-Web-Frontend).
    // Der /summarize-Endpoint ist synchron (gegen den echten Server verifiziert) — kein
    // Task-Polling nötig, die Response enthält den fertigen Text direkt.
    const reprocessWithCategory = async (newCategoryId: number) => {
        if (!recordId || !sessionIdRef.current) return;
        const pollSessionId = sessionIdRef.current;
        setIsProcessing(true);
        processingTimeoutRef.current = setTimeout(() => {
            setIsProcessing(false);
            toast.error("Zeitüberschreitung – Neuverarbeitung fehlgeschlagen");
        }, 90_000);

        try {
            const record = await GoReprocessWithCategory(recordId, newCategoryId, getToken());
            if (sessionIdRef.current !== pollSessionId) return;
            setStableTranscript(record.summarize || "");
            toast.success(`Neuverarbeitung abgeschlossen – ${modKeyLabel}+V zum Einfügen`);
        } catch (err) {
            console.error("Neuverarbeitung fehlgeschlagen:", err);
            toast.error("Neuverarbeitung fehlgeschlagen");
        } finally {
            if (processingTimeoutRef.current) {
                clearTimeout(processingTimeoutRef.current);
                processingTimeoutRef.current = null;
            }
            setIsProcessing(false);
        }
    };

    const openWebSocket = async (userId: string, session_id: string) => {
        // WebSocket-URL (inkl. Token) vom Go-Backend holen
        const wsURL = await GoGetWSURL(session_id, userId, getToken());
        const ws = new WebSocket(wsURL);
        socketRef.current = ws;

        ws.onopen = () => {
            console.log("WebSocket verbunden");
        };

        ws.onmessage = (event) => {
            const data = JSON.parse(event.data);

            if (data.partial) {
                setLiveTranscript(data.partial);
            }

            if (data.cleanup !== undefined) {
                // Periodic LLM cleanup replaced the full stable transcript
                setStableTranscript(data.cleanup);
                setLiveTranscript("");
            }

            if (data.text !== undefined) {
                if (data.restored) {
                    // Resume: backend replays the accumulated transcript - set, don't append.
                    setStableTranscript(data.text);
                } else {
                    setStableTranscript((prev) =>
                        (prev + " " + data.text).trim()
                    );
                }
                setLiveTranscript("");
            }

            if (data.status === "finished") {
                // Signalisiert nur, dass die Hintergrund-Transkription gestartet wurde -
                // isProcessing bleibt an, bis pollTaskStatus das fertige Ergebnis hat.
                wsFinishedRef.current = true;
                toast.success("Aufnahme wird verarbeitet...");
                if (data.task_id && sessionIdRef.current) {
                    pollTaskStatus(data.task_id, sessionIdRef.current);
                }
            }

            if (data.status === "error") {
                // WS-seitiger Fehler ohne Task (z.B. leerer Audio-Buffer) - hier gibt es
                // nichts zu pollen, sofort zurück in den Ruhezustand.
                wsFinishedRef.current = true;
                if (processingTimeoutRef.current) {
                    clearTimeout(processingTimeoutRef.current);
                    processingTimeoutRef.current = null;
                }
                setIsProcessing(false);
                toast.error(data.message || "Fehler bei der Aufnahme");
            }
        };

        ws.onerror = (error) => {
            console.error("WebSocket Fehler:", error);
            toast.error("WebSocket Fehler");
        };

        ws.onclose = () => {
            console.log("WebSocket geschlossen");
            if (!wsFinishedRef.current) {
                // Unexpected disconnect (e.g. Apache proxy dropped the connection)
                setIsRecording(false);
                setIsProcessing(false);
                if (processingTimeoutRef.current) {
                    clearTimeout(processingTimeoutRef.current);
                    processingTimeoutRef.current = null;
                }
                toast.error("Verbindung unterbrochen – bitte Aufnahme neu starten");
            }
            wsFinishedRef.current = false;
        };
    };

    const startRecording = async () => {
        try {
            // Verwaiste Polls einer vorherigen Session (z.B. abgebrochen) verwerfen.
            if (pollTimeoutRef.current) {
                clearTimeout(pollTimeoutRef.current);
                pollTimeoutRef.current = null;
            }

            let record;

            // Neue Aufnahme: altes Transkript verwerfen.
            // Bei Resume (unfertige Session) den Text vom Server restoren lassen.
            if (!resumeSessionId) {
                setStableTranscript("");
                setLiveTranscript("");
                setRecordId(null);
            }

            // Resume an existing unfinished session or create a new recording.
            if (resumeSessionId) {
                // Reuse the existing session_id - backend restores the live transcript
                // and new audio is appended to the same session.
                record = {session_id: resumeSessionId};
                setSessionId(resumeSessionId);
                sessionIdRef.current = resumeSessionId;
            } else {
                record = await createRecord();
            }

            await openWebSocket(userId, record.session_id);

            // 1) User-Audio holen
            const stream = await navigator.mediaDevices.getUserMedia({
                audio: {
                    channelCount: 1,
                    sampleRate: 16000,
                    echoCancellation: true,
                    noiseSuppression: true,
                }
            });
            streamRef.current = stream;

            // 2) AudioContext + Worklet + Visualizer
            const audioCtx = new AudioContext({sampleRate: 16000});
            if (audioCtx.state === "suspended") {
                await audioCtx.resume();
            }

            // Analyser for Visualization
            const analyserNode = audioCtx.createAnalyser();
            analyserNode.fftSize = 256;
            setAnalyser(analyserNode);

            // Inline Worklet: Garantiert, dass der Code geladen wird und Float32 -> Int16 konvertiert
            const workletCode = `
        class PCMProcessor extends AudioWorkletProcessor {
          process(inputs, outputs, parameters) {
            const input = inputs[0];
            if (input.length > 0) {
              // Sende rohes Float32 (4 Byte pro Sample) direkt an das Backend
              this.port.postMessage(input[0].buffer);
            }
            return true;
          }
        }
        registerProcessor("pcm-worklet", PCMProcessor);
      `;
            const blob = new Blob([workletCode], {type: "application/javascript"});
            const workletUrl = URL.createObjectURL(blob);
            await audioCtx.audioWorklet.addModule(workletUrl);

            const source = audioCtx.createMediaStreamSource(stream);
            const worklet = new AudioWorkletNode(audioCtx, "pcm-worklet");

            // Float32 → WebSocket
            worklet.port.onmessage = (e) => {
                if (socketRef.current?.readyState === WebSocket.OPEN && !isPausedRef.current) {
                    socketRef.current.send(e.data);
                }
            };

            // Connect Graph
            source.connect(analyserNode); // source -> analyser
            source.connect(worklet);
            // We don't connect to destination (speakers) to avoid feedback loop unless desired

            audioContextRef.current = audioCtx;
            workletRef.current = worklet;
            setIsRecording(true);
            setIsPaused(false);
            isPausedRef.current = false;
            setIsProcessing(false);
        } catch (err) {
            console.error(err);
            toast.error("Fehler beim Starten der Aufnahme");
        }
    };

    const stopRecording = () => {
        if (!isRecording) return;

        isPausedRef.current = false;

        try {workletRef.current?.disconnect();} catch (_) {
        }
        streamRef.current?.getTracks().forEach(track => track.stop());
        audioContextRef.current?.close();
        setAnalyser(null);

        setIsRecording(false);
        setIsPaused(false);
        isPausedRef.current = false;
        setIsProcessing(true);

        if (socketRef.current?.readyState === WebSocket.OPEN) {
            wsFinishedRef.current = false;
            socketRef.current.send(JSON.stringify({event: "stop"}));
            // Safety timeout: reset spinner if the backend never finishes processing
            // (covers the full stop -> transcribe -> summarize pipeline, not just the WS
            // round trip that only signals the background task has started).
            processingTimeoutRef.current = setTimeout(() => {
                if (pollTimeoutRef.current) {
                    clearTimeout(pollTimeoutRef.current);
                    pollTimeoutRef.current = null;
                }
                setIsProcessing(false);
                toast.error("Zeitüberschreitung – Aufnahme konnte nicht verarbeitet werden");
            }, 90_000);
        } else {
            setIsProcessing(false);
            toast.error("Verbindung bereits unterbrochen");
        }
    };

    // Bricht eine laufende (noch nicht gestoppte) Aufnahme vollständig ab - im Gegensatz
    // zu stopRecording() wird kein "stop"-Event gesendet, die Transkriptions-Pipeline
    // läuft also nie an. Spiegelt den "Abbrechen"-Button der Web-GUI.
    const cancelRecording = async () => {
        if (!isRecording) return;

        isPausedRef.current = true;
        wsFinishedRef.current = true; // erwarteter Close, keine "Verbindung unterbrochen"-Meldung
        try {workletRef.current?.disconnect();} catch (_) {
        }
        try {streamRef.current?.getTracks().forEach((t) => t.stop());} catch (_) {
        }
        try {audioContextRef.current?.close();} catch (_) {
        }
        try {socketRef.current?.close();} catch (_) {
        }
        setAnalyser(null);

        setIsRecording(false);
        setIsPaused(false);
        setIsProcessing(false);
        setStableTranscript("");
        setLiveTranscript("");
        setSessionId(null);
        sessionIdRef.current = null;
        setRecordId(null);

        try {
            await GoCancelSession(parseInt(userId), getToken());
            toast.success("Aufnahme abgebrochen");
        } catch (err) {
            console.error(err);
            toast.error("Fehler beim Abbrechen der Aufnahme");
        }
    };

    const handleCancelClick = () => {
        setIsCancelDialogOpen(true);
    };

    const confirmCancel = () => {
        setIsCancelDialogOpen(false);
        cancelRecording();
    };

    const pauseSession = () => {
        if (!isRecording) return;
        isPausedRef.current = true;
        audioContextRef.current?.suspend();
        setIsPaused(true);
    };

    const resumeSession = () => {
        audioContextRef.current?.resume();
        setIsPaused(false);
        isPausedRef.current = false;
    };

    const acceptAndInsert = (viaHotkey = false) => {
        if (!stableTranscript) return;
        GoInsertTranscript(stableTranscript, viaHotkey);
    };

    const handleStopClick = () => {
        setIsStopDialogOpen(true);
    };

    const confirmStop = () => {
        setIsStopDialogOpen(false);
        stopRecording();
    };

    useEffect(() => {
        // Load settings to determine if live transcription is enabled
        GoGetWhisperSettings()
            .then((data) => {
                setLiveEnabled(data.live_backend !== "disabled");
            })
            .catch(() => setLiveEnabled(true));

        // Kategorien vom Server laden
        GoGetCategories(getToken())
            .then((cats) => {
                const list = cats || [];
                setCategories(list);
                const valid = list.some((c) => c.id === selectedCategoryId);
                if (!valid) {
                    const allgemein = list.find((c) => c.name === "Allgemein");
                    setSelectedCategoryId(allgemein ? allgemein.id : (list[0]?.id ?? 0));
                }
            })
            .catch(() => setCategories([]));
    }, []);

    useEffect(() => {
        localStorage.setItem(`selected_category_id_${userId}`, String(selectedCategoryId));
    }, [selectedCategoryId, userId]);

    useEffect(() => {
        isRecordingRef.current = isRecording;
        isProcessingRef.current = isProcessing;
    }, [isRecording, isProcessing]);

    // "Latest ref" auf die aktuellen Werte dieser Funktionen: die Hotkey-Effekte unten
    // abonnieren nur einmal beim Mount, würden aber sonst für immer die allererste Version
    // dieser Funktionen aufrufen - und die hat z.B. selectedCategoryId beim Mount (0)
    // eingefangen, nicht die später vom Nutzer im Dropdown gewählte Kategorie. Gilt genauso
    // für pauseSession, das intern isRecording aus dem State liest.
    const startRecordingRef = useRef(startRecording);
    const stopRecordingRef = useRef(stopRecording);
    const pauseSessionRef = useRef(pauseSession);
    const resumeSessionRef = useRef(resumeSession);
    const stableTranscriptRef = useRef(stableTranscript);
    const acceptAndInsertRef = useRef(acceptAndInsert);
    useEffect(() => {
        startRecordingRef.current = startRecording;
        stopRecordingRef.current = stopRecording;
        pauseSessionRef.current = pauseSession;
        resumeSessionRef.current = resumeSession;
        acceptAndInsertRef.current = acceptAndInsert;
        stableTranscriptRef.current = stableTranscript;
    });

    useEffect(() => {
        // Globale Hotkeys (Ctrl+Alt+R / Ctrl+Alt+P, siehe app.go) steuern die Aufnahme
        // unabhängig davon, welches Fenster gerade fokussiert ist.
        const offToggle = EventsOn("hotkey:toggle-record", () => {
            if (isProcessingRef.current) return; // wird gerade verarbeitet, ignorieren
            if (isRecordingRef.current) {
                // Bewusst ohne Bestätigungsdialog: der Sinn des Hotkeys ist ja gerade,
                // nicht zurück ins Overlay klicken zu müssen.
                stopRecordingRef.current();
            } else {
                startRecordingRef.current();
            }
        });
        const offPause = EventsOn("hotkey:toggle-pause", () => {
            if (!isRecordingRef.current || isProcessingRef.current) return;
            if (isPausedRef.current) {
                resumeSessionRef.current();
            } else {
                pauseSessionRef.current();
            }
        });
        const offAccept = EventsOn("hotkey:accept-insert", () => {
            if (isRecordingRef.current || isProcessingRef.current) return;
            if (!stableTranscriptRef.current) return;
            acceptAndInsertRef.current(true);
        });
        return () => {
            offToggle();
            offPause();
            offAccept();
        };
    }, []);

    useEffect(() => {
        // Resume an existing unfinished session or auto-start a new recording.
        // The server (session_store) holds the live state across reconnects.
        if (resumeSessionId || autoStart) {
            startRecording();
        }

        return () => {
            // On unmount without stopping (e.g. "Zur Übersicht"): tear down capture and
            // close the socket as a plain disconnect -> backend keeps the session so it
            // can be resumed later.
            isPausedRef.current = true;
            try {workletRef.current?.disconnect();} catch (_) {
            }
            try {streamRef.current?.getTracks().forEach((t) => t.stop());} catch (_) {
            }
            try {audioContextRef.current?.close();} catch (_) {
            }
            try {socketRef.current?.close();} catch (_) {
            }
            if (pollTimeoutRef.current) {
                clearTimeout(pollTimeoutRef.current);
                pollTimeoutRef.current = null;
            }
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    return (
        <div className="recorder">
            {/* Status-Leiste */}
            <div className="recorder-status">
                <span
                    className={`recorder-dot ${
                        isRecording
                            ? (isPaused ? "paused" : "recording")
                            : (isBackendReachable ? "reachable" : "unreachable")
                    }`}
                    title={isRecording ? undefined : (isBackendReachable ? "openDox-Server erreichbar" : "openDox-Server nicht erreichbar")}
                />
                <span className="recorder-status-text">
                    {isPaused ? "Aufnahme pausiert" : isRecording ? "Aufnahme läuft..." : (isBackendReachable ? "Bereit" : "Nicht erreichbar")}
                </span>

                {/* Rechts gruppierte Aktionen: Kategorie-Dropdown + Logout, getrennt
                    voneinander und vom global fixierten Close-Button (App.tsx). */}
                <div className="recorder-status-actions">
                    {!isRecording && !isProcessing && categories.length > 0 && (
                        <select
                            className="recorder-category-select"
                            value={selectedCategoryId}
                            onChange={(e) => {
                                const newCategoryId = parseInt(e.target.value);
                                setSelectedCategoryId(newCategoryId);
                                // Liegt bereits ein fertiges Transkript vor, löst der
                                // Kategoriewechsel eine Neuverarbeitung durch die KI aus
                                // (wie im openDox-Web-Frontend) statt nur die Vorauswahl
                                // für die nächste Aufnahme zu setzen.
                                if (stableTranscript && recordId) {
                                    reprocessWithCategory(newCategoryId);
                                }
                            }}
                            title="Kategorie auswählen"
                        >
                            {categories.map((cat) => (
                                <option key={cat.id} value={cat.id}>{cat.name}</option>
                            ))}
                        </select>
                    )}

                    {onLogout && (
                        <button
                            className="recorder-logout-btn"
                            onClick={onLogout}
                            title="Abmelden"
                            aria-label="Abmelden"
                        >
                            <LogOut size={14}/>
                        </button>
                    )}
                </div>
            </div>

            {/* Transkriptions-Bereich */}
            <div ref={liveTranscriptRef} className="recorder-transcript" onContextMenu={(e) => e.preventDefault()}>
                {liveEnabled ? (
                    <div className="recorder-transcript-inner">
                        {(stableTranscript || liveTranscript) ? (
                            <p className="recorder-text select-none">
                                {stableTranscript}
                                {liveTranscript && (
                                    <span className="recorder-text-live">{liveTranscript}</span>
                                )}
                            </p>
                        ) : (
                            <div className="recorder-placeholder">
                                <p>Sprechen Sie, um die Transkription zu starten</p>
                            </div>
                        )}
                    </div>
                ) : (
                    <div className="recorder-orb">
                        {isRecording && !isPaused && (
                            <VoiceVisualizer analyser={analyser}/>
                        )}
                    </div>
                )}
            </div>

            {/* Live bar spectrum, direkt ueber den Controls */}
            {liveEnabled && isRecording && !isPaused && (
                <BarVisualizer analyser={analyser}/>
            )}

            {/* Footer / Controls */}
            <div className="recorder-controls">
                {!isProcessing ? (
                    <>
                        {isRecording && (
                            <button
                                onClick={handleCancelClick}
                                className="recorder-cancel-btn"
                                aria-label="Aufnahme abbrechen"
                                title="Aufnahme abbrechen"
                            >
                                <X className="icon-fill"/>
                            </button>
                        )}

                        {isRecording && (
                            <button
                                onClick={isPaused ? resumeSession : pauseSession}
                                className={`recorder-pause-btn ${isPaused ? "resuming" : "pausing"}`}
                                aria-label={isPaused ? "Fortsetzen" : "Pausieren"}
                            >
                                {isPaused ? <Play className="icon-fill"/> : <Pause className="icon-fill"/>}
                            </button>
                        )}

                        <button
                            onClick={isRecording ? handleStopClick : startRecording}
                            className={`recorder-mic-btn ${isRecording ? "recording" : ""}`}
                            aria-label={isRecording ? "Stop" : "Aufnahme starten"}
                        >
                            {isRecording ? (
                                <Square className="icon-fill"/>
                            ) : (
                                <Mic className="icon-fill"/>
                            )}
                            {isRecording && (
                                <span className="recorder-pulse-ring"/>
                            )}
                        </button>
                    </>
                ) : (
                    <div className="recorder-processing">
                        <Loader2 className="icon-spin"/>
                        <span>Verarbeite...</span>
                    </div>
                )}
            </div>

            {/* Aktions-Buttons unter dem Aufnahme-Button (nur wenn fertig transkribiert) */}
            {!isProcessing && stableTranscript && !isRecording && (
                <div className="recorder-actions">
                    <button
                        onClick={() => acceptAndInsert(false)}
                        className="recorder-insert-btn"
                        title={`In fokussiertes Fenster einfügen (${modKeyLabel}+V)`}
                    >
                        Akzeptieren und einfügen
                    </button>
                    <button
                        onClick={() => GoOpenDashboard()}
                        className="recorder-insert-btn"
                        title="Aufnahme in openDox öffnen"
                    >
                        <ExternalLink className="icon-inline"/>
                        In openDox öffnen
                    </button>
                </div>
            )}

            {!isProcessing && (
                <div className="recorder-shortcuts">
                    <span className="shortcut">
                        <kbd>{modKeyLabel}+R</kbd> {isRecording ? "Stop" : "Start"}
                    </span>
                    {isRecording && (
                        <span className="shortcut">
                            <kbd>{modKeyLabel}+P</kbd> {isPaused ? "Fortsetzen" : "Pausieren"}
                        </span>
                    )}
                    {stableTranscript && !isRecording && (
                        <span className="shortcut">
                            <kbd>{modKeyLabel}+V</kbd> Akzeptieren &amp; einfügen
                        </span>
                    )}
                </div>
            )}

            <ConfirmModal
                open={isStopDialogOpen}
                title="Aufnahme beenden"
                description="Möchten Sie die Aufnahme wirklich beenden und speichern?"
                confirmLabel="Aufnahme beenden"
                cancelLabel="Abbrechen"
                onConfirm={confirmStop}
                onCancel={() => setIsStopDialogOpen(false)}
            />

            <ConfirmModal
                open={isCancelDialogOpen}
                title="Aufnahme abbrechen"
                description="Aufnahme wirklich abbrechen? Die Aufnahme wird nicht gespeichert und kann nicht wiederhergestellt werden."
                confirmLabel="Aufnahme abbrechen"
                cancelLabel="Zurück"
                onConfirm={confirmCancel}
                onCancel={() => setIsCancelDialogOpen(false)}
            />
        </div>
    );
});

export default Recorder;
