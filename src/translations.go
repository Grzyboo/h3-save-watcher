package main

type Lang string

const (
	LangEN Lang = "en"
	LangPL Lang = "pl"
	LangUA Lang = "ua"
	LangRU Lang = "ru"
)

type TranslationKey string

const (
	KeyNoDirectorySelected TranslationKey = "no_directory_selected"
	KeySelectedFolder      TranslationKey = "selected_folder"
	KeyChooseDirectory     TranslationKey = "choose_directory"
	KeyNoDirectorySet      TranslationKey = "no_directory_set"

	// Onboarding & settings
	KeySelectLanguage       TranslationKey = "select_language"
	KeyNext                 TranslationKey = "next"
	KeyPrevious             TranslationKey = "previous"
	KeyClose                TranslationKey = "close"
	KeySettings             TranslationKey = "settings"
	KeyLanguage             TranslationKey = "language"
	KeyHotAFolder           TranslationKey = "hota_folder"
	KeyChooseInstallation   TranslationKey = "choose_installation"
	KeyOtherSettings        TranslationKey = "other_settings"
	KeyAutoDetected         TranslationKey = "auto_detected"
	KeyManuallyAdded        TranslationKey = "manually_added"
	KeyAddInstallation      TranslationKey = "add_installation"
	KeyNoInstallationsFound TranslationKey = "no_installations_found"
	KeyInstallHintEmpty     TranslationKey = "installation_hint_empty"
	KeyInstallHintOne       TranslationKey = "installation_hint_one"
	KeyInstallHintMany      TranslationKey = "installation_hint_many"
	KeyAddToAutostart       TranslationKey = "add_to_autostart"
	KeyAutostartDescription TranslationKey = "autostart_description"
	KeyOther                TranslationKey = "other"

	// Log messages
	KeyLogReadError           TranslationKey = "log_read_error"
	KeyLogConnectionRefused   TranslationKey = "log_connection_refused"
	KeyLogUploadError         TranslationKey = "log_upload_error"
	KeyLogInvalidAnalyzeResp  TranslationKey = "log_invalid_analyze_resp"
	KeyLogEmptyResults        TranslationKey = "log_empty_results"
	KeyLogServerError         TranslationKey = "log_server_error"
	KeyLogInvalidUploadResp   TranslationKey = "log_invalid_upload_resp"
	KeyLogServerRejected      TranslationKey = "log_server_rejected"
	KeyLogUploaded            TranslationKey = "log_uploaded"
	KeyLogBackfillUploaded    TranslationKey = "log_backfill_uploaded"
	KeyLogWatcherInitError    TranslationKey = "log_watcher_init_error"
	KeyLogWatchError          TranslationKey = "log_watch_error"
	KeyLogWatcherError        TranslationKey = "log_watcher_error"
	KeyLogWatching            TranslationKey = "log_watching"
	KeyLogUpdated             TranslationKey = "log_updated"
	KeyLogPasswordsReadError  TranslationKey = "log_passwords_read_error"
	KeyLogPasswordsParseError TranslationKey = "log_passwords_parse_error"
	KeyLogNewGameDetected     TranslationKey = "log_new_game_detected"
	KeyLogWatchingWaiting     TranslationKey = "log_watching_waiting"

	// Startup
	KeyStartupError     TranslationKey = "startup_error"
	KeyStartupWarnTitle TranslationKey = "startup_warn_title"
	KeyStartupWarnMsg   TranslationKey = "startup_warn_msg"

	// Tray
	KeyTrayShow TranslationKey = "tray_show"
	KeyTrayQuit TranslationKey = "tray_quit"
)

var translations = map[Lang]map[TranslationKey]string{
	LangEN: {
		KeyNoDirectorySelected: "No directory selected",
		KeySelectedFolder:      "Selected folder",
		KeyChooseDirectory:     "Choose Directory",
		KeyNoDirectorySet:      "No watch directory set.",

		KeySelectLanguage:       "Select your language:",
		KeyNext:                 "Next >",
		KeyPrevious:             "< Previous",
		KeyClose:                "Close",
		KeySettings:             "Settings",
		KeyLanguage:             "Language",
		KeyHotAFolder:           "HotA folder",
		KeyChooseInstallation:   "Choose Horn of the Abyss installation",
		KeyOtherSettings:        "Other settings",
		KeyAutoDetected:         "[autodetected] ",
		KeyManuallyAdded:        "[added manually] ",
		KeyAddInstallation:      "Add HotA installation",
		KeyNoInstallationsFound: "Could not automatically find HOMM3:HotA installed on the system - please add HotA folder.\nThe application will reject any non-hota folder.",
		KeyInstallHintEmpty:     "add a hota installation folder to continue",
		KeyInstallHintOne:       "select the installation above the button or add another one",
		KeyInstallHintMany:      "select one of the installations from the list above to continue",
		KeyAddToAutostart:       "Add to autostart",
		KeyAutostartDescription: "if checked, The application will run on system startup",
		KeyOther:                "Other",

		KeyLogReadError:           "read %s: %v",
		KeyLogConnectionRefused:   "upload %s: connection refused",
		KeyLogUploadError:         "upload %s: %v",
		KeyLogInvalidAnalyzeResp:  "upload %s: invalid analyze response",
		KeyLogEmptyResults:        "upload %s: empty results from server",
		KeyLogServerError:         "upload %s: %s",
		KeyLogInvalidUploadResp:   "upload %s: invalid upload response",
		KeyLogServerRejected:      "upload %s: server rejected file",
		KeyLogUploaded:            "uploaded %s",
		KeyLogBackfillUploaded:    "[backfill] uploaded %s",
		KeyLogWatcherInitError:    "watcher init error: %v",
		KeyLogWatchError:          "watch error: %v",
		KeyLogWatcherError:        "watcher error: %v",
		KeyLogWatching:            "configuration correct\nYou can start playing the game, Ajit will detect HotA saves",
		KeyLogUpdated:             "The application was updated to version: %s",
		KeyLogPasswordsReadError:  "passwords.txt: read error: %v",
		KeyLogPasswordsParseError: "passwords.txt: parse error: %v",
		KeyLogNewGameDetected:     "detected new game: %s vs %s",
		KeyLogWatchingWaiting:     "Could not read the latest game. File 'passwords.txt' may not be present or the file is broken",

		KeyStartupError:     "Startup registration failed: %v",
		KeyStartupWarnTitle: "Warning",
		KeyStartupWarnMsg:   "The application is running from a temporary or build path:\n%s\n\nStartup entry may not work after moving the binary. Continue?",
		KeyTrayShow:         "Show",
		KeyTrayQuit:         "Quit",
	},
	LangPL: {
		KeyNoDirectorySelected: "Nie wybrano katalogu",
		KeySelectedFolder:      "Wybrany folder",
		KeyChooseDirectory:     "Wybierz katalog",
		KeyNoDirectorySet:      "Nie ustawiono katalogu.",

		KeySelectLanguage:       "Wybierz język:",
		KeyNext:                 "Dalej >",
		KeyPrevious:             "< Wstecz",
		KeyClose:                "Zamknij",
		KeySettings:             "Ustawienia",
		KeyLanguage:             "Język",
		KeyHotAFolder:           "Folder HotA",
		KeyChooseInstallation:   "Wybierz instalację Horn of the Abyss",
		KeyOtherSettings:        "Inne ustawienia",
		KeyAutoDetected:         "[wykryto automatycznie] ",
		KeyManuallyAdded:        "[dodano ręcznie] ",
		KeyAddInstallation:      "Dodaj instalację HotA",
		KeyNoInstallationsFound: "Nie udało się automatycznie znaleźć zainstalowanego HOMM3:HotA w systemie - dodaj folder HotA.\nAplikacja odrzuci każdy folder, który nie jest folderem HotA.",
		KeyInstallHintEmpty:     "dodaj folder instalacji HotA, aby kontynuować",
		KeyInstallHintOne:       "wybierz instalację powyżej przycisku lub dodaj kolejną",
		KeyInstallHintMany:      "wybierz jedną z instalacji z listy powyżej, aby kontynuować",
		KeyAddToAutostart:       "Dodaj do autostartu",
		KeyAutostartDescription: "jeśli zaznaczone, aplikacja będzie uruchamiana przy starcie systemu",
		KeyOther:                "Inne",

		KeyLogReadError:           "odczyt %s: %v",
		KeyLogConnectionRefused:   "przesyłanie %s: odmowa połączenia",
		KeyLogUploadError:         "przesyłanie %s: %v",
		KeyLogInvalidAnalyzeResp:  "przesyłanie %s: nieprawidłowa odpowiedź analizy",
		KeyLogEmptyResults:        "przesyłanie %s: brak wyników z serwera",
		KeyLogServerError:         "przesyłanie %s: %s",
		KeyLogInvalidUploadResp:   "przesyłanie %s: nieprawidłowa odpowiedź serwera",
		KeyLogServerRejected:      "przesyłanie %s: serwer odrzucił plik",
		KeyLogUploaded:            "przesłano %s",
		KeyLogBackfillUploaded:    "[backfill] przesłano %s",
		KeyLogWatcherInitError:    "błąd inicjalizacji obserwatora: %v",
		KeyLogWatchError:          "błąd obserwowania: %v",
		KeyLogWatcherError:        "błąd obserwatora: %v",
		KeyLogWatching:            "konfiguracja poprawna\nMożesz rozpocząć grę, Ajit wykryje zapisy HotA",
		KeyLogUpdated:             "Aplikacja została zaktualizowana do wersji: %s",
		KeyLogPasswordsReadError:  "passwords.txt: błąd odczytu: %v",
		KeyLogPasswordsParseError: "passwords.txt: błąd parsowania: %v",
		KeyLogNewGameDetected:     "wykryto nową grę: %s vs %s",
		KeyLogWatchingWaiting:     "Nie udało się odczytać ostatniej gry. Plik 'passwords.txt' może nie istnieć lub być uszkodzony",

		KeyStartupError:     "Błąd rejestracji autostartu: %v",
		KeyStartupWarnTitle: "Uwaga",
		KeyStartupWarnMsg:   "Aplikacja działa ze ścieżki tymczasowej lub buildowej:\n%s\n\nAutostart może nie działać po przeniesieniu pliku. Kontynuować?",
		KeyTrayShow:         "Pokaż",
		KeyTrayQuit:         "Zamknij",
	},
	LangUA: {
		KeyNoDirectorySelected: "Каталог не вибрано",
		KeySelectedFolder:      "Вибрана папка",
		KeyChooseDirectory:     "Вибрати каталог",
		KeyNoDirectorySet:      "Каталог не встановлено.",

		KeySelectLanguage:       "Оберіть мову:",
		KeyNext:                 "Далі >",
		KeyPrevious:             "< Назад",
		KeyClose:                "Закрити",
		KeySettings:             "Налаштування",
		KeyLanguage:             "Мова",
		KeyHotAFolder:           "Папка HotA",
		KeyChooseInstallation:   "Оберіть встановлення Horn of the Abyss",
		KeyOtherSettings:        "Інші налаштування",
		KeyAutoDetected:         "[виявлено автоматично] ",
		KeyManuallyAdded:        "[додано вручну] ",
		KeyAddInstallation:      "Додати встановлення HotA",
		KeyNoInstallationsFound: "Не вдалося автоматично знайти встановлену HOMM3:HotA у системі - додайте папку HotA.\nПрограма відхилить будь-яку папку, яка не є папкою HotA.",
		KeyInstallHintEmpty:     "додайте папку встановлення HotA, щоб продовжити",
		KeyInstallHintOne:       "виберіть встановлення над кнопкою або додайте ще одне",
		KeyInstallHintMany:      "виберіть одне зі встановлень зі списку вище, щоб продовжити",
		KeyAddToAutostart:       "Додати до автозапуску",
		KeyAutostartDescription: "якщо позначено, програма запускатиметься під час запуску системи",
		KeyOther:                "Інше",

		KeyLogReadError:           "читання %s: %v",
		KeyLogConnectionRefused:   "завантаження %s: з'єднання відхилено",
		KeyLogUploadError:         "завантаження %s: %v",
		KeyLogInvalidAnalyzeResp:  "завантаження %s: невірна відповідь аналізу",
		KeyLogEmptyResults:        "завантаження %s: порожні результати від сервера",
		KeyLogServerError:         "завантаження %s: %s",
		KeyLogInvalidUploadResp:   "завантаження %s: невірна відповідь завантаження",
		KeyLogServerRejected:      "завантаження %s: сервер відхилив файл",
		KeyLogUploaded:            "завантажено %s",
		KeyLogBackfillUploaded:    "[backfill] завантажено %s",
		KeyLogWatcherInitError:    "помилка ініціалізації спостерігача: %v",
		KeyLogWatchError:          "помилка спостереження: %v",
		KeyLogWatcherError:        "помилка спостерігача: %v",
		KeyLogWatching:            "конфігурація правильна\nМожете почати грати, Ajit виявлятиме збереження HotA",
		KeyLogUpdated:             "Програму оновлено до версії: %s",
		KeyLogPasswordsReadError:  "passwords.txt: помилка читання: %v",
		KeyLogPasswordsParseError: "passwords.txt: помилка парсингу: %v",
		KeyLogNewGameDetected:     "виявлено нову гру: %s проти %s",
		KeyLogWatchingWaiting:     "Не вдалося прочитати останню гру. Файл 'passwords.txt' може бути відсутній або пошкоджений",

		KeyStartupError:     "Помилка реєстрації автозапуску: %v",
		KeyStartupWarnTitle: "Попередження",
		KeyStartupWarnMsg:   "Програма запущена з тимчасового або збірного шляху:\n%s\n\nАвтозапуск може не працювати після переміщення файлу. Продовжити?",
		KeyTrayShow:         "Показати",
		KeyTrayQuit:         "Вийти",
	},
	LangRU: {
		KeyNoDirectorySelected: "Каталог не выбран",
		KeySelectedFolder:      "Выбранная папка",
		KeyChooseDirectory:     "Выбрать каталог",
		KeyNoDirectorySet:      "Каталог не установлен.",

		KeySelectLanguage:       "Выберите язык:",
		KeyNext:                 "Далее >",
		KeyPrevious:             "< Назад",
		KeyClose:                "Закрыть",
		KeySettings:             "Настройки",
		KeyLanguage:             "Язык",
		KeyHotAFolder:           "Папка HotA",
		KeyChooseInstallation:   "Выберите установку Horn of the Abyss",
		KeyOtherSettings:        "Другие настройки",
		KeyAutoDetected:         "[обнаружено автоматически] ",
		KeyManuallyAdded:        "[добавлено вручную] ",
		KeyAddInstallation:      "Добавить установку HotA",
		KeyNoInstallationsFound: "Не удалось автоматически найти установленную HOMM3:HotA в системе - добавьте папку HotA.\nПриложение отклонит любую папку, которая не является папкой HotA.",
		KeyInstallHintEmpty:     "добавьте папку установки HotA, чтобы продолжить",
		KeyInstallHintOne:       "выберите установку над кнопкой или добавьте еще одну",
		KeyInstallHintMany:      "выберите одну из установок в списке выше, чтобы продолжить",
		KeyAddToAutostart:       "Добавить в автозапуск",
		KeyAutostartDescription: "если флажок установлен, приложение будет запускаться при старте системы",
		KeyOther:                "Прочее",

		KeyLogReadError:           "чтение %s: %v",
		KeyLogConnectionRefused:   "загрузка %s: соединение отклонено",
		KeyLogUploadError:         "загрузка %s: %v",
		KeyLogInvalidAnalyzeResp:  "загрузка %s: неверный ответ анализа",
		KeyLogEmptyResults:        "загрузка %s: пустые результаты от сервера",
		KeyLogServerError:         "загрузка %s: %s",
		KeyLogInvalidUploadResp:   "загрузка %s: неверный ответ загрузки",
		KeyLogServerRejected:      "загрузка %s: сервер отклонил файл",
		KeyLogUploaded:            "загружено %s",
		KeyLogBackfillUploaded:    "[backfill] загружено %s",
		KeyLogWatcherInitError:    "ошибка инициализации наблюдателя: %v",
		KeyLogWatchError:          "ошибка наблюдения: %v",
		KeyLogWatcherError:        "ошибка наблюдателя: %v",
		KeyLogWatching:            "конфигурация верна\nВы можете начать играть, Ajit будет обнаруживать сохранения HotA",
		KeyLogUpdated:             "Приложение обновлено до версии: %s",
		KeyLogPasswordsReadError:  "passwords.txt: ошибка чтения: %v",
		KeyLogPasswordsParseError: "passwords.txt: ошибка разбора: %v",
		KeyLogNewGameDetected:     "обнаружена новая игра: %s против %s",
		KeyLogWatchingWaiting:     "Не удалось прочитать последнюю игру. Файл 'passwords.txt' может отсутствовать или быть поврежден",

		KeyStartupError:     "Ошибка регистрации автозапуска: %v",
		KeyStartupWarnTitle: "Предупреждение",
		KeyStartupWarnMsg:   "Приложение запущено из временного или сборочного пути:\n%s\n\nАвтозапуск может не работать после перемещения файла. Продолжить?",
		KeyTrayShow:         "Показать",
		KeyTrayQuit:         "Выйти",
	},
}
