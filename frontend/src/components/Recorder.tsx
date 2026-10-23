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

import {useState, useRef, useEffect, forwardRef, useImperativeHandle} from "react";
import {Mic, Square, Loader2, Pause, Play, X, ExternalLink, LogOut, Check, Eye, EyeOff, HelpCircle} from "lucide-react";
import {toast} from "sonner";
import {CreateRecord as GoCreateRecord, GetWSURL as GoGetWSURL, GetLiveViewDefault as GoGetLiveViewDefault, GetWhisperSettings as GoGetWhisperSettings, GetCategories as GoGetCategories} from "../../wailsjs/go/main/App";
import {InsertTranscript as GoInsertTranscript, SetOverlayNoActivate as GoSetOverlayNoActivate} from "../../wailsjs/go/main/App";
import {GetInsertTarget as GoGetInsertTarget} from "../../wailsjs/go/main/App";
import type {main} from "../../wailsjs/go/models";
import {GetTaskStatus as GoGetTaskStatus, GetRecordBySession as GoGetRecordBySession} from "../../wailsjs/go/main/App";
import {ReprocessWithCategory as GoReprocessWithCategory} from "../../wailsjs/go/main/App";
import {OpenDashboard as GoOpenDashboard, CancelSession as GoCancelSession} from "../../wailsjs/go/main/App";
import {CheckAccessibility as GoCheckAccessibility, CheckHotkeys as GoCheckHotkeys} from "../../wailsjs/go/main/App";
import {EventsOn, Environment} from "../../wailsjs/runtime/runtime";
import {useAuth} from "@/contexts/AuthContext";
import {useSettings} from "@/contexts/SettingsContext";
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

// Meldungen für Fehlercodes, mit denen der Server den WebSocket ablehnt und schließt.
// Unbekannte Codes zeigen den (englischen) Fehlertext des Servers.
const SERVER_ERROR_MESSAGES: Record<string, string> = {
    recording_audio_unavailable:
        "Diese Aufnahme kann hier nicht fortgesetzt werden: Ihr bisheriges Audio liegt auf einer anderen Server-Instanz. Bitte die Aufnahme neu starten.",
};

const Recorder = forwardRef<RecorderHandle, RecorderProps>(({userId, title, categoryId, onFinalTranscription, onRecordingStart, autoStart, resumeSessionId, onLogout}, ref) => {
    const {getToken} = useAuth();
    const {settings} = useSettings();
    const restURL = settings?.server_url || "http://localhost:10000";
    const wsBaseURL = restURL.startsWith('https://')
        ? `wss://${restURL.slice(8)}`
        : `ws://${restURL.slice(7)}`;
    const [isRecording, setIsRecording] = useState(false);
    const [isProcessing, setIsProcessing] = useState(false);
    const [liveTranscript, setLiveTranscript] = useState("");
    const [stableTranscript, setStableTranscript] = useState("");
    const [isPaused, setIsPaused] = useState(false);
    const [helpOpen, setHelpOpen] = useState(false);
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
    // Gesetzt, wenn der Server den Abbruch mit einem Fehlercode erklärt hat -
    // dann zeigt onclose nicht zusätzlich "Verbindung unterbrochen".
    const serverErrorShownRef = useRef(false);
    // Unfertige Session vom App-Start (App.tsx, GetActiveSession): wird nur beim nächsten
    // Start EINMAL fortgesetzt. Bliebe sie dauerhaft gesetzt, würde jede weitere "Neue
    // Aufnahme" wieder dieselbe session_id nehmen und an die alte Aufnahme anhängen.
    const pendingResumeRef = useRef<string | undefined>(resumeSessionId);
    useEffect(() => {
        pendingResumeRef.current = resumeSessionId;
    }, [resumeSessionId]);
    const processingTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const pollTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    // Für den globalen Hotkey-Listener (unten): der spiegelt isRecording/isProcessing,
    // damit der einmalig registrierte Event-Handler immer den aktuellen Stand sieht statt
    // eines veralteten Closures.
    const isRecordingRef = useRef(false);
    const isProcessingRef = useRef(false);
    // Push-to-talk des Tastenmikrofons (hotkey:mic-record-down/-up, Hold-Modus): Die Aufnahme
    // wurde per gehaltener Record-Taste gestartet, das fertige Ergebnis wird dann automatisch
    // eingefügt. micStartingRef/micReleasedEarlyRef fangen ein Loslassen ab, bevor der
    // (asynchrone) Aufnahmestart durch ist.
    const autoInsertRef = useRef(false);
    const micStartingRef = useRef(false);
    const micReleasedEarlyRef = useRef(false);

    // Live-Ansicht: der Server kann sie global abschalten (live_backend "disabled", serverLive);
    // zusätzlich der Nutzer (Umschalter im Overlay, localStorage) bzw. settings.json (live_view,
    // Default). liveEnabled ist das Ergebnis aus beidem — bei false zeigt das Overlay nur den
    // Orb und der Server startet keine Live-ASR (live=0 am WebSocket).
    const [serverLive, setServerLive] = useState(true);
    const [liveView, setLiveView] = useState<boolean>(() => {
        try {
            const v = localStorage.getItem("live_view");
            if (v === "1") return true;
            if (v === "0") return false;
        } catch (_) { /* localStorage nicht verfügbar */ }
        return true;
    });
    const liveEnabled = serverLive && liveView;
    const liveEnabledRef = useRef(true);
    liveEnabledRef.current = liveEnabled;
    const toggleLiveView = () => {
        const next = !liveView;
        setLiveView(next);
        try {localStorage.setItem("live_view", next ? "1" : "0");} catch (_) { /* egal */ }
    };
    const [categories, setCategories] = useState<{id: number; name: string}[]>([]);
    const [selectedCategoryId, setSelectedCategoryId] = useState<number>(() => {
        const stored = localStorage.getItem(`selected_category_id_${userId}`);
        if (!stored) return 0;
        const parsed = parseInt(stored, 10);
        return isNaN(parsed) ? 0 : parsed;
    });
    const liveTranscriptRef = useRef<HTMLDivElement | null>(null);

    // Mikrofon-Auswahl: Liste der Eingabegeräte + persistierte Auswahl
    // ("" = System-Standard). Wechsel nur im Leerlauf möglich — während einer
    // Aufnahme bleibt das gerade verwendete Gerät aktiv.
    const [micDevices, setMicDevices] = useState<MediaDeviceInfo[]>([]);
    const [selectedMicId, setSelectedMicId] = useState<string>(() => {
        return localStorage.getItem(`selected_mic_id_${userId}`) || "";
    });
    const [micMenuOpen, setMicMenuOpen] = useState(false);
    const selectedMicIdRef = useRef(selectedMicId);

    useEffect(() => {
        Environment().then((env) => {
            if (env.platform === "darwin") {
                setModKeyLabel("⌘+⌥");
            }
        });
    }, []);

    // Windows: Overlay nimmt bei Klicks keinen Fokus, damit "Akzeptieren und
    // einfügen" ins Zielprogramm tippt wie der Hotkey. Beim Verlassen (Login-Screen
    // braucht Tastatureingaben) wieder aus. No-Op auf macOS/Linux.
    useEffect(() => {
        GoSetOverlayNoActivate(true);
        return () => { GoSetOverlayNoActivate(false); };
    }, []);

    useEffect(() => {
        let cancelled = false;
        const checkReachability = () => {
            fetch(restURL)
                .then(() => { if (!cancelled) setIsBackendReachable(true); })
                .catch(() => { if (!cancelled) setIsBackendReachable(false); });
        };
        checkReachability();
        const interval = setInterval(checkReachability, 15000);
        return () => {
            cancelled = true;
            clearInterval(interval);
        };
    }, [restURL]);

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
        const record = await GoCreateRecord(parseInt(userId), title || "Client-Aufnahme", selectedCategoryId, getToken());

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
                const autoInsert = autoInsertRef.current;
                autoInsertRef.current = false;
                if (autoInsert && record.summarize) {
                    acceptAndInsertRef.current(true, record.summarize);
                } else {
                    toast.success(`Transkription abgeschlossen – ${modKeyLabel}+V zum Einfügen`);
                }
                if (onFinalTranscription) {
                    onFinalTranscription(pollSessionId, taskId);
                }
            } catch (err) {
                console.error(err);
                autoInsertRef.current = false;
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
            autoInsertRef.current = false;
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
        // WebSocket-URL direkt aus den Settings bauen (Fallback wenn Go-Binding nicht da ist)
        let wsURL: string;
        try {
            wsURL = await GoGetWSURL(session_id, userId, getToken(), liveEnabledRef.current);
        } catch {
            wsURL = `${wsBaseURL}/api/v1/records/ws/record?session_id=${encodeURIComponent(session_id)}&user_id=${encodeURIComponent(userId)}&token=${encodeURIComponent(getToken())}${liveEnabledRef.current ? "" : "&live=0"}`;
        }
        const ws = new WebSocket(wsURL);
        socketRef.current = ws;

        ws.onopen = () => {
            console.log("WebSocket verbunden");
        };

        ws.onmessage = (event) => {
            const data = JSON.parse(event.data);

            if (data.error && data.code) {
                // Der Server lehnt ab und schließt den Socket direkt danach (z.B. wenn das
                // bisherige Audio einer fortgesetzten Aufnahme auf einer anderen Instanz liegt).
                serverErrorShownRef.current = true;
                toast.error(SERVER_ERROR_MESSAGES[data.code] ?? data.error);
                return;
            }

            if (data.partial && liveEnabledRef.current) {
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
                if (!serverErrorShownRef.current) {
                    toast.error("Verbindung unterbrochen – bitte Aufnahme neu starten");
                }
            }
            wsFinishedRef.current = false;
            serverErrorShownRef.current = false;
        };
    };

    const startRecording = async (opts?: {autoInsert?: boolean}) => {
        // Nur ein ausdrückliches autoInsert:true (Tastenmikrofon, Hold-Modus) schaltet das
        // Auto-Einfügen ein; jeder andere Start (Button, Hotkey, Resume) setzt es zurück.
        autoInsertRef.current = opts?.autoInsert === true;
        try {
            // Verwaiste Polls einer vorherigen Session (z.B. abgebrochen) verwerfen.
            if (pollTimeoutRef.current) {
                clearTimeout(pollTimeoutRef.current);
                pollTimeoutRef.current = null;
            }

            let record;
            const resumeId = pendingResumeRef.current;
            pendingResumeRef.current = undefined;

            // Neue Aufnahme: altes Transkript verwerfen.
            // Bei Resume (unfertige Session) den Text vom Server restoren lassen.
            if (!resumeId) {
                setStableTranscript("");
                setLiveTranscript("");
                setRecordId(null);
            }

            // Resume an existing unfinished session or create a new recording.
            if (resumeId) {
                // Reuse the existing session_id - backend restores the live transcript
                // and new audio is appended to the same session.
                record = {session_id: resumeId};
                setSessionId(resumeId);
                sessionIdRef.current = resumeId;
            } else {
                record = await createRecord();
            }

            await openWebSocket(userId, record.session_id);

            // 1) User-Audio holen — deviceId aus dem Ref, weil startRecording über
            //    Hotkey-/Mount-Closures laufen kann, die einen veralteten State sähen.
            const stream = await navigator.mediaDevices.getUserMedia({
                audio: {
                    channelCount: 1,
                    sampleRate: 16000,
                    echoCancellation: true,
                    noiseSuppression: true,
                    ...(selectedMicIdRef.current
                        ? {deviceId: {exact: selectedMicIdRef.current}}
                        : {}),
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
        } catch (err: any) {
            console.error(err);
            // Auf macOS schlägt getUserMedia ohne NSMicrophoneUsageDescription in der
            // Info.plist (oder verweigerte TCC-Berechtigung) fehl — das ist der
            // häufigste Grund, dass der Button nicht umschaltet, obwohl die Session
            // auf dem Server bereits angelegt ist.
            const msg = err?.name === "NotAllowedError"
                ? "Mikrofonzugriff verweigert – in macOS-Einstellungen prüfen"
                : err?.name === "NotFoundError"
                    ? "Kein Mikrofon gefunden"
                    : "Fehler beim Starten der Aufnahme";
            toast.error(msg);
            autoInsertRef.current = false;
            micReleasedEarlyRef.current = false;
            // Verwaiste Session auf dem Server abbrechen, damit keine leere Aufnahme
            // im Web-Frontend liegen bleibt. wsFinishedRef=true verhindert, dass
            // onclose "Verbindung unterbrochen" toast.
            wsFinishedRef.current = true;
            try {socketRef.current?.close();} catch (_) {
            }
            socketRef.current = null;
            if (sessionIdRef.current) {
                try {await GoCancelSession(parseInt(userId), getToken());} catch (_) {
                }
                setSessionId(null);
                sessionIdRef.current = null;
            }
            setAnalyser(null);
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
                autoInsertRef.current = false;
                setIsProcessing(false);
                toast.error("Zeitüberschreitung – Aufnahme konnte nicht verarbeitet werden");
            }, 90_000);
        } else {
            autoInsertRef.current = false;
            setIsProcessing(false);
            toast.error("Verbindung bereits unterbrochen");
        }
    };

    // Bricht eine laufende (noch nicht gestoppte) Aufnahme vollständig ab - im Gegensatz
    // zu stopRecording() wird kein "stop"-Event gesendet, die Transkriptions-Pipeline
    // läuft also nie an. Spiegelt den "Abbrechen"-Button der Web-GUI.
    const cancelRecording = async () => {
        if (!isRecording) return;

        autoInsertRef.current = false;
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

    // Einfügeziel (nur unter Windows befüllt): zeigt, in welches Programm "Akzeptieren und
    // einfügen" schreibt. Ist per settings.json eine Allow-Liste konfiguriert (Standard:
    // keine) und das Ziel nicht darauf, wird vorher nachgefragt.
    const [insertTarget, setInsertTarget] = useState<main.InsertTarget | null>(null);
    const [pendingInsert, setPendingInsert] = useState<{target: main.InsertTarget; viaHotkey: boolean; text: string} | null>(null);
    const insertVisible = !isProcessing && !!stableTranscript && !isRecording;
    useEffect(() => {
        if (!insertVisible) {
            setInsertTarget(null);
            setPendingInsert(null);
            return;
        }
        let cancelled = false;
        const poll = () => {
            GoGetInsertTarget()
                .then((t) => { if (!cancelled) setInsertTarget(t); })
                .catch(() => { /* Die Anzeige ist optional. */ });
        };
        poll();
        const id = setInterval(poll, 1500);
        return () => { cancelled = true; clearInterval(id); };
    }, [insertVisible]);

    const insertNow = (viaHotkey: boolean, text: string = stableTranscript) => {
        setPendingInsert(null);
        GoInsertTranscript(text, viaHotkey).catch((err: unknown) => {
            toast.error(String(err), {duration: 12000});
        });
    };

    // text: für das Auto-Einfügen nach dem Loslassen der Mic-Taste, wo der State
    // stableTranscript im aufrufenden Closure noch der alte (leere) Stand ist.
    const acceptAndInsert = (viaHotkey = false, text: string = stableTranscript) => {
        if (!text) return;
        // Standard (keine Allow-Liste): wie bisher direkt einfügen — der Client ist ein
        // allgemeines Diktierwerkzeug, nicht nur für ein bestimmtes Zielprogramm.
        if (insertTarget && !insertTarget.list_active) {
            insertNow(viaHotkey, text);
            return;
        }
        // Liste aktiv oder Ziel noch unbekannt: das Ziel jetzt frisch prüfen.
        GoGetInsertTarget()
            .then((target) => {
                if (target.list_active && !target.allowed) {
                    setPendingInsert({target, viaHotkey, text});
                } else {
                    insertNow(viaHotkey, text);
                }
            })
            .catch(() => insertNow(viaHotkey, text)); // Prüfung nicht möglich: wie bisher einfügen
    };

    const confirmPendingInsert = () => {
        if (!pendingInsert) return;
        const {target: asked, viaHotkey, text} = pendingInsert;
        // Der Fokus kann seit der Rückfrage gewechselt haben: bei einem anderen, ebenfalls
        // nicht freigegebenen Ziel erneut nachfragen statt dorthin einzufügen.
        GoGetInsertTarget()
            .then((now) => {
                if (now.list_active && !now.allowed && now.process !== asked.process) {
                    setPendingInsert({target: now, viaHotkey, text});
                } else {
                    insertNow(viaHotkey, text);
                }
            })
            .catch(() => insertNow(viaHotkey, text));
    };

    const handleStopClick = () => {
        setIsStopDialogOpen(true);
    };

    const confirmStop = () => {
        setIsStopDialogOpen(false);
        stopRecording();
    };

    useEffect(() => {
        // Auf macOS braucht die App zwei separate Berechtigungen:
        //   - Bedienhilfen (Accessibility)    → Tastatur-Injektion (Cmd+V)
        //   - Eingabeüberwachung (Input Mon.) → systemweite Hotkeys (CGEventTap)
        // Die Go-Seite löst beim Start den Bedienhilfen-Dialog aus, aber
        // Eingabeüberwachung muss der Nutzer manuell in den Systemeinstellungen
        // erlauben. Hier wird nach dem Mount geprüft und bei Bedarf ein
        // sichtbarer Hinweis gezeigt — ein still fehlender Hotkey ist sonst
        // schwer zu debuggen. Auf Linux/Windows liefern die Stubs immer true.
        Promise.all([
            GoCheckAccessibility().catch(() => true),
            GoCheckHotkeys().catch(() => true),
        ]).then(([axOk, hkOk]: [boolean, boolean]) => {
            if (!axOk && !hkOk) {
                toast.error(
                    "Bedienhilfen und Eingabeüberwachung fehlen – in den macOS-Systemeinstellungen unter Datenschutz → Bedienhilfen und Eingabeüberwachung erlauben.",
                    {duration: 12000}
                );
            } else if (!axOk) {
                toast.error(
                    "Bedienhilfen-Berechtigung fehlt – Einfügen deaktiviert. In den macOS-Systemeinstellungen unter Datenschutz → Bedienhilfen erlauben.",
                    {duration: 12000}
                );
            } else if (!hkOk) {
                toast.error(
                    "Eingabeüberwachung-Berechtigung fehlt – Hotkeys deaktiviert. In den macOS-Systemeinstellungen unter Datenschutz → Eingabeüberwachung erlauben.",
                    {duration: 12000}
                );
            }
        });
    }, []);

    useEffect(() => {
        // Default der Live-Ansicht (live_view aus Server-Richtlinie bzw. settings.json), sofern
        // der Nutzer nichts gewählt hat. Ändert der Admin ihn am Server, kommt policy:changed.
        const applyLiveViewDefault = () => {
            try {
                if (localStorage.getItem("live_view") === null) {
                    GoGetLiveViewDefault().then(setLiveView).catch(() => { /* Default true */ });
                }
            } catch (_) { /* localStorage nicht verfügbar */ }
        };
        applyLiveViewDefault();
        const offPolicy = EventsOn("policy:changed", applyLiveViewDefault);

        // Load settings to determine if live transcription is enabled
        GoGetWhisperSettings()
            .then((data) => {
                setServerLive(data.live_backend !== "disabled");
            })
            .catch(() => setServerLive(true));

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

        return () => offPolicy();
    }, []);

    useEffect(() => {
        localStorage.setItem(`selected_category_id_${userId}`, String(selectedCategoryId));
    }, [selectedCategoryId, userId]);

    // Mikrofonliste laden + auf Hot-Plug (USB-Mikro an/ab) reagieren.
    // Vor der ersten getUserMedia-Berechtigung liefern WebKit-Browser leere
    // Labels — die UI zeigt dann einen Fallback-Text ("Mikrofon N").
    useEffect(() => {
        const refreshMics = () => {
            navigator.mediaDevices?.enumerateDevices()
                .then((devices) => {
                    const mics = devices.filter((d) => d.kind === "audioinput");
                    setMicDevices(mics);
                    // Verschwindenes Gerät (z.B. USB-Mikro gezogen) aus der
                    // Auswahl nehmen, sonst schlägt getUserMedia({exact}) fehl.
                    setSelectedMicId((current) =>
                        current && !mics.some((m) => m.deviceId === current) ? "" : current
                    );
                })
                .catch(() => setMicDevices([]));
        };
        refreshMics();
        const md = navigator.mediaDevices;
        if (!md) return;
        md.addEventListener("devicechange", refreshMics);
        return () => md.removeEventListener("devicechange", refreshMics);
    }, []);

    useEffect(() => {
        localStorage.setItem(`selected_mic_id_${userId}`, selectedMicId);
        selectedMicIdRef.current = selectedMicId;
    }, [selectedMicId, userId]);

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
        const offMicDown = EventsOn("hotkey:mic-record-down", () => {
            if (isRecordingRef.current || isProcessingRef.current || micStartingRef.current) return;
            micStartingRef.current = true;
            micReleasedEarlyRef.current = false;
            Promise.resolve(startRecordingRef.current({autoInsert: true})).finally(() => {
                micStartingRef.current = false;
            });
        });
        const offMicUp = EventsOn("hotkey:mic-record-up", () => {
            if (isRecordingRef.current) {
                stopRecordingRef.current();
            } else if (micStartingRef.current) {
                // Losgelassen, bevor die Aufnahme lief: der Effekt auf isRecording stoppt sie.
                micReleasedEarlyRef.current = true;
            }
        });
        return () => {
            offToggle();
            offPause();
            offAccept();
            offMicDown();
            offMicUp();
        };
    }, []);

    useEffect(() => {
        if (isRecording && micReleasedEarlyRef.current) {
            micReleasedEarlyRef.current = false;
            stopRecordingRef.current();
        }
    }, [isRecording]);

    useEffect(() => {
        // Resume an existing unfinished session or auto-start a new recording.
        // The server (session_store) holds the live state across reconnects.
        if (pendingResumeRef.current || autoStart) {
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

                {/* Rechts gruppierte Aktionen: Mikrofon-Wahl + Kategorie-Dropdown +
                    Logout, getrennt voneinander und vom global fixierten
                    Close-Button (App.tsx). */}
                <div className="recorder-status-actions">
                    {serverLive && !isRecording && !isProcessing && (
                        <button
                            className="recorder-logout-btn"
                            onClick={toggleLiveView}
                            title={liveView ? "Live-Ansicht ausschalten" : "Live-Ansicht einschalten"}
                            aria-label={liveView ? "Live-Ansicht ausschalten" : "Live-Ansicht einschalten"}
                            aria-pressed={liveView}
                        >
                            {liveView ? <Eye size={14}/> : <EyeOff size={14}/>}
                        </button>
                    )}
                    <button
                        className="recorder-logout-btn"
                        onClick={() => setHelpOpen(true)}
                        title="Tastenkürzel"
                        aria-label="Tastenkürzel anzeigen"
                    >
                        <HelpCircle size={14}/>
                    </button>
                    {!isRecording && !isProcessing && micDevices.length > 1 && (
                        <div className="recorder-mic-wrap">
                            <button
                                className={`recorder-mic-select-btn ${micMenuOpen ? "open" : ""}`}
                                onClick={() => setMicMenuOpen((o) => !o)}
                                title="Mikrofon auswählen"
                                aria-label="Mikrofon auswählen"
                                aria-haspopup="menu"
                                aria-expanded={micMenuOpen}
                            >
                                <Mic size={14}/>
                            </button>
                            {micMenuOpen && (
                                <>
                                    {/* Fängt Klicks außerhalb des Menüs ab */}
                                    <div className="recorder-mic-backdrop" onClick={() => setMicMenuOpen(false)}/>
                                    <div className="recorder-mic-menu" role="menu">
                                        <div className="recorder-mic-menu-title">Mikrofon</div>
                                        {micDevices.map((mic, idx) => (
                                            <button
                                                key={mic.deviceId || idx}
                                                role="menuitemradio"
                                                aria-checked={mic.deviceId === selectedMicId}
                                                className={`recorder-mic-item ${mic.deviceId === selectedMicId ? "active" : ""}`}
                                                onClick={() => {
                                                    setSelectedMicId(mic.deviceId);
                                                    setMicMenuOpen(false);
                                                }}
                                            >
                                                <span className="recorder-mic-item-label">
                                                    {mic.label || `Mikrofon ${idx + 1}`}
                                                </span>
                                                {mic.deviceId === selectedMicId && <Check size={13}/>}
                                            </button>
                                        ))}
                                    </div>
                                </>
                            )}
                        </div>
                    )}

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
                {/* Auch bei deaktivierter Live-Transkription (ORB-Modus) muss das
                    fertige Transkript nach dem Stop sichtbar sein — der ORB ist nur
                    für laufende Aufnahmen gedacht. */}
                {liveEnabled || (stableTranscript && !isRecording) ? (
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
                            onClick={isRecording ? handleStopClick : () => startRecording()}
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
                    {insertTarget?.process && (
                        <div
                            className={`recorder-target${insertTarget.list_active && !insertTarget.allowed ? ' warn' : ''}`}
                            title={insertTarget.title ? `Einfügen in: ${insertTarget.process} – ${insertTarget.title}` : `Einfügen in: ${insertTarget.process}`}
                        >
                            <strong>{insertTarget.process}</strong>
                        </div>
                    )}
                    <div className="recorder-action-icons">
                        <button
                            onClick={() => acceptAndInsert(false)}
                            className="recorder-icon-btn primary"
                            title={`Akzeptieren und einfügen (${modKeyLabel}+V)`}
                            aria-label="Akzeptieren und einfügen"
                        >
                            <Check size={18}/>
                        </button>
                        <button
                            onClick={() => startRecording()}
                            className="recorder-icon-btn"
                            title={`Neue Aufnahme starten (${modKeyLabel}+R)`}
                            aria-label="Neue Aufnahme"
                        >
                            <Mic size={18}/>
                        </button>
                        <button
                            onClick={() => GoOpenDashboard()}
                            className="recorder-icon-btn"
                            title="In openDox öffnen"
                            aria-label="In openDox öffnen"
                        >
                            <ExternalLink size={18}/>
                        </button>
                    </div>
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

            {helpOpen && (
                <div className="recorder-help-overlay" onClick={() => setHelpOpen(false)}>
                    <div className="recorder-help" onClick={(e) => e.stopPropagation()}>
                        <div className="recorder-help-head">
                            <span>Tastenkürzel</span>
                            <button
                                className="recorder-logout-btn"
                                onClick={() => setHelpOpen(false)}
                                aria-label="Schließen"
                            >
                                <X size={14}/>
                            </button>
                        </div>
                        <ul>
                            <li><kbd>{modKeyLabel}+R</kbd> Aufnahme starten / stoppen</li>
                            <li><kbd>{modKeyLabel}+P</kbd> Pausieren / Fortsetzen</li>
                            <li><kbd>{modKeyLabel}+V</kbd> Akzeptieren und einfügen</li>
                            <li><kbd>{modKeyLabel}+D</kbd> Halten zum Diktieren</li>
                            <li><kbd>{modKeyLabel}+H</kbd> Overlay ein- / ausblenden</li>
                        </ul>
                    </div>
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

            <ConfirmModal
                open={pendingInsert !== null}
                title="Einfügen bestätigen"
                description={`„${pendingInsert?.target.process || 'Unbekanntes Fenster'}“ ist auf diesem Arbeitsplatz nicht freigegeben. Trotzdem einfügen?`}
                confirmLabel="Trotzdem einfügen"
                cancelLabel="Abbrechen"
                onConfirm={confirmPendingInsert}
                onCancel={() => setPendingInsert(null)}
            />
        </div>
    );
});

export default Recorder;
