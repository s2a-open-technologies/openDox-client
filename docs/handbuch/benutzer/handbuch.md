---
title: "openDox"
subtitle: "Benutzerhandbuch Client"
---

# Über dieses Handbuch

Dieses Handbuch richtet sich an alle, die mit dem openDox-Client diktieren und
den fertigen Text direkt in ein anderes Programm übernehmen: Ärztinnen und
Ärzte, Pflege- und Assistenzpersonal, Schreibkräfte. Der Client ist ein kleines
Fenster, das über allen anderen Programmen liegt. Sie diktieren, openDox
verschriftlicht, und der Text landet an der Cursorposition in Ihrer
Praxissoftware, Ihrem Mailprogramm oder Ihrer Textverarbeitung.

Die Weboberfläche von openDox (Aufnahmen verwalten, Kategorien, Wörterbücher)
beschreibt das *Benutzerhandbuch* von openDox. Installation und zentrale
Einstellungen des Clients sind Sache Ihrer IT.

Bezeichnungen von Schaltflächen, Menüs und Feldern sind im Text **fett**
gesetzt und entsprechen der deutschen Oberfläche. Tastenkürzel sind für
Windows und Linux angegeben (**Strg+Alt**); auf dem Mac gilt stattdessen
**Cmd+Option** (Befehlstaste und Wahltaste, im Client als ⌘+⌥ angezeigt).

## Wichtiger Hinweis: Kein Medizinprodukt

openDox ist **kein zugelassenes Medizinprodukt** gemäß Medizinproduktegesetz
(MPG) oder vergleichbaren Regularien (z. B. EU-MDR, FDA). Es dient
ausschließlich der Unterstützung bei der Dokumentation.

Alle maschinell erstellten Transkriptionen und Zusammenfassungen müssen von
qualifiziertem Fachpersonal überprüft werden, bevor sie verwendet oder in
eine Patientenakte übernommen werden. openDox ersetzt keine ärztliche
Diagnose, Beratung oder Therapieempfehlung.

Spracherkennung und KI-Zusammenfassung arbeiten nicht fehlerfrei. Typische
Fehler sind falsch erkannte Fachbegriffe, Medikamentennamen, Dosierungen und
Zahlen, vertauschte Seitenangaben (links/rechts) sowie ausgelassene oder
hinzugefügte Inhalte. Prüfen Sie den Text im Zielprogramm, bevor Sie ihn
speichern oder versenden.

## Was der Client macht

1. **Aufnahme:** Sie diktieren über das Mikrofon Ihres Rechners, ein Headset
   oder ein Diktiermikrofon. Während Sie sprechen, erscheint auf Wunsch eine
   Live-Vorschau des Textes.
2. **Verarbeitung:** Nach dem Beenden verschriftlicht der openDox-Server die
   vollständige Aufnahme ein zweites Mal und genauer und bereitet sie nach der
   gewählten **Kategorie** auf.
3. **Einfügen:** Mit **Akzeptieren und einfügen** schreibt der Client den
   fertigen Text in das Programm, in dem Ihr Cursor zuletzt stand.

Jede Aufnahme liegt danach auch in der openDox-Weboberfläche, wo Sie sie erneut
aufrufen können.

# Erste Schritte

## Voraussetzungen

- Der Client ist auf Ihrem Rechner installiert (Windows, macOS oder Linux) und
  von Ihrer IT mit der Adresse Ihres openDox-Servers eingerichtet.
- Ein Mikrofon. Ein Headset oder Diktiermikrofon liefert deutlich bessere
  Ergebnisse als ein eingebautes Laptop-Mikrofon.
- Ihre openDox-Zugangsdaten (dieselben wie für die Weboberfläche).

## Anmelden

![Anmeldung im Client](bilder/benutzer/01-anmeldung.png)

1. Starten Sie **openDox** über das Startmenü, das Dock oder die
   Programmübersicht.
2. Geben Sie **Benutzername** und **Passwort** ein und klicken Sie auf
   **Anmelden**.

Ist in Ihrer Einrichtung eine zentrale Anmeldung eingerichtet, sehen Sie
zusätzlich **Mit SSO anmelden**. Ein Klick öffnet Ihren Browser; nach der
Anmeldung dort wechseln Sie zurück in den Client, er ist dann angemeldet.

Sie bleiben angemeldet, bis Sie sich abmelden. Beim nächsten Start geht es
direkt los.

## Hinweis bei der ersten Anmeldung bestätigen

Beim ersten Anmelden zeigt openDox den Hinweis **Kein Medizinprodukt**. Lesen
Sie ihn, setzen Sie das Häkchen und klicken Sie auf **Bestätigen**. Die
Bestätigung gilt für Ihr Konto, also auch für die Weboberfläche. Mit
**Abmelden** brechen Sie ab.

![Hinweis bei der ersten Anmeldung](bilder/benutzer/02-hinweis.png)

## Das Fenster im Überblick

![Der Client nach dem Start](bilder/benutzer/03-uebersicht.png)

**Statusleiste (oben)**

- Der farbige Punkt zeigt den Zustand: grün **Bereit**, rot **Nicht
  erreichbar** (keine Verbindung zum Server), rot pulsierend **Aufnahme
  läuft...**, gelb **Aufnahme pausiert**.
- Das **Auge** schaltet die Live-Ansicht ein oder aus (siehe *Live-Ansicht*).
- Das **Zahnrad** öffnet die **Einstellungen**.
- Das **Fragezeichen** zeigt die **Tastenkürzel**.
- Das **Mikrofon** wählt das Eingabegerät (nur sichtbar, wenn mehrere
  angeschlossen sind).
- Die Auswahlliste daneben ist die **Kategorie**.
- **Abmelden** meldet Sie ab, das **X** ganz rechts beendet den Client.

**Textbereich (Mitte):** Hier erscheint der Text während und nach der
Aufnahme. Er lässt sich nicht markieren oder kopieren; in andere Programme
gelangt er nur über **Akzeptieren und einfügen**.

**Steuerung (unten):** Aufnahme starten und beenden, pausieren, abbrechen.
Darunter stehen die wichtigsten Tastenkürzel.

Das Fenster bleibt immer im Vordergrund. Sie verschieben es am oberen Rand.

## Berechtigungen auf dem Mac

macOS fragt beim ersten Start nach dem **Mikrofon**; erlauben Sie den Zugriff.
Für Tastenkürzel und das Einfügen braucht der Client zusätzlich zwei
Freigaben in **Systemeinstellungen → Datenschutz & Sicherheit**:

| Freigabe | Wofür |
|---|---|
| **Bedienhilfen** | Text in andere Programme einfügen |
| **Eingabeüberwachung** | Tastenkürzel, die in jedem Programm funktionieren |

Fehlt eine Freigabe, zeigt der Client einen Hinweis. Schalten Sie openDox in
der jeweiligen Liste ein und starten Sie den Client neu.

# Diktieren

## Kategorie und Mikrofon wählen

Wählen Sie vor der Aufnahme die passende **Kategorie** in der Statusleiste,
z. B. *Allgemein* oder *Arztbrief*. Sie bestimmt, wie openDox den Text
aufbereitet. Der Client merkt sich Ihre Wahl.

Sind mehrere Mikrofone angeschlossen, wählen Sie über das Mikrofon-Symbol das
richtige aus. Auch diese Wahl bleibt gespeichert.

## Aufnahme starten, pausieren und beenden

![Laufende Aufnahme mit Live-Ansicht](bilder/benutzer/04-aufnahme-laeuft.png)

| Aktion | Im Fenster | Tastenkürzel |
|---|---|---|
| Aufnahme starten | runde Schaltfläche mit Mikrofon | **Strg+Alt+R** |
| Pausieren / Fortsetzen | Pause-Schaltfläche neben der Aufnahme | **Strg+Alt+P** |
| Aufnahme beenden | runde Schaltfläche mit Quadrat | **Strg+Alt+R** |
| Aufnahme abbrechen | **X** links neben der Aufnahme | – |

Die Tastenkürzel funktionieren in jedem Programm; Sie müssen dafür nicht in das
openDox-Fenster klicken. Lassen Sie den Cursor also einfach dort stehen, wo der
Text später hin soll.

Beenden Sie die Aufnahme über die Schaltfläche, fragt der Client nach
(**Aufnahme beenden**). Über das Tastenkürzel endet sie ohne Rückfrage.

**Aufnahme abbrechen** verwirft die Aufnahme nach einer Rückfrage endgültig;
sie wird nicht gespeichert und kann nicht wiederhergestellt werden.

## Halten zum Diktieren

Halten Sie **Strg+Alt+D** gedrückt, solange Sie sprechen. Beim Loslassen endet
die Aufnahme, und sobald die Verarbeitung fertig ist, fügt der Client den Text
**automatisch** an der Cursorposition ein. Das entspricht dem gewohnten
Arbeiten mit einem Diktiermikrofon (siehe *Diktiermikrofon*).

> Prüfen Sie automatisch eingefügten Text genauso sorgfältig wie jeden anderen.

## Live-Ansicht

Mit eingeschalteter Live-Ansicht sehen Sie den Text, während Sie sprechen. Die
Live-Vorschau ist vorläufig; maßgeblich ist der Text nach der Verarbeitung.

Ist die Live-Ansicht aus, zeigt der Client während der Aufnahme nur einen
leuchtenden Kreis, der auf Ihre Stimme reagiert. Das Endergebnis ist dasselbe.
Manche Anwender finden das ruhiger; außerdem entlastet es den Server.

![Laufende Aufnahme ohne Live-Ansicht](bilder/benutzer/05-aufnahme-ohne-live.png)

Sie schalten die Live-Ansicht über das **Auge** in der Statusleiste oder in den
**Einstellungen** um. Hat Ihre Administration die Live-Vorschau für den ganzen
Server abgeschaltet, gibt es den Schalter nicht.

## Tipps für gute Ergebnisse

- Sprechen Sie in normalem Tempo und gleichmäßiger Lautstärke, etwa eine
  Handbreit vom Mikrofon entfernt.
- Diktieren Sie Satzzeichen nur, wenn Ihre Kategorie das vorsieht.
- Buchstabieren Sie seltene Eigennamen beim ersten Auftreten oder tragen Sie
  sie in ein Wörterbuch der Weboberfläche ein.
- Pausieren Sie bei Unterbrechungen, statt die Aufnahme laufen zu lassen.

## Unterbrochene Aufnahme fortsetzen

Bricht die Verbindung ab oder wird der Client während einer Aufnahme beendet,
bleibt die Aufnahme auf dem Server erhalten. Beim nächsten Start setzt der
Client sie automatisch fort, das bisher Gesagte bleibt erhalten.

# Text prüfen und einfügen

## Verarbeitung abwarten

Nach dem Beenden zeigt der Client **Verarbeite...**. Je nach Länge dauert das
einige Sekunden. Danach erscheint der fertige Text, und eine Meldung weist auf
**Strg+Alt+V** zum Einfügen hin.

## Akzeptieren und einfügen

![Fertiger Text mit Einfügeziel](bilder/benutzer/06-fertiger-text.png)

1. Lesen Sie den Text im Fenster.
2. Klicken Sie in Ihrem Zielprogramm an die Stelle, an der der Text stehen
   soll, sofern der Cursor nicht schon dort steht.
3. Drücken Sie **Strg+Alt+V** oder klicken Sie auf **Akzeptieren und
   einfügen** (Häkchen).

Unter Windows zeigt der Client vorher an, wohin er einfügt: **Einfügen in:**
mit dem Namen des Programms. Klicken Sie auf die Schaltfläche, gibt der Client
den Fokus kurz an dieses Programm zurück und fügt dort ein.

Eingefügt wird an der Cursorposition. Steht der Cursor in einem Feld, das
keinen Text annimmt, kommt nichts an; setzen Sie den Cursor dann neu und
fügen Sie erneut ein.

## Nicht freigegebene Programme

Ihre IT kann festlegen, in welche Programme ohne Rückfrage eingefügt werden
darf. Ist das Zielprogramm nicht freigegeben, erscheint **Einfügen
bestätigen**. Prüfen Sie, ob das richtige Fenster aktiv ist, und klicken Sie
auf **Trotzdem einfügen** oder auf **Abbrechen**. Ist die Einfügen-Anzeige
gelb, ist das aktuelle Ziel nicht freigegeben.

## Zwischenablage

Unter Windows legt der Client den Text beim Einfügen in die Zwischenablage,
markiert ihn aber so, dass er nicht im Zwischenablage-Verlauf (**Win+V**) und
nicht in der Cloud-Zwischenablage landet. Nach 30 Sekunden leert der Client die
Zwischenablage wieder, sofern Sie zwischendurch nichts anderes kopiert haben,
und ebenso beim Beenden. Ihre IT kann diese Zeit ändern.

Unter macOS und Linux bleibt der Text in der Zwischenablage, bis Sie etwas
anderes kopieren.

## Kategorie nachträglich wechseln

Passt die Aufbereitung nicht, wählen Sie bei fertigem Text eine andere
**Kategorie**. openDox verarbeitet dieselbe Aufnahme dann neu; Sie müssen nicht
erneut diktieren. Eine Meldung zeigt an, wann der neue Text bereitsteht.

## Weiter in der Weboberfläche

Mit **In openDox öffnen** öffnen Sie die openDox-Weboberfläche im Browser. Dort
finden Sie die Aufnahme mit Transkript und Zusammenfassung, können sie
bearbeiten oder später erneut aufrufen. Mit **Neue Aufnahme starten** (oder
**Strg+Alt+R**) beginnen Sie direkt die nächste Aufnahme.

# Diktiermikrofon

Der Client unterstützt die Tasten von Diktiermikrofonen der Reihen **Philips
SpeechMike** und **Nuance PowerMic**. Sie schließen das Gerät per USB an; der
Client erkennt es nach wenigen Sekunden von selbst. Als Mikrofon wählen Sie es
wie jedes andere über das Mikrofon-Symbol.

| Taste | Wirkung |
|---|---|
| Aufnahme (rot) | je nach Einstellung halten zum Diktieren oder Start/Stopp |
| Stopp | Aufnahme starten oder beenden |
| Zurückspulen | Pausieren / Fortsetzen |
| Einfügen (INS/OVR) | Akzeptieren und einfügen |

Wie sich die Aufnahmetaste verhält, stellen Sie in den **Einstellungen** unter
**Mikrofontaste** ein:

- **Halten** (Standard): Taste gedrückt halten, solange Sie sprechen. Beim
  Loslassen endet die Aufnahme, und der Text wird nach der Verarbeitung
  automatisch eingefügt.
- **Umschalten**: Ein Druck startet, der nächste beendet die Aufnahme.
  Eingefügt wird mit der Einfügen-Taste oder **Strg+Alt+V**.

> Die Belegung der übrigen Tasten kann je nach Gerätemodell abweichen. Funktioniert
> eine Taste nicht wie beschrieben, nutzen Sie die Tastenkürzel und melden Sie
> das Modell Ihrer IT.

Ist das openDox-Fenster ausgeblendet (**Strg+Alt+H**), reagieren weder die
Tasten des Diktiermikrofons noch die Tastenkürzel. Eine gehaltene Aufnahme
endet aber in jedem Fall beim Loslassen.

# Tastenkürzel

Alle Tastenkürzel gelten in jedem Programm. Über das **Fragezeichen** in der
Statusleiste zeigt der Client sie jederzeit an.

| Windows und Linux | Mac | Wirkung |
|---|---|---|
| **Strg+Alt+R** | **Cmd+Option+R** | Aufnahme starten / beenden |
| **Strg+Alt+P** | **Cmd+Option+P** | Pausieren / Fortsetzen |
| **Strg+Alt+V** | **Cmd+Option+V** | Akzeptieren und einfügen |
| **Strg+Alt+D** (halten) | **Cmd+Option+D** (halten) | Halten zum Diktieren |
| **Strg+Alt+H** | **Cmd+Option+H** | Fenster aus- und einblenden |

Ist ein Tastenkürzel bereits von einem anderen Programm oder vom
Betriebssystem belegt, kann der Client es nicht nutzen und zeigt einen
Hinweis. Unter Linux mit Xfce etwa ist **Strg+Alt+D** für „Desktop anzeigen“
vergeben.

# Einstellungen

![Einstellungen](bilder/benutzer/07-einstellungen.png)

Über das **Zahnrad** in der Statusleiste öffnen Sie die **Einstellungen**:

| Einstellung | Auswahl | Wirkung |
|---|---|---|
| **Design** | **Dunkel** / **Hell** | Farbgebung des Fensters |
| **Live-Ansicht** | **An** / **Aus** | Text während des Sprechens anzeigen |
| **Mikrofontaste** | **Halten** / **Umschalten** | Verhalten der Aufnahmetaste am Diktiermikrofon |

Jede Änderung gilt sofort und bleibt auf diesem Rechner für Ihr Benutzerkonto
gespeichert, auch nach einem Neustart oder Update. Das Design wechseln Sie
auch auf dem Anmeldebildschirm über das Sonnen- bzw. Mond-Symbol.

Die Ausgangswerte legt Ihre IT oder Administration fest. Mit **Auf Standard
zurücksetzen** gelten wieder diese Vorgaben.

Einstellungen zur Zwischenablage und zu freigegebenen Programmen können Sie
nicht selbst ändern; sie gehören zum Datenschutz Ihrer Einrichtung.

# Fenster ausblenden, abmelden, beenden

- **Strg+Alt+H** blendet das Fenster aus und wieder ein. Ausgeblendet nimmt der
  Client keine Tastenkürzel an. Verkleinern Sie das Fenster nicht über die
  Taskleiste: Dann kann die Verarbeitung hängen, bis es wieder offen ist.
- **Abmelden** in der Statusleiste meldet Sie ab. Nutzen Sie das an Rechnern,
  an denen mehrere Personen arbeiten.
- Das **X** oben rechts beendet den Client.

# Häufige Fragen und Probleme

**Der Punkt in der Statusleiste ist rot, „Nicht erreichbar“.**
Der Client erreicht den openDox-Server nicht. Prüfen Sie Ihre
Netzwerkverbindung. Hält der Zustand an, wenden Sie sich an Ihre IT.

**„Kein Mikrofon gefunden“ oder „Mikrofonzugriff verweigert“.**
Prüfen Sie, ob das Mikrofon angeschlossen ist und im Mikrofon-Symbol gewählt
ist. Auf dem Mac erlauben Sie den Zugriff unter **Systemeinstellungen →
Datenschutz & Sicherheit → Mikrofon**.

**Die Tastenkürzel tun nichts.**
Ist das Fenster ausgeblendet? Blenden Sie es mit **Strg+Alt+H** ein. Auf dem
Mac fehlt sonst meist die Freigabe **Eingabeüberwachung** (siehe
*Berechtigungen auf dem Mac*).

**Der Text wird nicht eingefügt.**
Prüfen Sie, ob der Cursor im richtigen Feld steht, und fügen Sie erneut mit
**Strg+Alt+V** ein. Auf dem Mac ist die Freigabe **Bedienhilfen** nötig.

**„Verbindung unterbrochen – bitte Aufnahme neu starten“.**
Die Verbindung zum Server ist abgebrochen, etwa durch WLAN-Wechsel oder
Standby. Starten Sie die Aufnahme neu; meist setzt der Client die bisherige
Aufnahme fort.

**„Zeitüberschreitung – Aufnahme konnte nicht verarbeitet werden“.**
Die Verarbeitung hat zu lange gedauert. Die Aufnahme ist nicht verloren:
Öffnen Sie sie über **In openDox öffnen** in der Weboberfläche. Tritt das
öfter auf, melden Sie es Ihrer Administration.

**Ein Fachbegriff wird immer wieder falsch erkannt.**
Tragen Sie ihn in der openDox-Weboberfläche in eines Ihrer Wörterbücher ein.

# Glossar

| Begriff | Erklärung |
|:-----------|:-------------------------------|
| Client | Das openDox-Programm auf Ihrem Rechner, das dieses Handbuch beschreibt. |
| Weboberfläche | openDox im Browser, mit allen Aufnahmen, Kategorien und Wörterbüchern. |
| Live-Ansicht | Vorläufiger Text während der Aufnahme; nicht das Endergebnis. |
| Kategorie | Vorlage, die bestimmt, wie openDox den diktierten Text aufbereitet. |
| Zielprogramm | Das Programm, in das der Client den fertigen Text einfügt. |
| Diktiermikrofon | Handmikrofon mit Tasten, z. B. Philips SpeechMike oder Nuance PowerMic. |
| Halten zum Diktieren | Aufnahme läuft, solange eine Taste gedrückt ist (Push-to-Talk). |

# Impressum und Lizenz

openDox wird von der S2A Open Technologies gGmbH entwickelt und als freie
Software unter der GNU Affero General Public License, Version 3 oder später
(AGPL-3.0-or-later), veröffentlicht. Die Software wird ohne Gewährleistung
bereitgestellt; Einzelheiten regelt die Lizenz.

Für den Betrieb der jeweiligen Installation, einschließlich Datenschutz und
der Einhaltung medizinrechtlicher Vorschriften, ist die betreibende
Einrichtung verantwortlich. Bei Fragen zur Nutzung wenden Sie sich zuerst an
Ihre interne IT oder Administration.
