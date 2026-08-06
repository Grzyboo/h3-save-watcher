package main

// registerUploadHandlers subscribes the five save-detected events to the
// upload pipeline. Uploads run on their own goroutines so the event loop
// stays responsive and uploads stay concurrent.
func registerUploadHandlers(bus *Bus, a *App) {
	upload := func(path, fileType string) {
		go a.uploadFile(path, fileType)
	}
	Subscribe(bus, func(e BattleSaveDetected) { upload(e.Path, "BATTLE") })
	Subscribe(bus, func(e BattleNonContinuableSaveDetected) { upload(e.Path, "BATTLE_NC") })
	Subscribe(bus, func(e TurnBeginSaveDetected) { upload(e.Path, "TURN_BEGIN") })
	Subscribe(bus, func(e GameBeginSaveDetected) { upload(e.Path, "GAME_BEGIN") })
	Subscribe(bus, func(e TurnEndSaveDetected) { upload(e.Path, "TURN_END") })
}
