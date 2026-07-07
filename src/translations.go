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
	KeyNoDirectorySelected     TranslationKey = "no_directory_selected"
	KeySelectedFolder          TranslationKey = "selected_folder"
	KeyChooseDirectory         TranslationKey = "choose_directory"
	KeyInstallationFound       TranslationKey = "installation_found"
	KeyInstallationFoundMsg    TranslationKey = "installation_found_msg"
	KeyInstallationNotFound    TranslationKey = "installation_not_found"
	KeyInstallationNotFoundMsg TranslationKey = "installation_not_found_msg"
	KeyNoDirectorySet          TranslationKey = "no_directory_set"

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
	KeyLogWatcherInitError    TranslationKey = "log_watcher_init_error"
	KeyLogWatchError          TranslationKey = "log_watch_error"
	KeyLogWatcherError        TranslationKey = "log_watcher_error"
	KeyLogWatching            TranslationKey = "log_watching"
	KeyLogUpdated             TranslationKey = "log_updated"
	KeyLogPasswordsReadError  TranslationKey = "log_passwords_read_error"
	KeyLogPasswordsParseError TranslationKey = "log_passwords_parse_error"

	// Startup
	KeyStartupEnable         TranslationKey = "startup_enable"
	KeyStartupDisable        TranslationKey = "startup_disable"
	KeyStartupEnableConfirm  TranslationKey = "startup_enable_confirm"
	KeyStartupDisableConfirm TranslationKey = "startup_disable_confirm"
	KeyStartupEnableTitle    TranslationKey = "startup_enable_title"
	KeyStartupDisableTitle   TranslationKey = "startup_disable_title"
	KeyStartupSuccess        TranslationKey = "startup_success"
	KeyStartupRemoved        TranslationKey = "startup_removed"
	KeyStartupError          TranslationKey = "startup_error"
	KeyStartupWarnTitle      TranslationKey = "startup_warn_title"
	KeyStartupWarnMsg        TranslationKey = "startup_warn_msg"

	// Tray
	KeyTrayShow TranslationKey = "tray_show"
	KeyTrayQuit TranslationKey = "tray_quit"
)

var translations = map[Lang]map[TranslationKey]string{
	LangEN: {
		KeyNoDirectorySelected:     "No directory selected",
		KeySelectedFolder:          "Selected folder",
		KeyChooseDirectory:         "Choose Directory",
		KeyInstallationFound:       "Installation found",
		KeyInstallationFoundMsg:    "Found H3 installation at:\n%s\n\nStart watching this directory?",
		KeyInstallationNotFound:    "Installation not found",
		KeyInstallationNotFoundMsg: "Could not automatically locate a Heroes of Might and Magic III installation.\n\nPlease use the \"Choose Directory\" button to select your installation folder.",
		KeyNoDirectorySet:          "No watch directory set.",
		KeyLogReadError:            "read %s: %v",
		KeyLogConnectionRefused:    "upload %s: connection refused",
		KeyLogUploadError:          "upload %s: %v",
		KeyLogInvalidAnalyzeResp:   "upload %s: invalid analyze response",
		KeyLogEmptyResults:         "upload %s: empty results from server",
		KeyLogServerError:          "upload %s: %s",
		KeyLogInvalidUploadResp:    "upload %s: invalid upload response",
		KeyLogServerRejected:       "upload %s: server rejected file",
		KeyLogUploaded:             "uploaded %s",
		KeyLogWatcherInitError:     "watcher init error: %v",
		KeyLogWatchError:           "watch error: %v",
		KeyLogWatcherError:         "watcher error: %v",
		KeyLogWatching:             "watching BATTLE save files",
		KeyLogUpdated:              "The application was updated to version: %s",
		KeyLogPasswordsReadError:   "passwords.txt: read error: %v",
		KeyLogPasswordsParseError:  "passwords.txt: parse error: %v",

		KeyStartupEnable:         "Autostart: off",
		KeyStartupDisable:        "Autostart: on",
		KeyStartupEnableTitle:    "Enable startup",
		KeyStartupDisableTitle:   "Disable startup",
		KeyStartupEnableConfirm:  "Add H3SaveWatcher to system startup?\nIt will launch automatically when you log in.",
		KeyStartupDisableConfirm: "Remove H3SaveWatcher from system startup?",
		KeyStartupSuccess:        "Added to system startup.",
		KeyStartupRemoved:        "Removed from system startup.",
		KeyStartupError:          "Startup registration failed: %v",
		KeyStartupWarnTitle:      "Warning",
		KeyStartupWarnMsg:        "The application is running from a temporary or build path:\n%s\n\nStartup entry may not work after moving the binary. Continue?",
		KeyTrayShow:              "Show",
		KeyTrayQuit:              "Quit",
	},
	LangPL: {
		KeyNoDirectorySelected:     "Nie wybrano katalogu",
		KeySelectedFolder:          "Wybrany folder",
		KeyChooseDirectory:         "Wybierz katalog",
		KeyInstallationFound:       "Znaleziono instalację",
		KeyInstallationFoundMsg:    "Znaleziono instalację H3 w:\n%s\n\nCzy chcesz obserwować ten katalog?",
		KeyInstallationNotFound:    "Nie znaleziono instalacji",
		KeyInstallationNotFoundMsg: "Nie udało się automatycznie zlokalizować instalacji Heroes of Might and Magic III.\n\nUżyj przycisku \"Wybierz katalog\", aby wskazać folder instalacji.",
		KeyNoDirectorySet:          "Nie ustawiono katalogu.",
		KeyLogReadError:            "odczyt %s: %v",
		KeyLogConnectionRefused:    "przesyłanie %s: odmowa połączenia",
		KeyLogUploadError:          "przesyłanie %s: %v",
		KeyLogInvalidAnalyzeResp:   "przesyłanie %s: nieprawidłowa odpowiedź analizy",
		KeyLogEmptyResults:         "przesyłanie %s: brak wyników z serwera",
		KeyLogServerError:          "przesyłanie %s: %s",
		KeyLogInvalidUploadResp:    "przesyłanie %s: nieprawidłowa odpowiedź serwera",
		KeyLogServerRejected:       "przesyłanie %s: serwer odrzucił plik",
		KeyLogUploaded:             "przesłano %s",
		KeyLogWatcherInitError:     "błąd inicjalizacji obserwatora: %v",
		KeyLogWatchError:           "błąd obserwowania: %v",
		KeyLogWatcherError:         "błąd obserwatora: %v",
		KeyLogWatching:             "obserwowanie plików zapisu BATTLE",
		KeyLogUpdated:              "Aplikacja została zaktualizowana do wersji: %s",
		KeyLogPasswordsReadError:   "passwords.txt: błąd odczytu: %v",
		KeyLogPasswordsParseError:  "passwords.txt: błąd parsowania: %v",

		KeyStartupEnable:         "Autostart: wył.",
		KeyStartupDisable:        "Autostart: wł.",
		KeyStartupEnableTitle:    "Włącz autostart",
		KeyStartupDisableTitle:   "Wyłącz autostart",
		KeyStartupEnableConfirm:  "Dodać H3SaveWatcher do autostartu?\nAplikacja będzie uruchamiana automatycznie po zalogowaniu.",
		KeyStartupDisableConfirm: "Usunąć H3SaveWatcher z autostartu?",
		KeyStartupSuccess:        "Dodano do autostartu.",
		KeyStartupRemoved:        "Usunięto z autostartu.",
		KeyStartupError:          "Błąd rejestracji autostartu: %v",
		KeyStartupWarnTitle:      "Uwaga",
		KeyStartupWarnMsg:        "Aplikacja działa ze ścieżki tymczasowej lub buildowej:\n%s\n\nAutostart może nie działać po przeniesieniu pliku. Kontynuować?",
		KeyTrayShow:              "Pokaż",
		KeyTrayQuit:              "Zamknij",
	},
	LangUA: {
		KeyNoDirectorySelected:     "Каталог не вибрано",
		KeySelectedFolder:          "Вибрана папка",
		KeyChooseDirectory:         "Вибрати каталог",
		KeyInstallationFound:       "Знайдено встановлення",
		KeyInstallationFoundMsg:    "Знайдено встановлення H3 за адресою:\n%s\n\nПочати спостереження за цим каталогом?",
		KeyInstallationNotFound:    "Встановлення не знайдено",
		KeyInstallationNotFoundMsg: "Не вдалося автоматично знайти встановлення Heroes of Might and Magic III.\n\nБудь ласка, використайте кнопку \"Вибрати каталог\" для вибору папки встановлення.",
		KeyNoDirectorySet:          "Каталог не встановлено.",
		KeyLogReadError:            "читання %s: %v",
		KeyLogConnectionRefused:    "завантаження %s: з'єднання відхилено",
		KeyLogUploadError:          "завантаження %s: %v",
		KeyLogInvalidAnalyzeResp:   "завантаження %s: невірна відповідь аналізу",
		KeyLogEmptyResults:         "завантаження %s: порожні результати від сервера",
		KeyLogServerError:          "завантаження %s: %s",
		KeyLogInvalidUploadResp:    "завантаження %s: невірна відповідь завантаження",
		KeyLogServerRejected:       "завантаження %s: сервер відхилив файл",
		KeyLogUploaded:             "завантажено %s",
		KeyLogWatcherInitError:     "помилка ініціалізації спостерігача: %v",
		KeyLogWatchError:           "помилка спостереження: %v",
		KeyLogWatcherError:         "помилка спостерігача: %v",
		KeyLogWatching:             "спостереження за файлами збереження BATTLE",
		KeyLogUpdated:              "Програму оновлено до версії: %s",
		KeyLogPasswordsReadError:   "passwords.txt: помилка читання: %v",
		KeyLogPasswordsParseError:  "passwords.txt: помилка парсингу: %v",

		KeyStartupEnable:         "Автозапуск: вимк.",
		KeyStartupDisable:        "Автозапуск: увімк.",
		KeyStartupEnableTitle:    "Увімкнути автозапуск",
		KeyStartupDisableTitle:   "Вимкнути автозапуск",
		KeyStartupEnableConfirm:  "Додати H3SaveWatcher до автозапуску?\nПрограма запускатиметься автоматично при вході.",
		KeyStartupDisableConfirm: "Видалити H3SaveWatcher з автозапуску?",
		KeyStartupSuccess:        "Додано до автозапуску.",
		KeyStartupRemoved:        "Видалено з автозапуску.",
		KeyStartupError:          "Помилка реєстрації автозапуску: %v",
		KeyStartupWarnTitle:      "Попередження",
		KeyStartupWarnMsg:        "Програма запущена з тимчасового або збірного шляху:\n%s\n\nАвтозапуск може не працювати після переміщення файлу. Продовжити?",
		KeyTrayShow:              "Показати",
		KeyTrayQuit:              "Вийти",
	},
	LangRU: {
		KeyNoDirectorySelected:     "Каталог не выбран",
		KeySelectedFolder:          "Выбранная папка",
		KeyChooseDirectory:         "Выбрать каталог",
		KeyInstallationFound:       "Установка найдена",
		KeyInstallationFoundMsg:    "Найдена установка H3 по адресу:\n%s\n\nНачать наблюдение за этим каталогом?",
		KeyInstallationNotFound:    "Установка не найдена",
		KeyInstallationNotFoundMsg: "Не удалось автоматически найти установку Heroes of Might and Magic III.\n\nПожалуйста, используйте кнопку \"Выбрать каталог\" для выбора папки установки.",
		KeyNoDirectorySet:          "Каталог не установлен.",
		KeyLogReadError:            "чтение %s: %v",
		KeyLogConnectionRefused:    "загрузка %s: соединение отклонено",
		KeyLogUploadError:          "загрузка %s: %v",
		KeyLogInvalidAnalyzeResp:   "загрузка %s: неверный ответ анализа",
		KeyLogEmptyResults:         "загрузка %s: пустые результаты от сервера",
		KeyLogServerError:          "загрузка %s: %s",
		KeyLogInvalidUploadResp:    "загрузка %s: неверный ответ загрузки",
		KeyLogServerRejected:       "загрузка %s: сервер отклонил файл",
		KeyLogUploaded:             "загружено %s",
		KeyLogWatcherInitError:     "ошибка инициализации наблюдателя: %v",
		KeyLogWatchError:           "ошибка наблюдения: %v",
		KeyLogWatcherError:         "ошибка наблюдателя: %v",
		KeyLogWatching:             "наблюдение за файлами сохранения BATTLE",
		KeyLogUpdated:              "Приложение обновлено до версии: %s",
		KeyLogPasswordsReadError:   "passwords.txt: ошибка чтения: %v",
		KeyLogPasswordsParseError:  "passwords.txt: ошибка разбора: %v",

		KeyStartupEnable:         "Автозапуск: выкл.",
		KeyStartupDisable:        "Автозапуск: вкл.",
		KeyStartupEnableTitle:    "Включить автозапуск",
		KeyStartupDisableTitle:   "Отключить автозапуск",
		KeyStartupEnableConfirm:  "Добавить H3SaveWatcher в автозапуск?\nПриложение будет запускаться автоматически при входе.",
		KeyStartupDisableConfirm: "Удалить H3SaveWatcher из автозапуска?",
		KeyStartupSuccess:        "Добавлено в автозапуск.",
		KeyStartupRemoved:        "Удалено из автозапуска.",
		KeyStartupError:          "Ошибка регистрации автозапуска: %v",
		KeyStartupWarnTitle:      "Предупреждение",
		KeyStartupWarnMsg:        "Приложение запущено из временного или сборочного пути:\n%s\n\nАвтозапуск может не работать после перемещения файла. Продолжить?",
		KeyTrayShow:              "Показать",
		KeyTrayQuit:              "Выйти",
	},
}

func (a *App) T(key TranslationKey) string {
	a.mu.Lock()
	lang := a.lang
	a.mu.Unlock()
	if t, ok := translations[lang]; ok {
		if s, ok := t[key]; ok {
			return s
		}
	}
	// fallback to English
	if s, ok := translations[LangEN][key]; ok {
		return s
	}
	return string(key)
}
