# Event-Driven Refactor Plan

Rebuild the application around a small, hand-rolled event bus. Technical
operations (fsnotify callbacks, debounce timers, UI clicks) are translated into
granular, readable domain events; all reactions live in small per-concern
handler files.

Neither Go nor Fyne provides an application-level event bus (Fyne only offers
`fyne.Do` for thread marshaling and `data/binding` for widget values), so the
bus is ~60 lines of channel-based custom code. No new dependencies.

## Design principles

1. **One bus goroutine owns all state.** Producers (fsnotify loop, debounce
   timers, UI callbacks, upload goroutines) only `Publish`. Handlers run
   sequentially on the single bus goroutine, so state mutation is
   single-threaded and the current mutexes (except UI-facing ones) disappear.
   Event processing order = publish order.
2. **Granular events, shared handlers.** Every distinct happening gets its own
   event type. Subscribing the same handler to several event types costs one
   line each — this is intentional and preferred over generic events with
   type fields.
3. **Intent vs. fact naming.** UI emits `...Requested` intents; handlers emit
   past-tense facts after validation/persistence
   (e.g. `WatchDirChangeRequested` → validate, save config, restart watcher →
   `WatchDirChanged`).
4. **The log panel is a projection.** `handler_log.go` subscribes to fact
   events and appends `LogEntry`s. Business handlers never call `addLog`.
5. **Not everything becomes an event.** Pure-UI mechanics with no business
   meaning (tray Show/Quit) stay direct.

## Threading rules

1. Handlers run on the bus goroutine; they may mutate `State` freely, no locks.
2. Any UI touch from a handler is wrapped in `fyne.Do(...)` (same as today).
3. Slow work (network uploads) is spawned as a goroutine from the handler and
   reports back by publishing `UploadSucceeded` / `UploadFailed` — the event
   loop stays responsive and uploads stay concurrent.
4. The `widget.List` log callbacks read the log slice from Fyne's thread, so
   the log store keeps its own small mutex (the only one that survives).

## Event vocabulary (new file `events.go`)

### Watcher facts (from fsnotify, after debounce — debounce stays in the producer)

| Event | Payload | Emitted when |
|---|---|---|
| `WatchStarted` | `Dir` | watcher (re)started on H3 root |
| `WatchFailed` | `Dir, Err` | fsnotify init/add failed |
| `GamesDirAppeared` | `Dir` | `<root>/Games` created at runtime |
| `GamesDirRemoved` | — | `<root>/Games` deleted at runtime |
| `PasswordsCreated` | `Path` | passwords.txt create (pre-parse) |
| `PasswordsModified` | `Path` | passwords.txt write (pre-parse) |
| `PasswordsLoaded` | `Info GameInfo` | parse succeeded; drives game-folder resolution + log |
| `PasswordsInvalid` | `Path, Err` | read/parse failed; clears GameInfo |
| `BattleSaveDetected` | `Path` | `Games/BATTLE.GM2` written |
| `BattleNonContinuableSaveDetected` | `Path` | `Games/BATTLE_non_continuable.GM2` written |
| `TurnBeginSaveDetected` | `Path` | `Games/TURN_BEGIN.GM2` written |
| `GameBeginSaveDetected` | `Path` | game folder `GAME_BEGIN.GM2` written |
| `TurnEndSaveDetected` | `Path, Turn int` | game folder `N.GM2` written; turn number parsed from filename |

### Game folder

| Event | Payload | Note |
|---|---|---|
| `GameFolderResolveRequested` | — | intent; emitted after `PasswordsLoaded`, 5s debounce |
| `GameFolderResolved` | `Folder` | watch switched, initial scan runs |
| `GameFolderNotFound` | `Opponent, Err` | 60s retry loop keeps ticking |

### Uploads

| Event | Payload | Note |
|---|---|---|
| `UploadStarted` | `Path` | keeps the auto-restart wait counter working |
| `UploadSucceeded` | `Path` | updates sent-folders cache + log |
| `UploadAlreadyOnServer` | `Path` | server said `already_exists`; mark as sent |
| `UploadSkippedDuplicate` | `Path` | same hash already uploaded for this file type |
| `UploadFailed` | `Path, Kind, Err` | `Kind` enum (ReadError, ConnectionRefused, ServerError, ...) mapped to existing translation keys by the log handler — avoids 8 separate failure event types |

The initial folder scan publishes the same `...SaveDetected` events for unsent
files instead of calling upload directly — one code path for all uploads.

### UI intents / facts

| Event | Payload | Note |
|---|---|---|
| `WatchDirChangeRequested` | `Dir` | from browse dialog / auto-discovery |
| `WatchDirChanged` | `Dir` | fact after validation + config save + watcher restart |
| `WatchDirInvalid` | `Dir, Err` | `resolveH3Root` failed → error dialog |
| `LanguageChanged` | `Lang` | flag clicked → config save + relabel + log refresh |
| `StartupToggleRequested` | — | startup button clicked |
| `StartupEnabled` / `StartupDisabled` | — | facts; relabel button + info dialog |

## File layout after refactor

**New files:**

- `events.go` — `Event` interface + all event structs (~100 lines, the
  readable vocabulary of the app)
- `bus.go` — `Bus` type: `Subscribe` / `Publish` / `Run` (~60 lines)
- `state.go` — slim `State` struct (watchDir, gameInfo, watchedGameFolder,
  gamesDirWatched, isInitialRun, lastUploadedHash, sentFoldersCache,
  instanceID, lang) owned by the bus goroutine, plus `uiRefs` (widget pointers
  needed by UI-touching handlers)
- `handler_passwords.go` — `PasswordsCreated/Modified` → parse →
  `PasswordsLoaded`/`PasswordsInvalid`; `PasswordsLoaded` → store GameInfo,
  emit `GameFolderResolveRequested`
- `handler_gamefolder.go` — resolve + 60s retry + watch switch + initial scan
- `handler_uploads.go` — the 5 save-detected events → hash dedup → upload
  goroutine → publish result events; owns upload counter (auto-restart wait)
- `handler_language.go` — `LanguageChanged` → persist config, relabel widgets,
  refresh log list
- `handler_watchdir.go` — `WatchDirChangeRequested` → validate root → save
  config → restart watcher → `WatchDirChanged`; first-run startup prompt
- `handler_startup.go` — `StartupToggleRequested` → dialog flow →
  `StartupEnabled`/`StartupDisabled`
- `handler_log.go` — fact events → `LogEntry` append + list refresh

**Shrunk files:**

- `watcher.go`: 465 → ~150 lines; pure producer: fsnotify setup + debounce +
  publish. No business calls left.
- `uploader.go`: becomes a pure function returning a result; no `App`, no
  `addLog`.
- `passwords.go`: keeps only the pure parsing (`parsePasswordsTxt`,
  `stripBadges`); `loadPasswordsFile` dissolves into `handler_passwords.go`.

**Dissolved:**

- `app.go` — split into `state.go` + the handler files.

**Untouched:**

- `updater.go`, `discovery.go`, `config.go`, `translations.go`,
  `sentfolders.go`, `startup_windows.go`, `startup_notwindows.go`,
  `assets.go`, `buildvars.go`, `resolver.go`

**`main.go`** becomes pure wiring: build state → build bus → register handlers
→ start producers → run.

## Steps

Each step ends with a green `./build.sh`. Steps are independent enough to
stop after any of them and resume later — tick the checkbox when done.

- [x] **Step 1: Bus skeleton.**
  Add `events.go` (interface + event structs) and `bus.go`
  (`Subscribe`/`Publish`/`Run`). Instantiate and start the bus in `main.go`.
  No behavior change yet. Verify: `./build.sh`.

- [x] **Step 2: Passwords pipeline.**
  Watcher publishes `PasswordsCreated`/`PasswordsModified` (post-debounce)
  instead of calling `loadPasswordsFile`. New `handler_passwords.go` parses,
  updates `gameInfo`, publishes `PasswordsLoaded`/`PasswordsInvalid` and (for
  now) keeps the existing log calls. Verify: `./build.sh`.

- [x] **Step 3: Save-file uploads.**
  Watcher publishes the 5 `...SaveDetected` events (incl. turn number for
  `TurnEndSaveDetected`) instead of calling `uploadFile`. New
  `handler_uploads.go` subscribes all 5 to one upload handler that wraps the
  existing `uploadFile`. `uploadExistingGameFolderFiles` publishes events for
  unsent files instead of uploading directly. Verify: `./build.sh`.

- [x] **Step 4: Games dir + game folder.**
  `GamesDirAppeared`/`GamesDirRemoved`, `GameFolderResolveRequested`/
  `Resolved`/`NotFound` events; new `handler_gamefolder.go` takes over
  `scheduleGameFolderWatch`, `resolveAndWatchGameFolder`, `switchGameFolder`,
  initial scan. Watcher's Games-folder create/remove branch only publishes.
  Verify: `./build.sh`.

- [ ] **Step 5: UI intents.**
  Browse dialog + auto-discovery publish `WatchDirChangeRequested`; flag
  buttons publish `LanguageChanged`; startup button publishes
  `StartupToggleRequested`. New `handler_watchdir.go`, `handler_language.go`,
  `handler_startup.go` take over `setWatchDir`, `setLang`,
  `handleStartupToggle` (incl. first-run startup prompt). Verify: `./build.sh`.

- [ ] **Step 6: Log projection.**
  Move every `addLog` call site into `handler_log.go` subscriptions
  (incl. mapping `UploadFailed.Kind` to existing translation keys). Business
  handlers no longer import logging. Verify: `./build.sh`.

- [ ] **Step 7: Dissolve `App`.**
  Replace `App` with `State` + `uiRefs` in `state.go`; delete the now-dead
  mutexes (`mu`, `sentFoldersMu` where safe); keep the log-store mutex and the
  upload counter/cond used by the auto-updater. Final cleanup pass.
  Verify: `./build.sh`.

- [ ] **Step 8: Docs.**
  Add a short architecture paragraph to `AGENTS.md` (bus + events + handlers
  layout). Verify: `./build.sh`.

## Verification

- `./build.sh` must pass after every step (project has no tests; build is the
  verification per `AGENTS.md`).
- After Step 7, one manual smoke test: run the app against a real H3
  directory, start/join a game, end a turn, and confirm the log panel shows
  passwords loaded + uploads, and switching language retranslates the UI
  (including the "No directory selected" label when no dir is set).

## Bugs fixed by construction (found during analysis)

- `isInitialRun` was read/written from timer goroutines without a lock
  (old `watcher.go`); single-goroutine handlers eliminate the race.
- `setLang` compared `dirLabel.Text` against the *new* language's
  "No directory selected" while the label still showed the *old* text, so the
  label never retranslated until a dir was set. The language handler will
  check `state.watchDir == ""` instead.
- Uploads could race ahead of the passwords.txt update providing their
  GameInfo headers; ordered event processing fixes it.

## Out of scope (noted, deliberately untouched)

- Tray menu labels are created once and never retranslated on language change
  (pre-existing limitation). Can be added later by rebuilding the tray menu in
  the language handler.
- `updater.go` self-update logic stays as is; only its wait-for-uploads
  counter contract is preserved via `UploadStarted`/result events.
