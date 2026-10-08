# Zweckbestimmung openDox Client

> Stand: 2026-10-08 · gilt für openDox Client ab Version 0.8.0 (inkl. `[Unreleased]`)
> Legt fest, wofür das Produkt bestimmt ist und wofür nicht; Grundlage der Abgrenzung zum
> Medizinprodukt (Art. 2 Nr. 1 MDR). Keine Rechtsberatung — die Einordnung sollte vor einem
> Einsatz in Praxen/Kliniken von Jurist:innen bzw. Regulatory-Berater:innen geprüft werden.

## 1. Bestimmungsgemäßer Zweck

Der openDox Client ist eine Arbeitsplatz-Oberfläche zur **Unterstützung bei der
Dokumentation**: Er nimmt gesprochene Diktate auf, zeigt den vom openDox-Server erstellten
Text an und fügt ihn auf Anweisung der Nutzerin oder des Nutzers in das gerade aktive
Programm ein. Die inhaltliche Verantwortung für jeden übernommenen Text liegt bei der
anwendenden Fachperson.

## 2. Anwender und Einsatzumfeld

- **Anwender:** Ärztinnen und Ärzte sowie anderes medizinisches Fachpersonal, das
  Dokumentation diktiert. Der Client ist ein allgemeines Diktierwerkzeug und nicht auf
  Gesundheitsdaten beschränkt (z. B. auch private E-Mails; siehe `AGENTS.md`, Abschnitt
  „Clipboard hardening“).
- **Umfeld:** Arbeitsplatz-Rechner (Windows, macOS, Linux) in Praxis, Klinik oder Büro, mit
  Netzzugang zu einem **vom Betreiber selbst betriebenen openDox-Server**.
- **Nicht vorgesehen:** Patientinnen und Patienten oder sonstige Laien als Anwender.

## 3. Funktionsumfang im Rahmen des Zwecks

| Funktion | Beschreibung |
|---|---|
| Aufnahme und Live-Anzeige | Audio wird per WebSocket an den Server gesendet, Text erscheint während des Sprechens (abschaltbar) |
| Akzeptieren und einfügen | Fügt den fertigen Text in das aktive Programm ein (Button, `Strg+Alt+V` oder Diktiermikro-Taste) |
| Globale Hotkeys, Diktiermikro | Bedienung ohne Fokuswechsel (Philips SpeechMike, Nuance PowerMic) |
| Kategorien | Wählt die Dokumentationsvorlage des Servers |
| Anmeldung | Passwort oder SSO (OIDC) gegen den Server |
| Einfüge-Schutz | Zwischenablage-Schutz (Windows), Anzeige des Einfügeziels, optionale Allow-Liste |
| Client-Richtlinie | Zentrale Einstellungen vom Server (`policy_sync.go`) |

Transkription und Zusammenfassung finden **auf dem Server** statt, nicht im Client.

## 4. Ausdrücklich nicht bestimmungsgemäß

- Diagnose, Therapie, Prognose oder Therapieempfehlung
- Entscheidungsfindung, Triage, Dosierung
- Ungeprüfte Übernahme maschinell erstellter Texte in Patientenakten oder
  Weitergabe an Dritte
- Ableitung neuer medizinischer Aussagen aus den Transkripten
- Alleinige Entscheidungsgrundlage für medizinische Maßnahmen
- Einsatz in Notfall- und Intensivsituationen

## 5. Pflichten bei der Nutzung

- **Human-in-the-loop:** Jeder erzeugte Text muss von qualifiziertem Fachpersonal geprüft
  werden, bevor er verwendet wird. Bekannte Fehlerquellen: falsch erkannte, ausgelassene
  oder hinzugefügte Inhalte, falsche Zuordnung von Aussagen.
- **Betreiberverantwortung:** DSGVO (Art. 9, 28, 32; Verzeichnis nach Art. 30), § 203 StGB,
  IT-Sicherheit und ggf. Mitbestimmungsrechte liegen beim Betreiber. Der Datenfluss ist in
  [`data-mapping.md`](data-mapping.md) beschrieben.

### Technische Durchsetzung

| Maßnahme | Fundstelle |
|---|---|
| Pflicht-Bestätigung des Haftungsausschlusses beim ersten Login, serverseitig protokolliert | `frontend/src/components/DisclaimerGate.tsx`, `/auth/accept-disclaimer` |
| Text kopiert sich nach der Verarbeitung nicht automatisch; Weg ins Zielprogramm nur über den Einfüge-Schritt | `acceptAndInsert` in `frontend/src/components/Recorder.tsx`, `InsertTranscript` in `app.go` |
| Anzeige des Einfügeziels („Einfügen in: …“) und optionale Allow-Liste | `insert_target.go`, `Recorder.tsx` |
| Transkript im Overlay nicht markierbar | `Recorder.tsx` (`select-none`) |

### Bekannte Abweichung: Hold-Modus des Diktiermikros

Im **Hold-Modus** (Standard für Diktiermikros, `dictation_mic_mode`) wird nach dem
Loslassen der Aufnahmetaste ohne weiteren Bestätigungsklick eingefügt
(`startRecording({autoInsert: true})` → `acceptAndInsert(true, …)`, `Recorder.tsx`). Das
Loslassen ist hier die einzige Handlung der Nutzerin oder des Nutzers, die Prüfung findet
erst **nach** dem Einfügen im Zielprogramm statt. Das ist eine bewusste
Komfortentscheidung, schwächt aber die Aussage „Einfügen nur nach explizitem
Bestätigungsschritt“ (vgl. [`../DISCLAIMER.md`](../DISCLAIMER.md)). Betreiber, die eine
Prüfung **vor** dem Einfügen sicherstellen müssen, setzen `dictation_mic_mode` serverseitig
auf `"toggle"` bzw. nutzen die Hotkeys/Buttons (Anzeige, dann `Strg+Alt+V`).
Die Tastenbelegung der Mikros ist zudem noch nicht an echter Hardware verifiziert.

## 6. Änderung des Zwecks

Wer den Client abweichend einsetzt (z. B. diagnostische Vorlagen auf dem Server,
ungeprüfte Weiterverarbeitung der Texte), verändert die Zweckbestimmung und kann damit nach
Art. 16 MDR selbst zum Hersteller werden. Das ist vorab zu prüfen.

## 7. Hersteller und Kontakt

Open-Source-Projekt (AGPL-3.0-or-later), kein kommerzieller Vertrieb durch dieses Repository.
Primary Developer: Christopher Abanilla — Kontakt siehe [`../AUTHORS`](../AUTHORS) und
[`../.github/SECURITY.md`](../.github/SECURITY.md). Die Zweckbestimmung gilt gemeinsam für
den openDox-Server; dessen Repository enthält ein eigenes Dokument.

## Verwandte Dokumente

- [`../DISCLAIMER.md`](../DISCLAIMER.md) — Haftungsausschluss (Langfassung)
- [`opendox-client.md`](opendox-client.md) — Überblick
- [`data-mapping.md`](data-mapping.md) — Datenfluss
- [`audit-checklist-opendox-client.md`](audit-checklist-opendox-client.md) — Selbstauskunft
