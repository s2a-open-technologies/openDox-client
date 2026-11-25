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

import {useState, useRef, useEffect, forwardRef, useImperativeHandle, useCallback} from "react";
import {Mic, Square, Loader2, Pause, Play, X, ExternalLink, LogOut, Check, Eye, EyeOff, HelpCircle, Settings as SettingsIcon} from "lucide-react";
import {resetPreferences, savePreferences, usePreferences} from "@/lib/preferences";
import {toast} from "sonner";
import {useTranslation} from "react-i18next";
import i18n, {goMessage} from "@/i18n";
import {UiLanguageSelect} from "@/components/ui/UiLanguageSelect";
import {CreateRecord as GoCreateRecord, GetWSURL as GoGetWSURL, GetWhisperSettings as GoGetWhisperSettings, GetCategories as GoGetCategories, GetTranslationAvailability as GoGetTranslationAvailability} from "../../wailsjs/go/main/App";
import {InsertTranscript as GoInsertTranscript, SetOverlayNoActivate as GoSetOverlayNoActivate} from "../../wailsjs/go/main/App";
import {GetInsertTarget as GoGetInsertTarget} from "../../wailsjs/go/main/App";
import type {main} from "../../wailsjs/go/models";
import {GetTaskStatus as GoGetTaskStatus, GetRecordBySession as GoGetRecordBySession} from "../../wailsjs/go/main/App";
import {ReprocessWithCategory as GoReprocessWithCategory} from "../../wailsjs/go/main/App";
import {OpenDashboard as GoOpenDashboard, CancelSession as GoCancelSession} from "../../wailsjs/go/main/App";
import {CheckAccessibility as GoCheckAccessibility, CheckHotkeys as GoCheckHotkeys} from "../../wailsjs/go/main/App";
import {GetUpdateInfo as GoGetUpdateInfo} from "../../wailsjs/go/main/App";
import {BrowserOpenURL, EventsOn, Environment} from "../../wailsjs/runtime/runtime";
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
const SERVER_ERROR_KEYS: Record<string, string> = {
    recording_audio_unavailable: "toast.serverAudioElsewhere",
};

// Welches Eingabegerät gilt ohne eigene Wahl: der Eintrag "default", sonst das erste echte Gerät
// (Monitor-Quellen, also mitgehörte Ausgaben, überspringen). "" = noch nicht bestimmbar.
function chooseDefaultMic(mics: MediaDeviceInfo[]): string {
    const real = mics.filter((m) => m.deviceId && !/monitor/i.test(m.label));
    return (real.find((m) => m.deviceId === "default") ?? real[0])?.deviceId ?? "";
}

const Recorder = forwardRef<RecorderHandle, RecorderProps>(({userId, title, categoryId, onFinalTranscription, onRecordingStart, autoStart, resumeSessionId, onLogout}, ref) => {
    const {t} = useTranslation();
    const {getFreshToken} = useAuth();
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
    // zusätzlich der Nutzer (Umschalter im Overlay, preferences.json) mit settings.json bzw.
    // Server-Richtlinie als Vorgabe. liveEnabled ist das Ergebnis aus beidem — bei false zeigt
    // das Overlay nur den Orb und der Server startet keine Live-ASR (live=0 am WebSocket).
    const [serverLive, setServerLive] = useState(true);
    const prefs = usePreferences();
    const liveView = prefs.live_view;
    // Übersetzung (Diktat in Fremdsprache): nur wenn der Server sie für diesen Nutzer freigibt
    // (/settings/translation/me). Die gewählte Sprache zählt nur, solange sie dort noch angeboten wird.
    const [translation, setTranslation] = useState<{languages: string[]; live: string[]; target: string}>({languages: [], live: [], target: ""});
    useEffect(() => {
        let cancelled = false;
        getFreshToken()
            .then((t) => GoGetTranslationAvailability(t))
            .then((a) => {
                if (!cancelled && a.available) setTranslation({languages: a.languages ?? [], live: a.live_languages ?? [], target: a.target_language ?? ""});
            })
            .catch(() => { /* nicht verfügbar */ });
        return () => { cancelled = true; };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [userId]);
    const sourceLanguage = translation.languages.includes(prefs.dictation_language) ? prefs.dictation_language : "";
    const sourceLanguageRef = useRef("");
    sourceLanguageRef.current = sourceLanguage;
    // Sprachen ohne Live-Modell: der Server liefert dort keinen Live-Text, also nur den Orb zeigen.
    const liveEnabled = serverLive && liveView && !(sourceLanguage && !translation.live.includes(sourceLanguage));
    const languageName = (code: string) => {
        try {return new Intl.DisplayNames([i18n.language], {type: "language"}).of(code) ?? code;} catch (_) {return code;}
    };
    // "processing" zeigt den Server-Schritt (Transkription, Übersetzung, ...) im Overlay.
    const [processingStep, setProcessingStep] = useState("");
    // Zahlen, die in der Übersetzung fehlen oder dazugekommen sind: dann kein Auto-Insert.
    const [numberWarning, setNumberWarning] = useState<string[]>([]);
    const [prefsOpen, setPrefsOpen] = useState(false);
    // Neuere Client-Version laut Server (nur Info; installiert wird über die IT-Verteilung).
    const [updateLatest, setUpdateLatest] = useState<string | null>(null);
    useEffect(() => {
        if (!prefsOpen) return;
        GoGetUpdateInfo()
            .then((info) => setUpdateLatest(info.available ? info.latest : null))
            .catch(() => setUpdateLatest(null));
    }, [prefsOpen]);
    const liveEnabledRef = useRef(true);
    liveEnabledRef.current = liveEnabled;
    const toggleLiveView = () => {
        void savePreferences({live_view: !liveView});
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
        const record = await GoCreateRecord(parseInt(userId), title || "Client-Aufnahme", selectedCategoryId, sourceLanguageRef.current, await getFreshToken());

        setSessionId(record.session_id);
        sessionIdRef.current = record.session_id;
        toast.success(i18n.t("toast.started"));
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
            status = await GoGetTaskStatus(taskId, await getFreshToken());
        } catch (err) {
            console.error("Task-Status-Abfrage fehlgeschlagen:", err);
            pollTimeoutRef.current = setTimeout(() => pollTaskStatus(taskId, pollSessionId), 2000);
            return;
        }
        if (sessionIdRef.current !== pollSessionId) return;

        if (status.status === "finished") {
            try {
                const record = await GoGetRecordBySession(pollSessionId, await getFreshToken());
                if (sessionIdRef.current !== pollSessionId) return;
                setStableTranscript(record.summarize || "");
                setLiveTranscript("");
                setRecordId(record.id);
                // Übersetzung: weicht eine Zahl zwischen Original und Übersetzung ab, wird nicht
                // automatisch eingefügt, sondern erst nach manuellem Akzeptieren.
                const deviating = new Set<string>();
                for (const seg of record.translation_segments ?? []) {
                    for (const n of [...(seg.missing_numbers ?? []), ...(seg.extra_numbers ?? [])]) deviating.add(n);
                }
                setNumberWarning([...deviating]);
                const autoInsert = autoInsertRef.current && deviating.size === 0;
                if (deviating.size > 0) {
                    toast.warning(i18n.t("toast.numbersDeviate"), {duration: 6000});
                }
                autoInsertRef.current = false;
                if (autoInsert && record.summarize) {
                    acceptAndInsertRef.current(true, record.summarize);
                } else {
                    toast.success(i18n.t("toast.done", {mod: modKeyLabel}));
                }
                if (onFinalTranscription) {
                    onFinalTranscription(pollSessionId, taskId);
                }
            } catch (err) {
                console.error(err);
                autoInsertRef.current = false;
                toast.error(i18n.t("toast.fetchFailed"), {duration: 4000});
            } finally {
                setIsProcessing(false);
                setProcessingStep("");
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
            toast.error(status.error || i18n.t("toast.processingFailed"), {duration: 4000});
            return;
        }

        // pending / running / transcribing / translating / processing_llm -> weiter pollen
        setProcessingStep(status.status === "translating" ? i18n.t("status.translating") : "");
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
            toast.error(i18n.t("toast.reprocessTimeout"), {duration: 4000});
        }, 90_000);

        try {
            const record = await GoReprocessWithCategory(recordId, newCategoryId, await getFreshToken());
            if (sessionIdRef.current !== pollSessionId) return;
            setStableTranscript(record.summarize || "");
            toast.success(i18n.t("toast.reprocessDone", {mod: modKeyLabel}));
        } catch (err) {
            console.error("Neuverarbeitung fehlgeschlagen:", err);
            toast.error(i18n.t("toast.reprocessFailed"), {duration: 4000});
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
            wsURL = await GoGetWSURL(session_id, userId, await getFreshToken(), liveEnabledRef.current);
        } catch {
            wsURL = `${wsBaseURL}/api/v1/records/ws/record?session_id=${encodeURIComponent(session_id)}&user_id=${encodeURIComponent(userId)}&token=${encodeURIComponent(await getFreshToken())}${liveEnabledRef.current ? "" : "&live=0"}`;
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
                toast.error(SERVER_ERROR_KEYS[data.code] ? i18n.t(SERVER_ERROR_KEYS[data.code]) : data.error, {duration: 4000});
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
                toast.success(i18n.t("toast.processingStarted"));
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
                toast.error(data.message || i18n.t("toast.recordingError"), {duration: 4000});
            }
        };

        ws.onerror = (error) => {
            console.error("WebSocket Fehler:", error);
            toast.error(i18n.t("toast.wsError"), {duration: 4000});
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
                    toast.error(i18n.t("toast.connectionLost"), {duration: 4000});
                }
            }
            wsFinishedRef.current = false;
            serverErrorShownRef.current = false;
        };
    };

    // Ohne eigene Wahl ("Systemstandard") ein konkretes Eingabegerät bestimmen. WebKitGTK und
    // WKWebView liefern bei getUserMedia ohne deviceId teils ein Gerät ohne Signal (etwa eine
    // Monitor-Quelle) — die Aufnahme läuft, aber es kommt kein Ton. Mit explizitem deviceId
    // (wie bei einer Auswahl im Menü) funktioniert es. Vor der ersten Freigabe liefert
    // enumerateDevices nur Platzhalter ohne deviceId/Label, daher einmal kurz freigeben lassen.
    const pickDefaultMic = async (): Promise<string> => {
        const md = navigator.mediaDevices;
        if (!md) return "";
        try {
            let mics = (await md.enumerateDevices()).filter((d) => d.kind === "audioinput");
            if (mics.length === 0 || mics.some((m) => !m.deviceId || !m.label)) {
                const probe = await md.getUserMedia({audio: true});
                probe.getTracks().forEach((tr) => tr.stop());
                mics = (await md.enumerateDevices()).filter((d) => d.kind === "audioinput");
            }
            const pick = chooseDefaultMic(mics);
            if (pick) console.info("Mikrofon (Systemstandard) aufgelöst:", mics.find((m) => m.deviceId === pick)?.label || pick);
            return pick;
        } catch (err) {
            console.warn("Standard-Mikrofon nicht bestimmbar, nutze Systemvorgabe:", err);
            return "";
        }
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
                setStableTranscript(""); setNumberWarning([]);
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
            const micId = selectedMicIdRef.current || await pickDefaultMic();
            const audioConstraints = {
                channelCount: 1,
                sampleRate: 16000,
                echoCancellation: true,
                noiseSuppression: true,
            };
            let stream: MediaStream;
            try {
                stream = await navigator.mediaDevices.getUserMedia({
                    audio: micId ? {...audioConstraints, deviceId: {exact: micId}} : audioConstraints,
                });
            } catch (err: any) {
                // Gerät inzwischen weg oder ID nicht mehr gültig: mit der Systemvorgabe weiter.
                if (!micId || err?.name !== "OverconstrainedError") throw err;
                stream = await navigator.mediaDevices.getUserMedia({audio: audioConstraints});
            }
            streamRef.current = stream;
            // Mit erteilter Freigabe kennt der Browser jetzt alle Geräte samt Labels.
            refreshMics();

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
                ? i18n.t("toast.micDenied")
                : err?.name === "NotFoundError"
                    ? i18n.t("toast.noMic")
                    : i18n.t("toast.startFailed");
            toast.error(msg, {duration: 4000});
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
                try {await GoCancelSession(parseInt(userId), await getFreshToken());} catch (_) {
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
                toast.error(i18n.t("toast.processTimeout"), {duration: 4000});
            }, 90_000);
        } else {
            autoInsertRef.current = false;
            setIsProcessing(false);
            toast.error(i18n.t("toast.alreadyDisconnected"), {duration: 4000});
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
        setStableTranscript(""); setNumberWarning([]);
        setLiveTranscript("");
        setSessionId(null);
        sessionIdRef.current = null;
        setRecordId(null);

        try {
            await GoCancelSession(parseInt(userId), await getFreshToken());
            toast.success(i18n.t("toast.cancelled"));
        } catch (err) {
            console.error(err);
            toast.error(i18n.t("toast.cancelFailed"), {duration: 4000});
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
            toast.error(goMessage(err), {duration: 5000});
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
                    i18n.t("toast.macPermAll"),
                    {duration: 8000}
                );
            } else if (!axOk) {
                toast.error(
                    i18n.t("toast.macPermAx"),
                    {duration: 8000}
                );
            } else if (!hkOk) {
                toast.error(
                    i18n.t("toast.macPermInput"),
                    {duration: 8000}
                );
            }
        });
    }, []);

    useEffect(() => {
        // Load settings to determine if live transcription is enabled
        GoGetWhisperSettings()
            .then((data) => {
                setServerLive(data.live_backend !== "disabled");
            })
            .catch(() => setServerLive(true));

        // Kategorien vom Server laden
        getFreshToken()
            .then(GoGetCategories)
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

    // Mikrofonliste laden + auf Hot-Plug (USB-Mikro an/ab) reagieren.
    // Vor der ersten getUserMedia-Berechtigung liefern Browser (WebKitGTK, WKWebView
    // und WebView2 gleichermaßen) laut Spezifikation nur einen Platzhalter ohne
    // Label/deviceId — daher fragt openMicMenu die Freigabe bei Bedarf selbst an.
    const refreshMics = useCallback(() => {
        return navigator.mediaDevices?.enumerateDevices()
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
    }, []);

    useEffect(() => {
        refreshMics();
        const md = navigator.mediaDevices;
        if (!md) return;
        md.addEventListener("devicechange", refreshMics);
        return () => md.removeEventListener("devicechange", refreshMics);
    }, [refreshMics]);

    // Ohne Freigabe ist unbekannt, wie viele Mikrofone es gibt — dann die Auswahl
    // trotzdem anbieten, statt sie erst nach der ersten Aufnahme einzublenden.
    const micsUnknown = micDevices.some((d) => !d.label);
    // Ohne eigene Wahl ist das Gerät markiert, das beim Start tatsächlich benutzt wird.
    const effectiveMicId = selectedMicId || chooseDefaultMic(micDevices);

    const openMicMenu = async () => {
        if (micMenuOpen) {
            setMicMenuOpen(false);
            return;
        }
        if (micsUnknown) {
            try {
                const probe = await navigator.mediaDevices.getUserMedia({audio: true});
                probe.getTracks().forEach((t) => t.stop());
            } catch (err) {
                console.warn("Mikrofon-Freigabe für Geräteliste fehlgeschlagen:", err);
            }
            await refreshMics();
        }
        setMicMenuOpen(true);
    };

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
                    title={isRecording ? undefined : (isBackendReachable ? t("status.serverReachable") : t("status.serverUnreachable"))}
                />
                <span className="recorder-status-text">
                    {isPaused ? t("status.paused") : isRecording ? t("status.recording") : (isBackendReachable ? t("status.ready") : t("status.unreachable"))}
                </span>

                {/* Rechts gruppierte Aktionen: Mikrofon-Wahl + Kategorie-Dropdown +
                    Logout, getrennt voneinander und vom global fixierten
                    Close-Button (App.tsx). */}
                <div className="recorder-status-actions">
                    {serverLive && !isRecording && !isProcessing && (
                        <button
                            className="recorder-logout-btn"
                            onClick={toggleLiveView}
                            title={liveView ? t("toolbar.liveOff") : t("toolbar.liveOn")}
                            aria-label={liveView ? t("toolbar.liveOff") : t("toolbar.liveOn")}
                            aria-pressed={liveView}
                        >
                            {liveView ? <Eye size={14}/> : <EyeOff size={14}/>}
                        </button>
                    )}
                    {!isRecording && !isProcessing && (
                        <button
                            className="recorder-logout-btn"
                            onClick={() => setPrefsOpen(true)}
                            title={t("toolbar.settings")}
                            aria-label={t("toolbar.settings")}
                        >
                            <SettingsIcon size={14}/>
                        </button>
                    )}
                    <button
                        className="recorder-logout-btn"
                        onClick={() => setHelpOpen(true)}
                        title={t("toolbar.shortcuts")}
                        aria-label={t("toolbar.shortcutsShow")}
                    >
                        <HelpCircle size={14}/>
                    </button>
                    {!isRecording && !isProcessing && (micDevices.length > 1 || micsUnknown || micMenuOpen) && (
                        <div className="recorder-mic-wrap">
                            <button
                                className={`recorder-mic-select-btn ${micMenuOpen ? "open" : ""}`}
                                onClick={openMicMenu}
                                title={t("toolbar.micSelect")}
                                aria-label={t("toolbar.micSelect")}
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
                                        <div className="recorder-mic-menu-title">{t("toolbar.mic")}</div>
                                        {micDevices.map((mic, idx) => (
                                            <button
                                                key={mic.deviceId || idx}
                                                role="menuitemradio"
                                                aria-checked={mic.deviceId === effectiveMicId}
                                                className={`recorder-mic-item ${mic.deviceId === effectiveMicId ? "active" : ""}`}
                                                onClick={() => {
                                                    setSelectedMicId(mic.deviceId);
                                                    setMicMenuOpen(false);
                                                }}
                                            >
                                                <span className="recorder-mic-item-label">
                                                    {mic.label || t("toolbar.micNumber", {n: idx + 1})}
                                                </span>
                                                {mic.deviceId === effectiveMicId && <Check size={13}/>}
                                            </button>
                                        ))}
                                    </div>
                                </>
                            )}
                        </div>
                    )}

                    {sourceLanguage && (
                        <div className="recorder-target" title={t("toolbar.dictationLanguageHint")}>
                            <strong>{languageName(sourceLanguage)} → {languageName(translation.target || "de")}</strong>
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
                            title={t("toolbar.category")}
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
                            title={t("common.logout")}
                            aria-label={t("common.logout")}
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
                                <p>{t("transcript.placeholder")}</p>
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

            {/* Footer / Controls — bei fertigem Text ausgeblendet: dann startet der
                Mikrofon-Button in den Aktions-Buttons die neue Aufnahme. */}
            {!insertVisible && (
                <div className="recorder-controls">
                    {!isProcessing ? (
                        <>
                            {isRecording && (
                                <button
                                    onClick={handleCancelClick}
                                    className="recorder-cancel-btn"
                                    aria-label={t("controls.cancel")}
                                    title={t("controls.cancel")}
                                >
                                    <X className="icon-fill"/>
                                </button>
                            )}

                            {isRecording && (
                                <button
                                    onClick={isPaused ? resumeSession : pauseSession}
                                    className={`recorder-pause-btn ${isPaused ? "resuming" : "pausing"}`}
                                    aria-label={isPaused ? t("controls.resume") : t("controls.pause")}
                                >
                                    {isPaused ? <Play className="icon-fill"/> : <Pause className="icon-fill"/>}
                                </button>
                            )}

                            <button
                                onClick={isRecording ? handleStopClick : () => startRecording()}
                                className={`recorder-mic-btn ${isRecording ? "recording" : ""}`}
                                aria-label={isRecording ? t("controls.stop") : t("controls.start")}
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
                            <span>{processingStep || t("status.processing")}</span>
                        </div>
                    )}
                </div>
            )}

            {/* Aktions-Buttons statt Aufnahme-Button (nur wenn fertig transkribiert) */}
            {insertVisible && (
                <div className="recorder-actions">
                    {insertTarget?.process && (
                        <div
                            className={`recorder-target${insertTarget.list_active && !insertTarget.allowed ? ' warn' : ''}`}
                            title={insertTarget.title ? t("actions.insertInTitle", {process: insertTarget.process, title: insertTarget.title}) : t("actions.insertIn", {process: insertTarget.process})}
                        >
                            <strong>{insertTarget.process}</strong>
                        </div>
                    )}
                    {numberWarning.length > 0 && (
                        <div className="recorder-target warn" role="alert">
                            {t("actions.numbersCheck")} <strong>{numberWarning.join(", ")}</strong>
                        </div>
                    )}
                    <div className="recorder-action-icons">
                        <button
                            onClick={() => acceptAndInsert(false)}
                            className="recorder-icon-btn primary"
                            title={t("actions.acceptTitle", {mod: modKeyLabel})}
                            aria-label={t("actions.accept")}
                        >
                            <Check size={18}/>
                        </button>
                        <button
                            onClick={() => startRecording()}
                            className="recorder-icon-btn"
                            title={t("actions.newRecordingTitle", {mod: modKeyLabel})}
                            aria-label={t("actions.newRecording")}
                        >
                            <Mic size={18}/>
                        </button>
                        <button
                            onClick={() => GoOpenDashboard()}
                            className="recorder-icon-btn"
                            title={t("actions.openInOpenDox")}
                            aria-label={t("actions.openInOpenDox")}
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
                            <span>{t("shortcuts.title")}</span>
                            <button
                                className="recorder-logout-btn"
                                onClick={() => setHelpOpen(false)}
                                aria-label={t("common.close")}
                            >
                                <X size={14}/>
                            </button>
                        </div>
                        <ul>
                            <li><kbd>{modKeyLabel}+R</kbd> {t("shortcuts.record")}</li>
                            <li><kbd>{modKeyLabel}+P</kbd> {t("shortcuts.pause")}</li>
                            <li><kbd>{modKeyLabel}+V</kbd> {t("shortcuts.insert")}</li>
                            <li><kbd>{modKeyLabel}+D</kbd> {t("shortcuts.hold")}</li>
                            <li><kbd>{modKeyLabel}+H</kbd> {t("shortcuts.hide")}</li>
                        </ul>
                    </div>
                </div>
            )}

            {prefsOpen && (
                <div className="recorder-help-overlay" onClick={() => setPrefsOpen(false)}>
                    <div className="recorder-help" onClick={(e) => e.stopPropagation()}>
                        <div className="recorder-help-head">
                            <span>{t("settings.title")}</span>
                            <button
                                className="recorder-logout-btn"
                                onClick={() => setPrefsOpen(false)}
                                aria-label={t("common.close")}
                            >
                                <X size={14}/>
                            </button>
                        </div>
                        <div className="recorder-prefs">
                            <PrefRow
                                label={t("settings.theme")}
                                value={prefs.theme}
                                options={[["dark", t("settings.themeDark")], ["light", t("settings.themeLight")]]}
                                onChange={(theme) => savePreferences({theme})}
                            />
                            {serverLive && (
                                <PrefRow
                                    label={t("settings.liveView")}
                                    value={prefs.live_view ? "on" : "off"}
                                    options={[["on", t("settings.on")], ["off", t("settings.off")]]}
                                    onChange={(v) => savePreferences({live_view: v === "on"})}
                                />
                            )}
                            <div className="recorder-prefs-row">
                                <span>{t("settings.uiLanguage")}</span>
                                <UiLanguageSelect className="recorder-category-select"/>
                            </div>
                            {translation.languages.length > 0 && (
                                <div className="recorder-prefs-row">
                                    <span>{t("settings.dictationLanguage")}</span>
                                    <select
                                        className="recorder-category-select"
                                        value={sourceLanguage}
                                        onChange={(e) => savePreferences({dictation_language: e.target.value})}
                                        aria-label={t("settings.dictationLanguage")}
                                    >
                                        <option value="">{t("settings.noTranslation", {language: languageName(translation.target || "de")})}</option>
                                        {translation.languages.map((l) => (
                                            <option key={l} value={l}>{`${languageName(l)} → ${languageName(translation.target || "de")}`}</option>
                                        ))}
                                    </select>
                                </div>
                            )}
                            <PrefRow
                                label={t("settings.micButton")}
                                value={prefs.dictation_mic_mode}
                                options={[["hold", t("settings.micHold")], ["toggle", t("settings.micToggle")]]}
                                onChange={(dictation_mic_mode) => savePreferences({dictation_mic_mode})}
                            />
                            <button className="recorder-prefs-reset" onClick={() => resetPreferences()}>
                                {t("settings.reset")}
                            </button>
                            {/* AGPL: Hinweis auf Lizenz und Quellcode. Die Lizenztexte aller
                                Komponenten liegen auch neben dem Programm (THIRD_PARTY_LICENSES.txt). */}
                            <div className="recorder-prefs-about">
                                openDox Client {__APP_VERSION__} · AGPL-3.0 ·{" "}
                                <button onClick={() => BrowserOpenURL(SOURCE_URL)}>
                                    {t("settings.source")}
                                </button>
                                {updateLatest && (
                                    <div className="recorder-prefs-update">
                                        {t("settings.updateAvailable", {version: updateLatest})}
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>
                </div>
            )}

            <ConfirmModal
                open={isStopDialogOpen}
                title={t("dialog.stopTitle")}
                description={t("dialog.stopDescription")}
                confirmLabel={t("dialog.stopTitle")}
                onConfirm={confirmStop}
                onCancel={() => setIsStopDialogOpen(false)}
            />

            <ConfirmModal
                open={isCancelDialogOpen}
                title={t("dialog.cancelTitle")}
                description={t("dialog.cancelDescription")}
                confirmLabel={t("dialog.cancelTitle")}
                cancelLabel={t("common.back")}
                onConfirm={confirmCancel}
                onCancel={() => setIsCancelDialogOpen(false)}
            />

            <ConfirmModal
                open={pendingInsert !== null}
                title={t("dialog.insertTitle")}
                description={t("dialog.insertDescription", {process: pendingInsert?.target.process || t("dialog.unknownWindow")})}
                confirmLabel={t("dialog.insertAnyway")}
                onConfirm={confirmPendingInsert}
                onCancel={() => setPendingInsert(null)}
            />
        </div>
    );
});

export default Recorder;

// Quellcode des Clients (AGPL), dort liegen auch LICENSE und THIRD_PARTY_LICENSES.txt.
const SOURCE_URL = "https://github.com/s2a-open-technologies/openDox-client";

// Eine Zeile im Einstellungen-Dialog: Beschriftung + Segment-Umschalter.
function PrefRow<T extends string>({label, value, options, onChange}: {
    label: string;
    value: T;
    options: [T, string][];
    onChange: (v: T) => void;
}) {
    return (
        <div className="recorder-prefs-row">
            <span>{label}</span>
            <div className="recorder-seg" role="radiogroup" aria-label={label}>
                {options.map(([v, text]) => (
                    <button
                        key={v}
                        role="radio"
                        aria-checked={value === v}
                        className={value === v ? "active" : ""}
                        onClick={() => value !== v && onChange(v)}
                    >
                        {text}
                    </button>
                ))}
            </div>
        </div>
    );
}
