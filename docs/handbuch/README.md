# Handbücher des openDox-Clients

Quelle der Handbücher für den Desktop-Client. Geschrieben wird Markdown; PDF,
DOCX und XWiki-Seiten werden daraus erzeugt. Aufbau, Vorlage und Build-Script
sind dieselben wie im Server-Repo (`docs/handbuch/` dort), damit alle
openDox-Handbücher gleich aussehen.

| Handbuch | Datei | Zielgruppe |
|---|---|---|
| Benutzerhandbuch | `benutzer/handbuch.md` | Alle, die mit dem Client diktieren und einfügen |
| Installations- und Betriebshandbuch (geplant) | `installation/handbuch.md` | IT / Betrieb (Installation, `settings.json`, Verteilung) |

Ein neues Handbuch ist ein Ordner mit einer `handbuch.md`; das Build-Script
findet es von selbst.

## Bauen

```bash
scripts/build_handbuch.sh                  # alle, PDF + DOCX
scripts/build_handbuch.sh benutzer         # nur eins
FORMATS="docx xwiki" scripts/build_handbuch.sh
```

Ergebnis in `build/handbuch/` (nicht versioniert), z. B.
`openDox-Client-Benutzerhandbuch-0.8.3.pdf`. Benötigt nur Docker (Pandoc
`pandoc/core`, Typst `ghcr.io/typst/typst`) und ImageMagick für die
Bild-Platzhalter. Die Versionsnummer kommt aus `wails.json`
(`info.productVersion`).

## Release

Jedes Release bringt das Handbuch mit: `.github/workflows/release.yml` baut
beim Push eines `vX.Y.Z`-Tags die PDFs (Version aus dem Tag) und hängt sie als
Assets an das GitHub Release. Schlägt der Build fehl, schlägt das Release fehl.

## Layout

- **PDF** (Hauptausgabe): `vorlage/handbuch.typst`, ein Pandoc-Template für
  Typst – eine Kopie der Server-Vorlage, nur die Zeile unter dem Deckblatt
  sagt „Desktop-Client …“. Änderungen am Layout in beiden Repos nachziehen.
- **Logo**: `vorlage/s2a-open-technologies.svg` (weiß, fürs Deckblatt); die
  dunkle Variante für die Kopfzeile erzeugt das Build-Script.
- **Schriften**: `vorlage/fonts/` (Poppins für Überschriften, Lato für Text,
  beide SIL Open Font License, Lizenztexte liegen daneben).
- **DOCX** (bearbeitbare Nebenausgabe): optional `vorlage/reference.docx` mit
  Pandoc-Formatvorlagen.

## Schreiben

- Sie-Form, sachlich, kurze Sätze. UI-Bezeichnungen **fett** und exakt wie im
  Frontend (`frontend/src/components/*.tsx`).
- Tastenkürzel für Windows/Linux (**Strg+Alt**) schreiben; die Mac-Variante
  steht in der Tabelle im Kapitel *Tastenkürzel*, als **Cmd+Option** (die
  Zeichen ⌘/⌥ fehlen in Lato und kämen nur aus einer Ersatzschrift).
- Bilder: `![Bildunterschrift](bilder/<handbuch>/NN-name.png)`, jedes neue
  Bild auch in `screenshots.md` eintragen.
- Der Hinweis „Kein Medizinprodukt“ gehört in jedes Handbuch.
- Ändert sich die Oberfläche, das Handbuch im selben Commit nachziehen.
