# Screenshot-Liste

Alle Bilder, die die Handbücher referenzieren. Fehlende Dateien ersetzt
`scripts/build_handbuch.sh` beim Bauen durch einen grauen Platzhalter.

**Konventionen**

- PNG, nur das Client-Fenster (kein ganzer Bildschirm), Skalierung 100 % bzw.
  Retina-Aufnahme, helles Design (**Einstellungen → Design → Hell**).
- Windows-Aufnahmen bevorzugt (dort zeigt der Client „Einfügen in: …“).
- Nur Demo-Daten: keine echten Patienten-, Personen- oder Servernamen, keine
  echten Fenstertitel im Einfügeziel, keine Passwörter.
- Dateiname exakt wie unten, Ablage unter `docs/handbuch/<Pfad>`.
- Im Markdown mit fester Breite einbinden (`![…](bilder/…png){width=6.5cm}`): Die Vorlage
  zieht Bilder sonst auf Seitenbreite, das schmale Client-Fenster füllte dann eine ganze Seite.

## Benutzerhandbuch

| Datei | Was zu sehen ist | Rolle |
|---|---|---|
| `bilder/benutzer/01-anmeldung.png` | Anmeldebildschirm, idealerweise mit **Mit SSO anmelden** | Benutzer |
| `bilder/benutzer/02-hinweis.png` | Hinweis **Kein Medizinprodukt** bei der ersten Anmeldung | Benutzer (neues Konto) |
| `bilder/benutzer/03-uebersicht.png` | Fenster im Zustand **Bereit**, Statusleiste vollständig (mehrere Mikrofone, damit das Mikrofon-Symbol sichtbar ist) | Benutzer |
| `bilder/benutzer/04-aufnahme-laeuft.png` | Laufende Aufnahme mit Live-Text, Pause/Stopp/Abbrechen sichtbar | Benutzer |
| `bilder/benutzer/05-aufnahme-ohne-live.png` | Laufende Aufnahme mit ausgeschalteter Live-Ansicht (leuchtender Kreis) | Benutzer |
| `bilder/benutzer/06-fertiger-text.png` | Fertiger Text, **Einfügen in:** mit Demo-Programm, Schaltflächen Akzeptieren / Neue Aufnahme / In openDox öffnen | Benutzer |
| `bilder/benutzer/07-einstellungen.png` | Dialog **Einstellungen** (Design, Live-Ansicht, Mikrofontaste) | Benutzer |
