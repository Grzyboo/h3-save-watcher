package main

// Event is the marker interface for all domain events flowing through the bus.
// Producers (fsnotify loop, debounce timers, UI callbacks, upload goroutines)
// only Publish; handlers react to events on the single bus goroutine.
type Event interface{}

// ---------------------------------------------------------------------------
// Watcher facts (from fsnotify, after debounce — debounce stays in the producer)
// ---------------------------------------------------------------------------

// WatchStarted is emitted when the watcher (re)started on an H3 root dir.
type WatchStarted struct{ Dir string }

// WatchFailureKind classifies watcher failures; the log projection maps kinds
// to translation keys.
type WatchFailureKind int

const (
	WatchInitFailed   WatchFailureKind = iota // fsnotify init failed
	WatchAddFailed                            // adding a dir to the watcher failed
	WatchRuntimeError                         // error reported on the watcher error channel
)

// WatchFailed is emitted when fsnotify init/add fails or a runtime watcher
// error is reported.
type WatchFailed struct {
	Dir  string
	Err  error
	Kind WatchFailureKind
}

// GamesDirAppeared is emitted when <root>/Games is created at runtime.
type GamesDirAppeared struct{ Dir string }

// GamesDirRemoved is emitted when <root>/Games is deleted at runtime.
type GamesDirRemoved struct{}

// PasswordsCreated is emitted on passwords.txt create (pre-parse).
type PasswordsCreated struct{ Path string }

// PasswordsModified is emitted on passwords.txt write (pre-parse).
type PasswordsModified struct{ Path string }

// PasswordsLoaded is emitted when passwords.txt parsed successfully; it
// drives game-folder resolution + log. Changed reports whether the parsed
// game differs from the previously stored one.
type PasswordsLoaded struct {
	Info    GameInfo
	Changed bool
}

// PasswordsFailureKind classifies passwords.txt failures for the log
// projection (read vs parse error).
type PasswordsFailureKind int

const (
	PasswordsReadFailed  PasswordsFailureKind = iota
	PasswordsParseFailed
)

// PasswordsInvalid is emitted when reading/parsing passwords.txt failed;
// the stored GameInfo is cleared.
type PasswordsInvalid struct {
	Path string
	Err  error
	Kind PasswordsFailureKind
}

// BattleSaveDetected is emitted when Games/BATTLE.GM2 is written.
type BattleSaveDetected struct{ Path string }

// BattleNonContinuableSaveDetected is emitted when
// Games/BATTLE_non_continuable.GM2 is written.
type BattleNonContinuableSaveDetected struct{ Path string }

// TurnBeginSaveDetected is emitted when Games/TURN_BEGIN.GM2 is written.
type TurnBeginSaveDetected struct{ Path string }

// GameBeginSaveDetected is emitted when the game folder GAME_BEGIN.GM2 is
// written.
type GameBeginSaveDetected struct{ Path string }

// TurnEndSaveDetected is emitted when the game folder N.GM2 is written;
// Turn is parsed from the filename.
type TurnEndSaveDetected struct {
	Path string
	Turn int
}

// ---------------------------------------------------------------------------
// Game folder
// ---------------------------------------------------------------------------

// GameFolderResolveRequested is the intent to (re)resolve the current game
// folder; emitted after PasswordsLoaded, debounced.
type GameFolderResolveRequested struct{}

// GameFolderResolved is emitted when the game folder was found; the watch
// switches to it and the initial scan runs.
type GameFolderResolved struct{ Folder string }

// GameFolderNotFound is emitted when the game folder could not be found;
// the retry loop keeps ticking.
type GameFolderNotFound struct {
	Opponent string
	Err      error
}

// ---------------------------------------------------------------------------
// Uploads
// ---------------------------------------------------------------------------

// UploadStarted keeps the auto-restart wait counter working.
type UploadStarted struct{ Path string }

// UploadSucceeded is emitted when a file was uploaded; updates the
// sent-folders cache + log.
type UploadSucceeded struct{ Path string }

// UploadAlreadyOnServer is emitted when the server said already_exists;
// the file is marked as sent.
type UploadAlreadyOnServer struct{ Path string }

// UploadSkippedDuplicate is emitted when the same hash was already uploaded
// for this file type.
type UploadSkippedDuplicate struct{ Path string }

// UploadErrorKind classifies upload failures; the log projection maps kinds
// to translation keys (avoids separate failure event types per cause).
type UploadErrorKind int

const (
	UploadErrRead               UploadErrorKind = iota // os.ReadFile failed
	UploadErrConnectionRefused                         // ECONNREFUSED
	UploadErrRequest                                   // other HTTP request error
	UploadErrInvalidAnalyzeResp                        // analyze response not JSON
	UploadErrEmptyResults                              // analyze returned no results
	UploadErrServer                                    // server error string (Err carries it)
	UploadErrInvalidUploadResp                         // upload response not JSON
	UploadErrServerRejected                            // upload response ok=false
)

// UploadFailed is emitted when an upload failed; Kind carries the cause.
type UploadFailed struct {
	Path string
	Kind UploadErrorKind
	Err  error
}

// ---------------------------------------------------------------------------
// UI intents / facts
// ---------------------------------------------------------------------------

// WatchDirChangeRequested is the intent from the browse dialog /
// auto-discovery to watch a different directory.
type WatchDirChangeRequested struct{ Dir string }

// WatchDirChanged is the fact after validation + config save + watcher
// restart.
type WatchDirChanged struct{ Dir string }

// WatchDirInvalid is emitted when resolveH3Root failed; an error dialog is
// shown.
type WatchDirInvalid struct {
	Dir string
	Err error
}

// LanguageChanged is emitted when a flag is clicked; persists config,
// relabels widgets, refreshes the log list.
type LanguageChanged struct{ Lang Lang }

// StartupToggleRequested is emitted when the startup button is clicked.
type StartupToggleRequested struct{}

// StartupEnabled / StartupDisabled are facts; they relabel the button and
// show an info dialog.
type StartupEnabled struct{}
type StartupDisabled struct{}

// AppUpdated is emitted once at startup after a successful self-update.
type AppUpdated struct{ Version string }
