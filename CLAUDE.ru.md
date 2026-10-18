# CLAUDE.md

## Стиль кода

**Не пиши комментарии в коде.** Никаких поясняющих комментариев к функциям, секционных заголовков (`// --- Foo ---`), описаний параметров и т.п. Код должен быть самодокументируемым через имена идентификаторов.

Исключение — только если без комментария читатель не поймёт *почему* (скрытый инвариант, обход бага, неочевидное ограничение API). В таких редких случаях — одна короткая строка.

## Язык

Весь код, скрипты, сообщения CLI, комментарии в CI-конфигах — **только на английском**.

Документация ведётся в двух языках, парами файлов:

- `<name>.ru.md` — русская версия (первоисточник);
- `<name>.en.md` — английская версия;
- `<name>.md` — симлинк на `<name>.en.md` (дефолт — английский).

Изменение документа требует правки **обеих** языковых версий. В оригинальном виде сохраняются только идентификаторы команд, флаги, ключи конфигов и shell-примеры.

## Назначение

`kdbx-cli` — обёртка, запускающая команду с секретами из `.kdbx`-хранилища. Каналов доставки пять: переменные окружения (`secrets`), stdin команды (`stdin`), файл через `memfd` + `/dev/fd/N` (`files`), файл, отрендеренный из шаблона с подстановкой секретов в содержимое, доставляемый тем же способом (`templates`), и askpass-хелпер (`askpass`). Секреты не попадают в историю shell, в argv и на диск в plaintext. Чтение `.kdbx` — через внешнюю `keepassxc-cli` (своего крипто нет).

```
kdbx-cli [--config <path>] [--key-store <path>] [--secrets=name:env,...]
         [--stdin=name,...] [--stdin-keep-open] [--secret-file=name,...]
         [--template=name:path,...] [--askpass=name] [--askpass-reads=N]
         [--dry-run] -- <cmd> [args...]
kdbx-cli config  [--config <path>] [-y]
kdbx-cli check   [--config <path>] [-y]
kdbx-cli show    [--config <path>]
kdbx-cli forget  [--config <path>]
kdbx-cli load    [--config <path>] [--key-store <path>] [--secrets=name:env,...]
                 [--stdin=name,...] [--secret-file=name,...]
                 [--template=name:path,...] [--askpass=name]
kdbx-cli migrate [--config <path>] [--from <N>] [--to <N>]
```

`--dry-run` печатает разрешённый план (конфиг, секция, маппинги, блоки stdin/files/templates/askpass, итоговая команда с плейсхолдерами `<secret from Title>` и `<file with Title>`) и завершается, не читая хранилище и не запрашивая пароль.

`config` после сохранения и `check` сверяют наличие `Title` в `.kdbx` (через `keepassxc-cli export`) и предлагают создать отсутствующий файл/записи (`-y` — без подтверждений). `check` агрегирует требуемые `Title` по файлам через `resolve` по всем секциям и учитывает все пять каналов (`Resolved.AllTitles` плюс `Resolved.TemplateTitles`, которая сканирует содержимое каждого шаблона на `{{Title}}`).

## Конфиг и кэш

Схема конфига — struct `Config{ Version int json:"version"; Sections map[string]Section json:"sections"; Cache *CacheConfig json:"cached,omitempty" }`; `Section{ KeyStore, Secrets map[string]string (имя env-переменной → имя записи; ключ уникален, поэтому одна запись может питать несколько переменных), Stdin []string, StdinKeepOpen bool, Files []string, Templates []Template, Askpass string, AskpassReads int }`, `Template{ Name, Path string }`. При слиянии секций `Secrets` дополняется поэлементно, `Stdin`/`Files`/`Templates` заменяются целиком (порядок строк значим). `config.Load` читает только схему текущей версии (`config.CurrentVersion`); конфиг без поля `version` считается версией 0, а любая версия, отличная от текущей, — ошибка с подсказкой запустить `kdbx-cli migrate`.

**Версионирование конфига.** Тулза зарелизилась, поэтому формат конфига меняется только явно: любое несовместимое изменение схемы (переименование/переворот/удаление поля, смена типа) обязано поднять `config.CurrentVersion`, добавить шаг `N → N+1` в `steps` (`internal/config/migrate.go`, работает над сырым JSON) и покрыть его тестом. `config.Save` всегда пишет текущую версию в конфиг. `kdbx-cli migrate [--from N] [--to M]` (по умолчанию `--from=0`, `--to=config.CurrentVersion`) применяет цепочку шагов к файлу на месте; если файл уже на `--to`, это не ошибка, а при несовпадении версии файла с `--from` — ошибка.

Кэш пароля (`internal/keyring`): opt-in только через `cached`-секцию (`enabled`, `ttl`); никаких флагов/env. Хранит master-пароль `.kdbx` и `Title`, разрешённые через `kdbx-cli load`, как JSON `entry{Password, Titles}` (`encodeEntry`/`decodeEntry`; голый пароль от прежних версий декодируется как запись без `Title`) за интерфейсом `backend`, для которого значение непрозрачно: на Linux — kernel keyring (`kernel_linux.go`, `keyctl` через `golang.org/x/sys/unix`, ключ `kdbx-cli:<канонический путь>` из `storeKey`, TTL — `KEYCTL_SET_TIMEOUT`), на остальных ОС — OS-keyring через `github.com/zalando/go-keyring` (`secretservice_other.go`, JSON с `expires_at`). `store.UnlockExport(…, allowCache)` — единая точка «достать пароль (кэш→prompt) + export»; кэш читается, только если все запрошенные Title есть в конфиге тулзы (`resolve.ConfiguredTitles`) или в `Cache.Granted`. `keyring.New(cfg.Cache)` строит политику. `Cache.Grant` объединяет `Title` с живой записью и перезапускает TTL, `Cache.Remember` сохраняет существующие `Title`. `kdbx-cli load` (`cmdLoad`) всегда спрашивает пароль (`UnlockExport` с `allowCache=false`), проверяет наличие каждого `Title` и вызывает `Cache.Grant`; резолвит секцию `default` плюс флаги и требует включённой секции `cached`. Разрешения исчезают только вместе с записью. `kdbx-cli forget` (`cmdForget` → `keyring.ForgetAll`) удаляет все ключи `kdbx-cli:*` и старые записи Secret Service. `Cache` держит зависимости в поле `storage` (`backend` плюс legacy `keyring.Keyring` из go-keyring); `New` и `ForgetAll` берут `systemStorage()` (`kernelBackend`/`secretServiceBackend` и `systemKeyring`), тесты собирают `Cache` поверх фейков; глобальных переменных для подмены нет. Без keyring деградирует (miss + warning, не падает).

## Структура

Код разбит на пакеты под `internal/` (без циклов: `utils`/`config`/`term`/`secretpipe` — листья; `keepass → utils`; `keyring → config,utils`; `tmpl → config,utils`; `resolve → config,tmpl,utils`; `store → config,resolve,keepass,keyring,term,utils`; `deliver → config,secretpipe,tmpl,utils`; `tui → config,resolve,store,keyring,utils`; `cmd` — клей над всеми ними; корневой `main → cmd`).

- `cmd/kdbx-cli/main.go` (package `main`) — только `var version` (задаётся через `-X main.version`) и вызов `cmd.Execute(version)`.
- `internal/utils` — обобщённые утилиты коллекций: `Filter`, `Map`, `FlatMap`, `Unique`, `UniqueBy`, `GroupBy`, `ToSet`, `SortedKeys`, `SortedUnique`.
- `internal/config` — `Config{Sections,Cache}`/`Section`/`CacheConfig` (JSON), `Load`/`Save` (`Load` предупреждает о небезопасных правах через `WarnUnsafePermissions`/`PermissionWarnings`), `CurrentVersion`, `Migrate` (шаги миграции схемы); `ExpandHome`, `Dir` (`~/.config/kdbx-cli`), `DefaultPath` (`~/.config/kdbx-cli/default`).
- `internal/keepass` — `CheckEngine` (ищет `keepassxc-cli`, проверяет его права через `checkEngineFile` и фиксирует путь в `enginePath`), `Run` (вызов зафиксированного `keepassxc-cli`), разбор KeePass XML (`ParseSecrets`, тип `Entry`), `LookupSecret`; операции записи `CreateStore` (`db-create`), `AddEmptySecret` (`mkdir`+`add`).
- `internal/keyring` — `Cache`, `New(cfg.Cache)`, методы `Get/Remember/Grant/Granted/Enabled/Forget/Status`, `entry`/`mergeTitles`, `ForgetAll`, интерфейс `backend` (`kernelBackend` на Linux, `secretServiceBackend` на остальных), `storeKey`, `storage`/`systemStorage`, `systemKeyring` (реализация `keyring.Keyring` на go-keyring).
- `internal/term` — ввод с терминала через `/dev/tty` (`KDBX_CLI_PASSWORD` для тестов): `ReadPassword` (возвращает `ErrNoTerminal`, если настоящего терминала нет), `ReadWithPrefill` (fallback-ввод), `Confirm` (Y/N, учитывает `-y`), `IsInteractive`.
- `internal/secretpipe` — доставка секретов в обход env: `Set.File` (анонимный `memfd` с правами `0600`, запечатанный `sealAnonFile`, передаётся ребёнку через `cmd.ExtraFiles` как `/dev/fd/N`), `Set.Askpass(name, value, reads)` (FIFO в каталоге `0700` и скрипт, который отказывает на `Username*`/`(yes/no` и выполняет `head -n 1`; горутина `feed` отдаёт секрет первым `AskpassReads(reads)` читателям, по умолчанию `DefaultAskpassReads` = 1, остальным — пустой ответ), `Set.Close`.
- `internal/tmpl` — шаблоны: `Titles` (поиск `{{Title}}` в содержимом шаблона), `Render` (подстановка значений, для путей `.json` — с JSON-экранированием).
- `internal/resolve` — `Resolve` (слияние `default` → секция тулзы → флаги слоями, типы `Overrides`/`Resolved`), `Resolved.AllTitles`, `Resolved.TemplateTitles`, `ConfiguredTitles`; `Mapping`/`MappingsFromMap`/`MappingsToMap`/`SortMappings`/`ValidateEnvNames`.
- `internal/store` — работа с файлами `.kdbx`: `UnlockExport` (кэш→промпт→export), `AggregateStores`/`AggregateStoreMappings` (через `sectionUsages`, метки `Channel*`), `ReconcileStores` (`gatherMissing` → отчёт → создание недостающего), `BuildStoreViews`/`StoreView`.
- `internal/deliver` — сборка входов дочерней команды: `InheritedEnv` (убирает `KDBX_CLI_PASSWORD`), `SecretEnv`, `WithAskpass`/`AskpassEnv`, `Stdin` (`stdinPayload`), `RenderFiles` → `Files` (`Paths` даёт `/dev/fd/N`, `Open` создаёт memfd), `Placeholder`, `SubstitutePlaceholders`, `CloseFiles`.
- `internal/tui` — модели Bubble Tea. `RunConfig` → `ConfigResult`: `configTUI` — клей над `keyStoreField` (автодополнение пути: `refreshPathSuggestions`, `deleteLastPathSegment` на `alt+backspace`, раскрытие `~` на `tab`), `pairList`, `pairEditor` (хоткеи `a`/`e`/`d`/`↑↓`/`tab`/`ctrl+s`/`esc`) и `cacheForm`. `RunShow(ShowData, LaunchFunc)`: read-only `showTUI` со списком `.kdbx` (`↑/↓`, `Enter` → открыть в GUI через `LaunchKeePassXC` или фейк в тестах); `PrintStores` для не-TTY; `GrantedLine`, `FormatDuration`.
- `internal/cmd` — CLI-слой (клей):
  - `execute.go` — `Execute(version)`: `handleBuiltin` (version, completion, `--path`), `runSubcommand` (`config`/`check`/`show`/`forget`/`load`/`migrate`), `runMode`; `usage`/`usageText`, `parseConfigArgs`.
  - `args.go` — `splitArgs` (по `--`), `parseRunFlags` (`parseArg` → `setSwitch`/`setValue`), `mergeSecretsFlag` (`name:env` → `dst[env]=name`), `splitList`, `parseTemplatesFlag`, тип `runFlags` и его `overrides()`.
  - `harden_linux.go`/`harden_other.go` — `hardenProcess` (`PR_SET_DUMPABLE=0` в начале `Execute`).
  - `secrets.go` — хелперы, общие для `run` и `load`: `configPathOrDefault`, `loadRunConfig`, `requestedTitles`, `existingKeyStore`, `parseEntries`, `lookupAll`, `titlesGranted`.
  - `run.go` — `cmdRun`: `resolveRun` → `runPlan` → (если `--dry-run` → `printPlan`) → `fetchSecrets` (пароль; `titlesGranted` решает, можно ли брать кэш, с учётом `Cache.Granted`) → `runChild` (`prepareChild` собирает `childSpec` через `deliver`, `execChild` запускает его и пробрасывает код выхода).
  - `plan.go` — `printPlan` (`printPlanHeader`/`Mappings`/`Stdin`/`Files`/`Templates`/`Askpass`) и хелперы dry-run (`describeSection`, `renderCommand`, `shellQuote`).
  - `migrate.go` — `parseMigrateArgs`, `cmdMigrate`.
  - `commands.go` — `cmdConfig` (`editDefaultSection`: TTY → `tui.RunConfig`, иначе `configFallback`; `applyDefaultEdit`; после сохранения — `ReconcileStores` для default), `cmdCheck`, `cmdForget`.
  - `load.go` — `cmdLoad` (`loadAndGrant`: всегда спрашивает пароль, проверяет `Title`, `Cache.Grant`), `printLoaded`.
  - `show.go` — `cmdShow`: собирает `tui.ShowData`, TTY → `tui.RunShow`, иначе `tui.PrintStores`.

## Сборка и тесты

Зависит от `keepassxc-cli` в PATH. Go 1.26.1. TUI — на Bubble Tea (`charmbracelet/bubbletea`, `bubbles`, `lipgloss`); кэш пароля — kernel keyring через `golang.org/x/sys/unix` на Linux и `zalando/go-keyring` (godbus, без cgo) на остальных ОС. Сборка статическая (`CGO_ENABLED=0`).

- `make build` — собрать бинарь `./kdbx-cli` (версия из `versions.txt`).
- `make unit-test` — юнит-тесты (`go test ./...`).
- `make test` — то же, что `unit-test`.
- `make bump-version` — поднять `versions.txt`.

Интеграционные тесты живут в отдельном репозитории `kdbx-cli-tests` и гоняются против выпущенных версий; здесь остаются только юнит-тесты.

Релиз — push в `main`/`master`, тег `v<versions.txt>` (см. конвенции `dimkarp93/install`). Две площадки, одинаковые артефакты:

- GitHub Actions — `.github/workflows/release.yml`.
- Gitea Actions — `.gitea/workflows/release.yml`: без внешних actions (checkout, установка Go и публикация — шаги `run:` на shell), релиз создаётся через Gitea API.
