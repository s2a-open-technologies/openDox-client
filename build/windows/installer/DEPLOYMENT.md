# openDox unter Windows verteilen (SCCM/Intune/GPO)

Die MSI (`opendox-client-installer.msi`, gebaut über `../../../build-windows-msi.sh`)
installiert die App maschinenweit nach `%ProgramFiles%\openDox\openDox\` und legt dort
eine `settings.json` mit der zum Build-Zeitpunkt eingebetteten Server-URL ab
(`settings.json.template` in diesem Verzeichnis). Für einen einzelnen Rechner reicht
das — ein Admin öffnet die Datei danach einmal im Texteditor und trägt die richtige
URL ein.

Für eine Verteilung an viele Rechner mit fester/unterschiedlicher URL **nicht** die
MSI pro Kunde neu bauen. Stattdessen: **eine** MSI für alle, und die passende
`settings.json` als zweiten, unabhängigen Schritt eurer Softwareverteilung an

```
%ProgramFiles%\openDox\openDox\settings.json
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
msiexec /i opendox-client-installer.msi /qn && copy /Y settings.json "%ProgramFiles%\openDox\openDox\settings.json"
```

Uninstall-Befehl unverändert `msiexec /x {ProductCode} /qn`.

## SCCM

Als Application mit einem skriptbasierten Deployment-Type (PowerShell/Batch), der
beide Schritte macht:

```powershell
Start-Process msiexec.exe -ArgumentList '/i opendox-client-installer.msi /qn' -Wait
Copy-Item .\settings.json 'C:\Program Files\openDox\openDox\settings.json' -Force
```

Alternativ als separates Configuration Item/Baseline, das nur die Datei durchsetzt —
praktisch, weil SCCM das dann laufend gegenprüfen und bei Abweichung reparieren kann.

## GPO

Über Group Policy **Preferences** → Dateien (nicht die klassische "Softwareinstallation
über GPO", die kann nur die MSI selbst, keine Zusatzdateien). Die Datei-Preference
kopiert `settings.json` an den Zielpfad, unabhängig von der MSI-Installation, pro
OU/Gruppe unterschiedlich konfigurierbar.
