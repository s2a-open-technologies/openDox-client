# Einordnung in Audit-Rahmenwerke — openDox Client

> Stand: 2026-10-08 · gilt für openDox Client ab Version 0.8.0 (inkl. `[Unreleased]`)
> Pro Rahmenwerk: **Was es prüft** (neutral) und **Projekt-Bezug** (was belegbar erfüllt ist, was
> nicht). Das ist eine Orientierung für Betreiber und Auditoren, **keine Zertifizierung und keine
> Rechtsberatung**. Der Client ist nur ein Teil des Gesamtsystems; Server-Themen (Rollen,
> Verschlüsselung at rest, Aufbewahrung) stehen im Dokument des openDox-Server-Repositorys.
> Belege im Einzelnen: [`audit-checklist-opendox-client.md`](audit-checklist-opendox-client.md).

## ISO/IEC 27001

**Was es prüft:** Ein Informationssicherheits-Managementsystem (ISMS) der *Organisation*: Risikoanalyse,
Richtlinien, Zuständigkeiten, Maßnahmen aus Annex A, kontinuierliche Verbesserung. Zertifiziert
werden Organisationen, keine einzelnen Software-Produkte.

**Projekt-Bezug:** Das Projekt kann nicht „ISO-27001-konform“ sein, liefert aber Nachweise für
Maßnahmen im Betrieb des Betreibers: Meldeweg für Schwachstellen (`.github/SECURITY.md`),
Abhängigkeits- und Lizenzprüfung (`scripts/check_licenses.py`, Dependabot), CI mit Tests,
Änderungshistorie (`CHANGELOG.md`). Lücken: Builds nicht signiert (Windows) bzw. nicht notarisiert
(macOS), keine CSP, kein CVE-Gate in CI (`govulncheck`), Token im `localStorage`.

## BSI IT-Grundschutz

**Was es prüft:** Standardisierte Bausteine (z. B. APP, SYS, NET) mit Anforderungen für
Basis-, Standard- und erhöhten Schutzbedarf, angewendet auf den *Informationsverbund* des Betreibers.

**Projekt-Bezug:** Relevant sind für den Client vor allem Bausteine zu Client-Anwendungen,
Windows/macOS-Clients und Protokollierung. Belegbar: keine Telemetrie, eine einzige Server-Adresse als
Kommunikationsziel, Zwischenablage-Schutz (Windows), konfigurierbare Allow-Liste für Zielprogramme,
verteilbar per Intune/SCCM/GPO (`build/windows/installer/DEPLOYMENT.md`). Betreiberpflicht:
Festplattenverschlüsselung, Patchmanagement
(es gibt bewusst keinen Auto-Update-Mechanismus).

## B3S Krankenhaus / § 75c SGB V

**Was es prüft:** Der branchenspezifische Sicherheitsstandard (B3S) für die Gesundheitsversorgung im
Krankenhaus konkretisiert, wie Krankenhäuser den „Stand der Technik“ der IT-Sicherheit nach
§ 75c SGB V umsetzen (u. a. ISMS, Notfallmanagement, Netzsegmentierung, Härtung).

**Projekt-Bezug:** Der Client ist ein Baustein im Arbeitsplatz-Umfeld des Krankenhauses, das
Gesamtkonzept liegt beim Betreiber. Der Client unterstützt: kein lokaler Datenspeicher für Audio und
Texte, gehärtete Zwischenablage, Einfügeziel-Kontrolle. Nicht belegt: Signierung der Binärdateien
(Voraussetzung für Smart App Control, siehe Audit-Checkliste), formale Härtungsleitfäden.

## KHZG (Krankenhauszukunftsgesetz)

**Was es prüft:** Fördervoraussetzungen für digitale Vorhaben; ein Teil der Förderung ist an
Maßnahmen der IT-Sicherheit gekoppelt. Der Förderzeitraum ist weitgehend abgelaufen; relevant bleibt
der Nachweis der Umsetzung.

**Projekt-Bezug:** Das Projekt erfüllt keine Förderkriterien von sich aus. Es kann als
Dokumentationswerkzeug Teil eines geförderten Vorhabens sein; die Nachweispflicht liegt beim
Krankenhaus.

## NIS2 (NIS2UmsuCG)

**Was es prüft:** Risikomanagement-Maßnahmen, Meldepflichten bei erheblichen Sicherheitsvorfällen und
Lieferkettensicherheit für „besonders wichtige“ und „wichtige“ Einrichtungen (u. a. Gesundheitswesen).

**Projekt-Bezug:** Das Projekt ist selbst keine regulierte Einrichtung. Für Betreiber im Anwendungsbereich ist
der Client ein Teil der **Lieferkette**: SBOM-nahe Nachweise (`THIRD_PARTY_NOTICES.md`,
`go.sum`/`package-lock.json`), Meldeweg (`SECURITY.md`, 48-h-Antwort, 90-Tage-Policy). CycloneDX-SBOMs
(Go-Module, produktive npm-Pakete) werden vom Job `sbom` in `.github/workflows/release.yml` pro
Release erzeugt und angehängt (eingerichtet, noch nicht auf GitHub gelaufen).

## EU Cyber Resilience Act (CRA)

**Was es prüft:** Cybersicherheitsanforderungen an Produkte mit digitalen Elementen über den
Lebenszyklus (Security by Design, Schwachstellenmanagement, Updates, Dokumentation, Meldepflichten).

**Projekt-Bezug:** Anwendbar auf Produkte, die im Rahmen einer **kommerziellen Tätigkeit** auf
dem Markt bereitgestellt werden; rein nicht-kommerzielle Open-Source-Entwicklung ist ausgenommen.
Dieses Repository wird unter AGPL ohne kommerziellen Vertrieb betrieben (`AUTHORS`, `LICENSE`).
Das kann sich ändern, sobald der Client im Rahmen eines kostenpflichtigen Angebots an Kunden
ausgeliefert wird (die Planung eines Kunden-Rollouts steht im Raum, siehe Audit-Checkliste) — **das
ist vorab rechtlich zu klären**. Schon jetzt vorhanden: Disclosure-Prozess, Changelog,
Lizenz-/Abhängigkeitsprüfung. Vorhanden (eingerichtet, noch ungetestet auf GitHub): SBOM pro Release. Fehlt: signierte Releases, definierter Support-Zeitraum.

## EU MDR (Medizinprodukteverordnung)

**Was es prüft:** Ob Software nach ihrer **Zweckbestimmung** ein Medizinprodukt ist (Art. 2 Nr. 1;
Regel 11 in Anhang VIII für Software, die Informationen für diagnostische oder therapeutische
Entscheidungen liefert).

**Projekt-Bezug:** Bewusste Abgrenzung: Dokumentationsunterstützung (Diktat → Text →
Zusammenfassung), keine Diagnose, Therapie oder Entscheidungsunterstützung. Ausführlich in
[`zweckbestimmung.md`](zweckbestimmung.md) und [`../DISCLAIMER.md`](../DISCLAIMER.md). **Grenzfälle:**
(1) Der **Hold-Modus** des Diktiermikros fügt ohne Klick ein; die Prüfung folgt erst danach.
(2) Die Zusammenfassung erzeugt der Server per LLM; Vorlagen, die Bewertungen oder Empfehlungen
erzeugen, würden die Abgrenzung aufheben (Art. 16 MDR). Beides liegt teilweise im Server und bei
der Konfiguration des Betreibers.

## DSGVO / BDSG

**Was es prüft:** Rechtmäßigkeit der Verarbeitung (Art. 6, bei Gesundheitsdaten Art. 9), Transparenz,
Betroffenenrechte, Auftragsverarbeitung (Art. 28), technische und organisatorische Maßnahmen
(Art. 32), Verzeichnis (Art. 30) und ggf. Folgenabschätzung (Art. 35).

**Projekt-Bezug:** Der Client hat **keine eigenen Auftragsverarbeiter** und speichert weder Audio
noch Texte lokal; er spricht nur mit dem konfigurierten Server. Datenfluss, lokale Speicherung und
bekannte Lücken: [`data-mapping.md`](data-mapping.md). Verantwortlich ist der Betreiber; § 203 StGB
(Berufsgeheimnis) und Mitbestimmungsrechte sind im Disclaimer genannt. Transportverschlüsselung:
der Client erzwingt sie nicht (`http://` wird akzeptiert), im Betrieb verlangt sie aber der
Server, und ohne `https://` funktioniert das Mikrofon nicht.

## Kernbotschaft

Der openDox Client ist ein schlankes, quelloffenes Diktier-Overlay ohne Telemetrie, ohne eigene
Datenhaltung und mit einer einzigen Kommunikationsstelle, dem Server des Betreibers. Er liefert
belegbare Bausteine für Sicherheits- und Datenschutznachweise, ersetzt aber weder ein ISMS noch
eine Zertifizierung. Offene Punkte sind benannt: nicht signierte Builds, keine Client-Warnung bei `http://`,
Hold-Modus ohne Bestätigung vor dem Einfügen, SBOM erst ab dem nächsten Release-Tag. Vor einem kommerziellen Rollout sind CRA
und die MDR-Abgrenzung rechtlich zu prüfen.
