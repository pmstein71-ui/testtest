# Передача проекта Pulse

Дата среза: 2026-11-23 (UTC). Документ предназначен агенту без истории переписки.

## 1. Требования и текущее состояние

Исходный запрос: программа для Windows с `setup.exe`, Go + HTML GUI, клиент-серверная архитектура, серверная база данных, резервное копирование по расписанию, вход по логину/паролю, немедленная сборка через GitHub Actions после изменений и обновления из GitHub при выпуске версии.

Сделано: рабочий стартовый код сервера и клиента, SQLite, авторизация, интервальные снимки базы, HTML UI, установщик Inno Setup и CI. Предметная область не задана: нет товаров, документов и другой бизнес-логики. Название Pulse выбрано при реализации, не является отдельным требованием пользователя.

Важные границы реализации:
- HTML UI открывается в браузере по умолчанию; нативного окна/WebView нет.
- Сервер включён в установщик, но запускается отдельно; службы и мастера настройки нет.
- Клиент работает через API, прямого сетевого доступа к SQLite нет.
- Backup — через интервал Go duration, не календарный cron.
- Проверка обновлений автоматическая; скачивание/установка требуют подтверждения.
- Публичный релиз в рамках выполненной работы не создавался. Известен успешный build artifact, не проверенный выпуск Release.

## 2. Карта репозитория

| Путь | Назначение |
|---|---|
| `cmd/server/main.go` | Конфигурация, SQLite schema, bootstrap admin, API, backup timer, TLS/HTTP, shutdown |
| `cmd/client/main.go` | Локальный HTTP UI, reverse proxy к серверу, update/install endpoints, открытие браузера |
| `internal/web/embed.go` | Встраивание HTML через go:embed |
| `internal/web/index.html` | Самодостаточный HTML/CSS/JS: логин, профиль, выход, проверка и установка обновления |
| `internal/release/release.go` | GitHub latest Release, загрузка assets, SHA-256, временный setup.exe |
| `internal/release/release_test.go` | Unit tests условий доступности обновления и запрета исходных недоверенных URL |
| `installer/setup.iss` | Inno Setup: per-user Windows x64 установка клиента, сервера и README |
| `.github/workflows/build.yml` | Linux vet/test, Windows EXE + setup.exe + checksums + artifact, tag Release |
| `go.mod` | Go 1.23.0, bcrypt x/crypto v0.31.0, modernc SQLite v1.34.5 |
| `.gitignore` | Исключение dist, data, EXE и SQLite-файлов |
| `README.md` | Пользовательские инструкции |
| `AGENTS.md` | Короткая точка входа и правила для агента |

## 3. Поток данных и безопасность

```text
Браузер (HTML, токен в памяти JS)
  -> http://127.0.0.1:<случайный порт> (Go client)
      /api/* -> reverse proxy -> HTTPS Go server -> SQLite
      /local/update -> GitHub Releases API
      /local/install -> GitHub assets -> SHA-256 -> запуск setup.exe
```

Клиент слушает только loopback. Host должен точно совпадать с адресом listener; непустой Origin должен совпадать с origin клиента. `/api/` и `/local/` требуют `X-Pulse-Client: 1`. Это защита браузерной границы, не аутентификация от других локальных процессов. Не включать CORS `*`.

Удалённый SERVER_URL требует HTTPS с обычной проверкой сертификата. HTTP разрешён лишь для localhost/127.0.0.1/::1. Клиент не должен отключать проверку TLS.

Пароли — bcrypt. Токен — 32 случайных байта, hex, в БД SHA-256 от его текстового представления. Срок сеанса — 8 часов. Клиентский токен не сохраняется в localStorage; перезагрузка страницы требует нового входа. Сервер валидирует expires при каждом `/api/me`, выход удаляет сеанс. Очистка просроченных записей выполняется с backup timer. Есть глобальный лимит попыток логина (1/сек), не per-user/per-IP.

## 4. Конфигурация

Все настройки — переменные окружения процесса, UI конфигурации и .env loader нет.

| Переменная | По умолчанию / поведение |
|---|---|
| `ADMIN_LOGIN` | `admin`, только создание первого пользователя |
| `ADMIN_PASSWORD` | Обязателен на пустой БД, минимум 12 байт; bcrypt имеет ограничение 72 байта |
| `DATA_DIR` | `data`, относительно текущего рабочего каталога процесса |
| `BACKUP_DIR` | `DATA_DIR/backups` |
| `BACKUP_INTERVAL` | `24h`, минимум `1m` |
| `LISTEN_ADDR` | `127.0.0.1:8080` |
| `TLS_CERT`, `TLS_KEY` | При наличии обоих включается ListenAndServeTLS; иначе HTTP |
| `SERVER_URL` | У клиента `http://127.0.0.1:8080` |

Для установки задавайте абсолютный DATA_DIR, например `%LOCALAPPDATA%\PulseServer\data`. Иначе расположение базы зависит от способа запуска. Изменение ADMIN_PASSWORD после первого запуска не меняет пароль существующего пользователя.

## 5. База и backup

SQLite `app.db`, WAL, busy_timeout=5000, пул ограничен одним соединением.

```sql
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  login TEXT UNIQUE NOT NULL,
  password BLOB NOT NULL
);
CREATE TABLE sessions (
  token TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL,
  expires INTEGER NOT NULL
);
```

Миграционного фреймворка и FK constraint нет. Таблицы создаются IF NOT EXISTS. Не изменять схему разрушительно без миграции и восстановления из backup.

Снимки через `VACUUM INTO`, имя `backup-<UTC timestamp с наносекундами>.db`. Первый backup запускается горутиной при старте, затем ticker. Ошибка журналируется, сервер продолжает работу. Retention, шифрование, offsite и проверка свободного места отсутствуют. VACUUM занимает единственное соединение и может задерживать API.

Восстановление подробно в README: остановить сервер, сохранить/убрать app.db и WAL/SHM, скопировать snapshot как app.db, запустить. Копия содержит хеши паролей и сеансы. При необходимости отозвать восстановленные сеансы — очистить sessions. Никогда не восстанавливать поверх работающей БД.

## 6. API

У серверных ответов Cache-Control: no-store. Ошибки в основном plain text. Клиент проксирует `/api/*`; браузер добавляет X-Pulse-Client.

| Метод / путь | Запрос | Успешный ответ |
|---|---|---|
| `POST /api/login` | JSON `{"login":"…","password":"…"}`, тело до 4096 байт | 200 JSON `{"token":"…"}` |
| `GET /api/me` | `Authorization: Bearer <token>` | 200 JSON `{"login":"…","message":"…"}` |
| `POST /api/logout` | Bearer token | 204, удаляет сеанс |
| `GET /local/update` | Только локальный клиент | JSON current, latest, available |
| `POST /local/install` | Только локальный клиент, Windows | JSON `{"started":true}` после запуска EXE, не после завершения установки |

Логин: 400 при ошибке JSON, 401 неверные данные, 429 частые попытки. `/me`: 401 при отсутствии/истечении сеанса. Update ошибки GitHub превращаются в 502, install ошибки — в 500. `/local/*` отсутствуют на сервере БД.

## 7. Сборка, установка, публикация

Локально с Go и доступом к module proxy:

```sh
go mod tidy
gofmt -w cmd internal
go vet ./...
go test -race ./...
go build -o pulse-client.exe ./cmd/client
go build -o pulse-server.exe ./cmd/server
```

На Linux последние команды создают Linux binaries с именем .exe; для реального Windows укажите GOOS=windows GOARCH=amd64 либо используйте Actions. Для `-race` нужны поддерживаемая платформа и соответствующий toolchain.

Сейчас `go.sum` не сохранён. CI запускает `go mod tidy` отдельно в Linux/Windows job. `go mod download` из старой инструкции без tidy не стоит считать полной подготовкой. При первой возможности сохранить go.sum и indirect requirements, заменить CI-модификацию исходников на проверки `gofmt -l` и readonly dependencies.

Workflow запускается на push, PR, workflow_dispatch. Linux: tidy, gofmt (пока исправляет код, а не проверяет), vet, test -race. Windows job после успеха: Go build клиента и сервера, ISCC из `C:\Program Files (x86)\Inno Setup 6\ISCC.exe`, SHA-256 и artifact `windows-installer`.

Артефакт содержит setup.exe, checksums.txt, pulse-client.exe, pulse-server.exe. Branch build: версия клиента `v0.0.0-preview`, версия Inno `0.0.0`; ручной go build без ldflags: `dev` (update disabled). Tag `vX.Y.Z`: версия клиента равна тегу, Inno без `v`. Для Inno использовать простые числовые версии, prerelease форматы отдельно проверить.

Установщик per-user, без elevation, `%LOCALAPPDATA%\Programs\Pulse`, ярлыки, опциональный запуск клиента. Службу не устанавливает, сервер не стартует, серверный DATA_DIR не задаёт. Подписи Authenticode нет, возможен SmartScreen.

Release step при refs/tags/v* вызывает `gh release create ... --verify-tag`. Предполагает существующий тег и отсутствие Release с таким тегом. Само создание Release в UI не эквивалентно push-триггеру этого workflow; уже созданный Release может конфликтовать с `gh release create`. Перед production стоит сделать идемпотентную публикацию и описать единый процесс. В агентской сессии не создавать теги/релизы/другие ветки без разрешения и проверки текущих ограничений.

## 8. Механизм обновления и ограничения

Репозиторий зашит как `pmstein71-ui/testtest`. API: `https://api.github.com/repos/pmstein71-ui/testtest/releases/latest`, без токена. Нужен доступный публичный latest Release. HTTP client timeout 5 минут. Initial URL допускает только HTTPS api.github.com/github.com, стандартные redirects http.Client отдельно не фильтруются.

Проверка при загрузке UI и каждый час. Сейчас available = версия не dev, тег непустой и НЕ РАВЕН установленной версии. Это не semver-сравнение: возможны предложение более старого Release и downgrade. Это важная задача продолжения, не считать проблему решённой.

Install повторно запрашивает latest, ищет assets с точными именами setup.exe и checksums.txt. Лимиты: JSON/manifest 1 MiB, installer 200 MiB; загрузка целиком в память. SHA-256 проверяется по manifest, создаётся temp directory, EXE запускается через exec.Command. Temp автоматически не чистится. Нет mutex установки, автоматического завершения клиента, rollback и мониторинга результата инсталляции. Сервер перед обновлением остановить вручную.

Manifest и EXE из одного Release: checksum подтверждает целостность, не независимую подлинность. Реальное доверие — HTTPS + контроль GitHub repository. Нужны подпись, защита release workflow и E2E проверка на Windows.

## 9. Что действительно проверено

Коммиты реализации:
- `835a7f0` — Go client/server, SQLite, backups, installer, updates, CI.
- `8cab8b0` — branch installer получает preview version, чтобы предлагать Release update.

Успешные GitHub Actions runs:
- https://github.com/pmstein71-ui/testtest/actions/runs/45173509022 — первая сборка.
- https://github.com/pmstein71-ui/testtest/actions/runs/45173993364 — последняя сборка кода на момент передачи, commit 8cab8b0.

В обоих: vet, test -race, Windows compilation и Inno installer успешны. Это НЕ означает E2E проверку. Реальные тесты сейчас только два в internal/release: таблица Available и отклонение исходных URL. Сервер/client не имеют интеграционных тестов; race detector проверяет только выполненные тестами пути.

Не проверены вручную: интерактивная установка Windows, логин/выход через UI, backup/restore с нагрузкой, TLS deployment, выпуск настоящего Release и полный update cycle.

В агентской среде на момент реализации `go` отсутствовал. Попытка скачать artifact через gh run download не удалась: GitHub перенаправляет на blob.core.windows.net, доступ в среде ограничен. Поэтому EXE локально не открыт/не сохранён, formatted-source и go.sum не извлечены. Не утверждать, что go.sum зафиксирован: в переписке было намерение, но скачивание не удалось. Установщик пользователь скачивает из Actions. В логах CI были предупреждения о deprecated Node20 actions; задачи при этом успешны.

## 10. Рекомендуемое продолжение (по приоритету)

1. В среде с Go сохранить go.sum, indirect dependencies, gofmt исходников; обновить CI до проверки формата и неизменности module files. Проверить актуальность Go/actions/dependencies и уязвимости.
2. Вынести создание server handler/DB/backup из main для тестирования. Добавить integration tests bootstrap/login/me/logout/expiry/throttle, snapshot consistency и restore; client Host/Origin/header tests.
3. Исправить semver-сравнение обновлений; покрыть downgrade, prerelease, ошибочные теги, checksum mismatch, assets missing, HTTP failures, redirects, parallel installs.
4. Сделать publication идемпотентным, проверить реальный Release и обновление на Windows VM. Подписать setup.exe при наличии сертификата и защищённого способа подписи.
5. Исправить обработку частичной TLS-конфигурации (сейчас может молча перейти на HTTP), добавить конфиг/мастер сервера и Windows service по согласованию с пользователем.
6. Добавить администрирование пользователей, изменение/сброс пароля, более точный rate limit, аудит, единые ответы ошибок. Неверный логин сейчас отличается по времени от проверки bcrypt.
7. Retention и offsite backups, проверка диска, Windows ACL, метрики последнего успешного backup, автоматизированная проверка restore.
8. Улучшить lifecycle клиента: exit/tray, закрытие перед update, контроль установки, cleanup temp, единичный процесс. Закрытие вкладки сейчас не завершает Go client.
9. Уточнить у пользователя предметные функции и нужен ли WebView вместо браузера. Не придумывать бизнес-схему без требований.

## 11. Правила передачи

После изменения реализации обновить этот документ, указать commit/run и честно разделить «реализовано», «проверено», «запланировано». Не сохранять секреты, реальные пароли и DB. Поддерживать короткую точку входа AGENTS.md. Источник истины — текущий код и результаты проверок, а не обещания прошлых ответов.

## 12. Проверка сообщения о падении при запуске (2026-11-23)

Пользователь сообщил, что программа после загрузки падает. Точного текста ошибки/Windows event log пока нет, поэтому единственную причину не утверждаем.

При анализе найдены реальные причины неудачного старта и пробелы диагностики:
- Сервер на пустой БД без ADMIN_PASSWORD вызывал log.Fatal; при двойном щелчке консоль могла сразу закрыться. Теперь в настоящем терминале запрашивает и подтверждает скрытый пароль (12–72 байта). В headless режиме без пароля возвращает понятную ошибку.
- База по умолчанию зависела от рабочего каталога. Теперь используется os.UserConfigDir()/PulseServer/data; существующая data/app.db в текущем каталоге сохраняет приоритет для совместимости. Явный DATA_DIR имеет наивысший приоритет. Для эксплуатации по-прежнему рекомендован абсолютный DATA_DIR.
- Клиент игнорировал ошибку запуска браузера; теперь пишет её и URL в журнал. Отсутствующий сервер даёт понятный 502 вместо технического Bad Gateway и не завершает клиент.
- Частичная TLS-конфигурация теперь ошибка, а не молчаливый HTTP fallback.
- Общий internal/startup пишет журналы client.log/server.log в os.UserConfigDir()/Pulse/logs; ошибки старта и panic показываются в Windows MessageBox через PowerShell (при недоступности PowerShell остаётся журнал). Fatal exit code 1. Секреты в лог не выводить.
- В меню установщика добавлен отдельный ярлык Pulse Server. Автозапуска/службы сервера всё ещё нет.

Добавлены unit tests выбора каталога и scripts/smoke.py: реальные server/client процессы, ошибки конфигурации, bootstrap с env, рестарт существующей БД без пароля, snapshot integrity, HTML, Host/Origin/header boundary (проверка Origin и header), login/me/logout, offline server 502. Smoke запускается в CI и на Linux, и на Windows. Интерактивный ввод пароля, MessageBox, открытие браузера и установка setup.exe этим тестом не автоматизированы.

Для CI/диагностики без UI: PULSE_NO_DIALOG=1 отключает окно ошибки; PULSE_NO_BROWSER=1 отключает запуск браузера. Пароли не передавать аргументами командной строки. До завершения нового CI не считать новые проверки пройденными.

Ключевой дефект клиента найден при проверке: регистрация `/api/` (все методы) вместе с `GET /` вызывает panic нового Go ServeMux из-за пересечения маршрутов, ни один из которых не более специфичен другого (метод против пути). Корневой маршрут изменён на `/` с явной проверкой GET внутри handler. Не возвращать сочетание `/api/` + `GET /`. Первый добавленный smoke run 45184785969 выявил failure; для последующих ошибок тест также публикует диагностику в Actions annotations, так как download сырых логов в этой среде блокируется на redirects.

Run 45185132444: Linux vet/unit/smoke прошли после исправления маршрутов; Windows smoke завершился ошибкой. В тесте исправлено закрытие sqlite3 connection (Python `with connection` завершает транзакцию, но не закрывает файл; на Windows это мешает удалению TemporaryDirectory). Добавлен top-level вывод traceback в annotations, включая ошибки cleanup. Ожидается повторный CI; успешный Linux run не выдавать за успешный Windows.

Итог проверки исправления: **run 45185690023, commit 90564d0, SUCCESS** — https://github.com/pmstein71-ui/testtest/actions/runs/45185690023. Прошли Linux vet/unit tests (-race), реальные Linux smoke и реальные Windows EXE smoke; Inno Setup создал установщик, artifact windows-installer загружен. Windows smoke теперь проходит после закрытия snapshot connection. Регрессия конфликта ServeMux покрыта реальным запуском клиента. Ручная установка/браузер/MessageBox/терминальный ввод по-прежнему не проверены интерактивно. Для воспроизведения пользовательской ошибки при её сохранении запросить client.log и server.log без секретов.

## 13. Windows ввод первого пароля — отдельное окно

Пользователь сообщил, что консоль не реагирует на набор/вставку. Старый term.ReadPassword не отображал даже звёздочки; реальная блокировка терминала не подтверждена. Убрана зависимость Windows bootstrap от stdin/console mode: internal/password/password_windows.go запускает статический встроенный WinForms script (prompt.ps1) через Windows PowerShell -STA. Два поля с маской, вставкой, checkbox показа, подтверждением, проверкой 12–72 UTF-8 байта и точным регистрозависимым сравнением. Enter сохраняет, Escape/закрытие отменяет. Ошибка длины/совпадения остаётся в форме. Существующие пользователи/пароли не меняются.

ADMIN_PASSWORD имеет приоритет; PULSE_NO_DIALOG=1 при пустом ADMIN_PASSWORD запрещает интерактивный bootstrap (для CI/headless). Linux сохраняет term.ReadPassword. Скрипт передаётся как UTF-16LE EncodedCommand для корректной кириллицы; пароль НЕ в аргументах/файлах/logs, возвращается в pipe как base64 UTF-8 (это кодирование, не защита от процессов пользователя). Ошибки запуска PowerShell не включают захваченный output. Windows GUI недоступен в служебной/headless сессии: использовать ADMIN_PASSWORD.

Тесты: общая валидация длины в байтах, env bootstrap, headless error; Windows — roundtrip encoded script/пароля, invalid output, разбор PowerShell AST без открытия окна. В workflow добавлен go test ./... на Windows перед smoke. Существующий smoke с env паролем не открывает окно; ручной ввод/вставка в настоящем пользовательском desktop ещё требует проверки. Не считать AST/unit тесты проверкой реальной клавиатуры/clipboard.

Результат: commit 126ceb4, run https://github.com/pmstein71-ui/testtest/actions/runs/45202745644 — SUCCESS. Прошли Linux vet/unit (-race)/smoke, Windows unit (включая parsing встроенного WinForms script) и smoke реальных EXE; setup.exe собран, artifact windows-installer опубликован. Интерактивная keyboard/clipboard проверка окна вручную не выполнялась. Для пользователя следует использовать именно установщик этой или более новой успешной сборки.

## 14. Отказ WinForms у пользователя: setup на loopback странице

Скриншоты: сервер застрял перед созданием admin, клиент получает connection refused на 127.0.0.1:8080. Это не «неверный пароль», API ещё не начал слушать. Причина невидимого окна WinForms не установлена; предыдущие AST/unit tests его реальную видимость не проверяли. Учётки admin/admin не существует по умолчанию.

WinForms/PowerShell bootstrap полностью удалён. internal/password/browser.go и setup.html предоставляют локальную форму на случайном 127.0.0.1 порту; `SETUP: http://127.0.0.1:...` всегда печатается в log/console и браузер запускается автоматически (PULSE_NO_BROWSER=1 отключает только автозапуск). Открытие вручную не зависит от PowerShell, терминального ввода и видимости native окна. ADMIN_PASSWORD сохраняет приоритет; PULSE_NO_DIALOG=1 без него по-прежнему fail-fast. Linux тоже использует browser setup. x/term более не требуется.

Setup ограничен loopback, точным Host, POST Origin и случайным X-Pulse-Setup nonce из HTML (не из логов/URL), body до 2048 байт, маска/подтверждение/валидация 12–72 UTF-8 байт, CSP и no-store. Принимается один пароль, после ответа listener закрывается и обычный server bootstrap создаёт пользователя и начинает API. Страница сообщает «пароль принят», не гарантирует готовность API: ждать `Server listening on`; при ошибке DB/порта смотреть log. Через 15 минут setup завершает ожидание ошибкой. Существующие данные не сбрасываются. Setup защищён от браузерных cross-origin атак, но не от других локальных процессов того же компьютера — они в доверенной границе.

Добавлены handler tests boundary/validation/повторной отправки. В scripts/smoke.py теперь основной happy path реально запускает EXE с пустым ADMIN_PASSWORD, находит SETUP URL, читает HTML nonce, проверяет плохой Origin/короткий admin и завершает настройку HTTP-запросом; далее login/logout/restart/snapshot. Это выполняется и Linux, и Windows, но не проверяет клики в настоящем браузере. Исторические разделы 12/13 описывают старые реализации, текущий источник истины — этот раздел и код.
