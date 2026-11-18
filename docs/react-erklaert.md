# React erklärt am Catch-me-Projekt

Die editierbare Skizze dazu liegt in [react-ueberblick.drawio](react-ueberblick.drawio).
Öffne sie mit diagrams.net über **Datei → Öffnen von → Gerät** oder in einer draw.io-Erweiterung deiner IDE.
Sie enthält zwei Seiten: **Projektaufbau** und **React-Datenfluss**.

## 1. Was macht React hier?

React beschreibt die Oberfläche anhand der aktuellen Daten. Du musst nicht selbst ein Textfeld suchen und seinen Inhalt aktualisieren oder einen Ladeindikator einblenden. Stattdessen beschreibst du, was bei einem bestimmten Zustand zu sehen sein soll.

**Merksatz: Oberfläche = Funktion von State und Props.**

Die beteiligten Werkzeuge sind unterschiedlich zuständig:

| Werkzeug | Aufgabe im Projekt |
| --- | --- |
| React (19.3) | Komponenten, State, Hooks und Berechnung der Oberfläche |
| React Native (0.88) | Darstellung mit nativen UI-Bausteinen auf Android und iOS |
| Expo (SDK 58) | Entwicklungswerkzeuge und Gerätefunktionen, etwa Standortzugriff |
| Expo Router | Navigation zwischen Screens anhand der Dateien in `app/src/app/` |
| TypeScript | Prüft Typen beim Entwickeln; `.tsx` erlaubt zusätzlich JSX |
| Go-Backend | Verbindlicher Spielzustand, Regeln und erlaubte Sichtbarkeit |
| SQLite / Ent | Persistenz im Backend, nicht React-State auf dem Handy |

Im Browser wird React Native Web verwendet. Die Karte hat dort nur eine Ersatzdarstellung; die native Spielplatzauswahl ist nicht im Browser implementiert.

## 2. Komponenten: Die Oberfläche besteht aus Bausteinen

Eine React-Komponente ist hier eine Funktion, die JSX zurückgibt. JSX sieht ähnlich wie HTML aus, beschreibt aber React-Elemente. Auf dem Handy sind `<View>`, `<Text>` und `<Pressable>` React-Native-Bausteine, keine HTML-Tags.

Der vereinfachte Aufbau ist:

```text
RootLayout (_layout.tsx)
└── Stack: Navigation zwischen den Screens
    ├── HomeScreen (index.tsx)       → Beitreten
    ├── CreateScreen (create.tsx)    → Spielplatz und Spiel erstellen
    └── LobbyScreen (lobby.tsx)      → Lobby und laufendes Spiel
```

Die Screens verwenden kleinere Komponenten: `Button` wird mehrfach genutzt, `PlaygroundMap` sowohl beim Erstellen als auch in der Lobby. Der Stack verwaltet die Screens; das bedeutet nicht, dass alle drei gleichzeitig sichtbar sind.

## 3. Props: Einstellungen für eine Komponente

In `app/src/app/index.tsx` steht:

```tsx
<Button label="Join game" onPress={() => void join()} disabled={busy || !canJoin} />
```

`label`, `onPress` und `disabled` sind **Props**: Werte, die der Elternbaustein an den Kindbaustein übergibt. In `app/src/components/button.tsx` empfängt `Button` sie und verwendet sie für `Pressable` und `Text`.

- `label`: Was steht auf dem Button?
- `disabled`: Darf man ihn drücken?
- `onPress`: Welche Funktion wird beim Drücken ausgeführt?

Der Button muss nichts über Teamcodes oder HTTP wissen. Diese Logik bleibt im Screen. Props werden vom Kind nicht verändert. Ein Callback ist eine als Prop übergebene Funktion, über die das Kind ein Ereignis an den Elternbaustein melden kann.

## 4. State: Das Gedächtnis einer Komponenteninstanz

In `HomeScreen` findest du:

```tsx
const [name, setName] = useState('');
```

- `name` ist der aktuelle Wert für diesen Renderdurchlauf.
- `setName` fordert eine Änderung an und veranlasst React, erneut zu rendern.
- `''` ist der Anfangswert beim ersten Einbinden der Komponente.

React bewahrt diesen State zwischen Renderdurchläufen auf. Eine gewöhnliche lokale Variable wäre dagegen bei jedem Funktionsaufruf neu angelegt.

Das Textfeld ist an den State gekoppelt:

```tsx
<TextInput value={name} onChangeText={setName} />
```

Das ist ein **kontrolliertes Eingabefeld**. Wenn du „Anna“ eingibst:

1. Das Textfeld meldet den neuen Text über `onChangeText`.
2. `setName('Anna')` fordert die State-Änderung an.
3. React führt `HomeScreen` mit dem neuen State erneut aus.
4. `canJoin` wird aus Code und Name neu berechnet.
5. Das neue JSX beschreibt unter anderem, ob der Button aktiviert sein soll.
6. React gleicht den neuen Elementbaum mit dem bisherigen ab; der Renderer übernimmt die nötigen UI-Änderungen.

**Rendern bedeutet nicht, den ganzen Bildschirm zu löschen und neu aufzubauen.** Es bedeutet zunächst, die neue Beschreibung der Oberfläche zu berechnen. React kann mehrere State-Änderungen zusammenfassen; ein Setter ist auch keine sofortige Zuweisung an die Variable des bereits laufenden Funktionsaufrufs.

## 5. Abgeleitete Werte und bedingte Darstellung

`canJoin` in `index.tsx` und die Spielplatzvorschau in `create.tsx` werden aus vorhandenen Daten berechnet. Dafür ist kein zusätzlicher State nötig. So entstehen keine zwei Werte, die man mühsam synchron halten muss.

JSX kann abhängig vom State unterschiedliche Elemente enthalten:

```tsx
{busy && <ActivityIndicator color={colors.accent} />}
```

Bei `busy === true` ist der Ladeindikator Teil der Oberfläche, sonst nicht. Das Projekt verwendet dasselbe Muster für Fehler, Join-Codes, Routen und Ready-Buttons.

## 6. Props hinunter, Ereignisse zurück: Das Kartenbeispiel

`CreateScreen` besitzt den ausgewählten Mittelpunkt im State `selected`. Es berechnet daraus den Spielplatz und übergibt ihn an die Karte:

```tsx
<PlaygroundMap
  playground={playground}
  editable={!created && !busy}
  onSelect={selectCenter}
/>
```

Die native Karte ruft bei einem erlaubten Tap `onSelect(...)` mit den Koordinaten auf. `selectCenter` validiert sie und ruft `setSelected(center)` auf. Anschließend berechnet der Screen die Vorschau neu und gibt aktualisierte Props an die Karte weiter.

**Daten fließen vom Elternbaustein zum Kind; das Kind meldet Ereignisse über Callbacks.** Der Import `@/components/playground-map` wird auf Android/iOS zur Datei `playground-map.native.tsx` aufgelöst, für die Browser-Vorschau zur Datei `playground-map.tsx`.

## 7. Hooks und Effekte: Die Verbindung zur Außenwelt

Hooks sind Funktionen, mit denen Komponenten React-Funktionen wie State und Effekte nutzen. Sie müssen in gleichbleibender Reihenfolge auf oberster Ebene der Komponente bzw. eines eigenen Hooks aufgerufen werden, nicht nur innerhalb eines `if` oder einer Schleife.

Im Projekt gibt es mehrere Beispiele:

- **`useState`**: hält Namen, Ladezustand und den empfangenen Spielzustand.
- **`useEffect`**: synchronisiert mit externen Systemen. Die native Karte passt nach dem Rendern ihren Ausschnitt an die Grenzen an. Der Standort-Hook richtet Standortbeobachtung und Timer ein und räumt sie wieder auf.
- **`useRef`**: hält einen Wert oder eine Referenz über Renderdurchläufe hinweg. `map.current` verweist auf die Karte. Änderungen an `.current` lösen selbst keinen neuen Renderdurchlauf aus.
- **`useCallback`**: bewahrt eine Funktionsreferenz auf, solange ihre Abhängigkeiten gleich bleiben. In der Lobby unterstützt das stabile Abhängigkeiten für die Synchronisierung; es führt die Funktion nicht automatisch aus.
- **`useFocusEffect`** aus Expo Router: Die Lobby startet beim Fokus sofort eine Abfrage und pollt danach alle zwei Sekunden. Die zurückgegebene Cleanup-Funktion stoppt Polling und Uhr beim Fokusverlust oder Entfernen des Screens.
- **`useLocationSharing`**: ein eigener Hook in `app/src/use-location-sharing.ts`. Er bündelt Berechtigungen, Standortbeobachtung, Versand und Aufräumen. Ein eigener Hook ist kein automatisch geteilter globaler State.

Effekte sind nicht für gewöhnliche Berechnungen wie `canJoin` nötig. Ein Request als direkte Folge eines Button-Taps kann dagegen in dessen Ereignisfunktion stehen, wie `join()`.

## 8. Was geschieht beim Beitreten und in der Lobby?

Beim Beitreten setzt `join()` zunächst `busy` auf `true`. Die Oberfläche zeigt den Ladeindikator und sperrt die Eingaben. Dann wird `joinGame(code, name)` aufgerufen:

```text
HomeScreen → api/client.ts → generierter Fetch-Client
           → HTTP → Go-Backend → Ent / SQLite
```

Der API-Wrapper normalisiert Eingaben, behandelt Fehler und prüft Antwortdaten zur Laufzeit. TypeScript allein kann nicht garantieren, dass ein Server gültiges JSON liefert.

Bei Erfolg wird der Spieler-Token in `playerSession` gehalten und `router.replace('/lobby')` aufgerufen. `playerSession` ist ein normales Modul mit einer Variable, **kein React-State**: `set()` löst selbst kein Rendern aus, und der Token geht beim App-Neustart verloren.

In der Lobby liefert `getGameState(token)` den erlaubten Spielzustand. `show(state)` speichert ihn mit `setSnapshot(...)` im lokalen React-State. Dadurch werden Spielerlisten, Karte und Status neu berechnet. Auch Ready- und Positionsanfragen können einen neuen Snapshot liefern.

Die lokale Uhr aktualisiert die Countdown-Anzeige. Sie macht das Handy aber nicht zum Schiedsrichter: Die Spielregeln und verbindlichen Zustandswechsel liegen im Backend. Verborgene Positionen werden serverseitig gefiltert, nicht bloß mit JSX versteckt. Fangen, Zielankunft und Spielende sind noch nicht implementiert.

## Empfohlene Lesereihenfolge

1. `app/src/components/button.tsx`: eine kleine Komponente und ihre Props.
2. `app/src/app/index.tsx`: State, Eingabe und asynchrone Aktion.
3. `app/src/app/create.tsx`: berechnete Vorschau und Karten-Callback.
4. `app/src/app/lobby.tsx`: Backend-Snapshot, Polling und Darstellung.
5. `app/src/use-location-sharing.ts`: eigene Hooks, Effekte und Cleanup.

Weiterführend: [React: Oberfläche beschreiben](https://react.dev/learn/describing-the-ui), [React: State](https://react.dev/learn/state-a-components-memory), [Expo SDK 58](https://docs.expo.dev/versions/v58.0.0/), [Expo Router](https://docs.expo.dev/router/introduction/).
