# openDox unter Windows verteilen (SCCM/Intune/GPO)

Die MSI (`opendox-client-installer.msi`, gebaut über `../../../scripts/build-windows-msi.sh`)
installiert die App maschinenweit nach `%ProgramFiles%\openDox\` und legt dort
eine `settings.json` mit der zum Build-Zeitpunkt eingebetteten Server-URL ab
(`settings.json.template` in diesem Verzeichnis). Für einen einzelnen Rechner reicht
das — ein Admin öffnet die Datei danach einmal im Texteditor und trägt die richtige
URL ein.

Die MSI legt `settings.json` **nur an, wenn sie noch fehlt**: Ein Update überschreibt eine
vorhandene Datei nie (Component-Attribut NeverOverwrite, siehe `opendox-client.wxs`);
eine Deinstallation entfernt sie. Eine einmal abgelegte Datei
übersteht also Updates; zum Ändern der URL einfach die Datei selbst überschreiben.

**Einfachster Weg:** Im openDox-Server unter Admin → **Clients** gibt es die Installer (Links auf das
neueste GitHub-Release), eine fertige `settings.json` mit der Adresse dieses Servers und die Befehle
unten bereits ausgefüllt zum Kopieren. Die `settings.json` enthält nur `server_url` und ändert sich
danach nicht mehr — alle übrigen Einstellungen (Zwischenablage, Allow-Liste, Diktiermikro, Live-Ansicht)
stellt der Admin dort zentral ein, die Clients holen sie sich selbst (siehe unten).

Für eine Verteilung an viele Rechner mit fester/unterschiedlicher URL **nicht** die
MSI pro Kunde neu bauen. Stattdessen: **eine** MSI für alle, und die passende
`settings.json` als zweiten, unabhängigen Schritt eurer Softwareverteilung an

```
%ProgramFiles%\openDox\settings.json
```

kopieren. `loadSettings()` (`app.go`) liest diese Datei bei jedem App-Start neu — kein
Neustart der Verteilung, kein Rebuild nötig, einfach die Datei überschreiben.

> Bewusst **kein** MSI-Property (`msiexec /i ... SERVERURL=...`) und **keine**
> CustomAction, die die Datei beim Setup selbst schreibt: genau das wurde schon
> versucht (deferred CustomAction, die `opendox-client.exe` in einem eigenen
> Kommandozeilen-Hilfsmodus erneut aufruft) und ist auf echter Windows-Hardware
> wiederholt und ohne eindeutig identifizierbare Ursache gescheitert — siehe
> Kommentar in `opendox-client.wxs`. `wixl` (msitools) ist zudem ein sehr
> eingeschränktes WiX-Subset ohne Extensions wie `util:XmlFile`. Ein einfacher
> Dateikopiervorgang nach der Installation ist zuverlässiger und lässt sich ohne
> Windows-Testumgebung viel leichter nachvollziehen als MSI-Interna.

## Intune (Win32-App)

Die kundenspezifische `settings.json` mit in dasselbe `.intunewin`-Paket packen wie
die MSI, und als Installationsbefehl statt nur `msiexec` einen Einzeiler verwenden:

```
msiexec /i opendox-client-installer.msi /qn && copy /Y settings.json "%ProgramFiles%\openDox\settings.json"
```

Uninstall-Befehl unverändert `msiexec /x {ProductCode} /qn`.

## SCCM

Als Application mit einem skriptbasierten Deployment-Type (PowerShell/Batch), der
beide Schritte macht:

```powershell
Start-Process msiexec.exe -ArgumentList '/i opendox-client-installer.msi /qn' -Wait
Copy-Item .\settings.json 'C:\Program Files\openDox\settings.json' -Force
```

Alternativ als separates Configuration Item/Baseline, das nur die Datei durchsetzt —
praktisch, weil SCCM das dann laufend gegenprüfen und bei Abweichung reparieren kann.

## GPO

Über Group Policy **Preferences** → Dateien (nicht die klassische "Softwareinstallation
über GPO", die kann nur die MSI selbst, keine Zusatzdateien). Die Datei-Preference
kopiert `settings.json` an den Zielpfad, unabhängig von der MSI-Installation, pro
OU/Gruppe unterschiedlich konfigurierbar.

## Zwischenablage und Einfügeziel absichern

Der Client legt diktierten Text zum Einfügen (Strg+V) kurz in die Windows-Zwischenablage. Ohne
Gegenmaßnahmen könnte dieser Text im Zwischenablage-Verlauf (Win+V) oder in der Cloud-Zwischenablage
liegen bleiben — auf gemeinsam genutzten Stationsrechnern ein Datenschutzproblem.

**1. Windows-Richtlinien (die eigentliche Absicherung).** Verlauf und geräteübergreifende
Synchronisation per GPO/Intune abschalten (Namen wie in der englischen ADMX `OSPolicy.admx`, die
deutsche Oberfläche zeigt sie übersetzt):

| Zweck | GPO (Computerkonfiguration → Administrative Vorlagen → System → OS Policies) | Intune |
| --- | --- | --- |
| Zwischenablage-Verlauf (Win+V) aus | "Allow Clipboard History" = Disabled | Policy CSP `./Device/Vendor/MSFT/Policy/Config/Experience/AllowClipboardHistory` = `0` |
| Cloud-Synchronisierung aus | "Allow Clipboard synchronization across devices" = Disabled (Registrywert `AllowCrossDeviceClipboard` im selben Schlüssel `HKLM\Software\Policies\Microsoft\Windows\System`) | Einstellungskatalog → Administrative Templates → System → OS Policies |

> Die erste Zeile ist gegen die Microsoft-Dokumentation (Policy CSP "Experience") geprüft. Für die zweite
> Zeile habe ich in der Doku keinen eigenen CSP-Eintrag gefunden; Bezeichnung und Registrywert stammen aus
> dem Gedächtnis und sollten in der eigenen ADMX-Version bzw. im Intune-Einstellungskatalog gegengeprüft
> werden.

**2. Schutz im Client (zusätzlich, abhängig vom Betriebssystem).** Ab dieser Version markiert der Client
den Text als vertraulich (das Betriebssystem schließt ihn dann aus Verlauf und Cloud aus) und leert die
Zwischenablage nach dem Einfügen, sofern sie unverändert ist. Beides stellt der Admin im openDox-Server
unter Admin → Clients ein (Vorrang vor der `settings.json`, siehe `README.md` → "Zwischenablage und
Einfügeziel"): `clipboard_exclude_from_history` (Default `true`) und
`clipboard_clear_after_seconds` (Default `30`, `0` = nie). Die Richtlinien aus Punkt 1 bleiben trotzdem
sinnvoll, denn die Markierung wirkt nur, wenn das Betriebssystem bzw. ein Zwischenablage-Manager sie beachtet.

**3. Optional: Allow-Liste für das Einfügeziel.** Auf Arbeitsplätzen, die ausschließlich für die
Patientendokumentation gedacht sind, lässt sich per `insert_allowlist` festlegen, in welche Programme ohne
Rückfrage eingefügt werden darf (z. B. `orbis.exe`); bei anderen Zielen fragt der Client nach. Die Liste
wird im openDox-Server (Admin → Clients) gepflegt und gilt für alle Clients dieses Servers; Nutzer können
sie am Rechner nicht überschreiben. Ohne Liste (Standard) gibt es keine Rückfrage — der Client bleibt ein
allgemeines Diktierwerkzeug.

**3a. Diktiermikro-Modus** (Admin → Clients). Bei SpeechMike/PowerMic gilt standardmäßig Push-to-Talk (`dictation_mic_mode`:
`"hold"`, Record-Taste halten, beim Loslassen wird gestoppt und der Text automatisch eingefügt). Mit
`"toggle"` lässt sich das bisherige Verhalten (Druck startet, Druck stoppt) wiederherstellen.

**3b. Live-Ansicht** (Admin → Clients). `live_view` (Default `true`) legt fest, ob die Live-Ansicht standardmäßig an ist. `false`
= Overlay ohne Live-Text (nur Orb) und ohne Live-Transkription auf dem Server; Nutzer können es im Overlay
selbst umschalten (gilt dann pro Rechner und überschreibt diesen Default).

**4. Citrix/RDP.** Ob die lokale Zwischenablage in eine Sitzung umgeleitet wird, bestimmt die Klinik-IT
(Zwischenablage-Umleitung der Sitzung). In Citrix-Clients (`wfica32.exe`, `cdviewer.exe`) tippt der Client
den Text und legt ihn dabei nicht in die lokale Zwischenablage; bei RDP wird per Strg+V eingefügt, die
Umleitung kann den Text also in die Sitzung tragen.
