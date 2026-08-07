package main

import "path/filepath"

// registerLogHandlers subscribes the log projection to fact events: every UI
// log entry is produced here; business handlers never call logStore directly.
func registerLogHandlers(bus *Bus, s *State, logs *logStore) {
	logf := func(success bool, key TranslationKey, args ...any) {
		logs.add(success, key, args...)
	}

	// WatchStarted is published right after PasswordsCreated, so by the time
	// this handler runs gameInfo reflects the freshly parsed file.
	Subscribe(bus, func(e WatchStarted) {
		if s.gameInfo.OpponentName != "" {
			logf(true, KeyLogWatching)
		} else {
			logf(true, KeyLogWatchingWaiting)
		}
	})

	Subscribe(bus, func(e WatchFailed) {
		switch e.Kind {
		case WatchInitFailed:
			logf(false, KeyLogWatcherInitError, e.Err)
		case WatchAddFailed:
			logf(false, KeyLogWatchError, e.Err)
		case WatchRuntimeError:
			logf(false, KeyLogWatcherError, e.Err)
		}
	})

	Subscribe(bus, func(e PasswordsLoaded) {
		if e.Changed {
			logf(true, KeyLogPasswordsLoaded, e.Info.PlayerName, e.Info.OpponentName)
		}
	})

	Subscribe(bus, func(e PasswordsInvalid) {
		if e.Kind == PasswordsReadFailed {
			logf(false, KeyLogPasswordsReadError, e.Err)
		} else {
			logf(false, KeyLogPasswordsParseError, e.Err)
		}
	})

	Subscribe(bus, func(e UploadSucceeded) {
		logf(true, KeyLogUploaded, filepath.Base(e.Path))
	})

	Subscribe(bus, func(e UploadFailed) {
		base := filepath.Base(e.Path)
		switch e.Kind {
		case UploadErrRead:
			logf(false, KeyLogReadError, base, e.Err)
		case UploadErrConnectionRefused:
			logf(false, KeyLogConnectionRefused, base)
		case UploadErrRequest:
			logf(false, KeyLogUploadError, base, e.Err)
		case UploadErrInvalidAnalyzeResp:
			logf(false, KeyLogInvalidAnalyzeResp, base)
		case UploadErrEmptyResults:
			logf(false, KeyLogEmptyResults, base)
		case UploadErrServer:
			logf(false, KeyLogServerError, base, e.Err)
		case UploadErrInvalidUploadResp:
			logf(false, KeyLogInvalidUploadResp, base)
		case UploadErrServerRejected:
			logf(false, KeyLogServerRejected, base)
		}
	})

	Subscribe(bus, func(e AppUpdated) {
		logf(true, KeyLogUpdated, e.Version)
	})
}
