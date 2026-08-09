# Ajit
The application is a simple UI application written in GO + fyne.
It watches changes in player's directory of heroes of might and magic III and sends gamesave files to be parsed on the server.

The application uses a small event bus in `src/bus.go`. Producers such as the
fsnotify watcher, debounce timers, upload goroutines, and UI callbacks publish
domain events defined in `src/events.go`; handlers process them sequentially on
the bus goroutine. Business state lives in `src/state.go`, UI references stay
in `uiRefs`, and per-concern behavior is organized in the `handler_*.go` files.

First-run users see the onboarding wizard (`src/onboarding.go`): language
selection, game directory selection (auto-detected via `src/discovery.go`),
and misc settings. Nothing is persisted until the wizard finishes
(`onboarding_finished` in the config), so a restart mid-onboarding starts it
over. Existing users (config with `watch_dir`) skip it. The main view
(`src/ui.go`) swaps with the settings panel (`src/settings.go`) in the same
window; the selectable flag row with gold border lives in `src/flagselect.go`.

# build
use `./build.sh`
The program has no tests, running the build is enough of a verification.
