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

`kdbx-cli` — обёртка, запускающая команду с секретами из `.kdbx`-хранилища. Каналов доставки четыре: переменные окружения (`secrets`), stdin команды (`stdin`), файл через `memfd` + `/dev/fd/N` (`files`) и askpass-хелпер (`askpass`). Секреты не попадают в историю shell, в argv и на диск в plaintext. Чтение `.kdbx` — через внешнюю `keepassxc-cli` (своего крипто нет).

```
kdbx-cli [--config <path>] [--key-store <path>] [--secrets=name:env,...]
         [--stdin=name,...] [--stdin-keep-open] [--secret-file=name,...]
         [--askpass=name] [--dry-run] -- <cmd> [args...]
kdbx-cli config  [--config <path>] [-y]
kdbx-cli check   [--config <path>] [-y]
kdbx-cli show    [--config <path>]
kdbx-cli forget  [--config <path>]
```

`--dry-run` печатает разрешённый план (конфиг, секция, маппинги, блоки stdin/files/askpass, итоговая команда с плейсхолдерами `<secret from Title>` и `<file with Title>`) и завершается, не читая хранилище и не запрашивая пароль.

`config` после сохранения и `check` сверяют наличие `Title` в `.kdbx` (через `keepassxc-cli export`) и предлагают создать отсутствующий файл/записи (`-y` — без подтверждений). `check` агрегирует требуемые `Title` по файлам через `resolve` по всем секциям и учитывает все четыре канала (`Resolved.AllTitles`).

## Конфиг и кэш

Схема конфига — struct `Config{ Sections map[string]Section json:"sections"; Cache *CacheConfig json:"cached,omitempty" }`; `Section{ KeyStore, Secrets map[string]string, Stdin []string, StdinKeepOpen bool, Files []string, Askpass string }`. При слиянии секций `Secrets` дополняется поэлементно, `Stdin`/`Files` заменяются целиком (порядок строк значим). `config.Load` читает только эту схему (программа в разработке, обратной совместимости со старыми форматами нет).

Кэш пароля (`internal/keyring`): opt-in только через `cached`-секцию (`enabled`, `ttl`); никаких флагов/env. Хранит master-пароль `.kdbx` в OS-keyring через `github.com/zalando/go-keyring`. `domain.UnlockExport` — единая точка «достать пароль (кэш→prompt) + export»; `keyring.New(cfg.Cache)` строит политику. `kdbx-cli forget` (`cmdForget`) чистит keyring для всех `.kdbx` конфига. keyring-обёртки (`keyringSet/Get/Delete`) — var'ы, стабятся в тестах. Деградирует без Secret Service (miss + warning, не падает).

## Структура

Код разбит на пакеты под `internal/` (без циклов: `config`/`keepass`/`term`/`secretpipe` — листья; `keyring → config`; `domain → config,keepass,keyring,term`; `cmd → domain,config,keepass,keyring,term,secretpipe`; корневой `main → cmd`).

- `main.go` (package `main`) — только `var version` (через `-X main.version`) и вызов `cmd.Execute(version)`.
- `internal/config` — `Config{Sections,Cache}`/`Section`/`CacheConfig` (JSON), `Load`/`Save`; `ExpandHome`, `Dir` (`~/.config/kdbx-cli`), `DefaultPath` (`~/.config/kdbx-cli/default`).
- `internal/keepass` — `CheckEngine`, `Run` (вызов `keepassxc-cli`), парсинг KeePass XML (`ParseSecrets`, тип `Entry`), `LookupSecret`; операции записи `CreateStore` (`db-create`), `AddEmptySecret` (`mkdir`+`add`).
- `internal/keyring` — `Cache`, `New(cfg.Cache)`, методы `Get/Remember/Forget`, подменяемые `keyringSet/Get/Delete` (go-keyring / Secret Service).
- `internal/term` — терминальный ввод через `/dev/tty` (`KDBX_CLI_PASSWORD` для тестов): `ReadPassword`, `ReadWithPrefill` (fallback-ввод), `Confirm` (Y/N, учитывает `-y`), `IsInteractive`.
- `internal/secretpipe` — доставка секретов вне env: `Set.File` (анонимный `memfd`, отдаётся ребёнку через `cmd.ExtraFiles` как `/dev/fd/N`), `Set.Askpass` (FIFO в каталоге `0700` + скрипт `head -n 1`, горутина `feed` пишет секрет каждому новому читателю), `Set.Close`.
- `internal/domain` — логика приложения: `Resolve` (слияние `default` → секция тулзы → флаги, типы `Overrides`/`Resolved`, `Resolved.AllTitles`), `Mapping`/`MappingsFromMap`/`MappingsToMap`, `UnlockExport` (кэш→prompt→export), `AggregateStores`/`AggregateStoreMappings`, `gatherMissing`, `ReconcileStores` (отчёт + создание недостающего), `BuildStoreViews`/`StoreView`.
- `internal/cmd` — CLI-слой:
  - `execute.go` — `Execute(version)`: разбор argv, диспетчеризация (`config`/`check`/`show`/`forget`/`version`/run-режим), `usage`, `parseConfigArgs`.
  - `args.go` — `splitArgs` (по `--`), `parseRunFlags`, `mergeSecretsFlag`, `splitTitles`, тип `runFlags` и его `overrides()`.
  - `run.go` — `cmdRun`: резолв → (если `--dry-run` → `printPlan`) → пароль → export → доставка по всем каналам (env, `stdinPayload`, `substitutePlaceholders` + `ExtraFiles`, `withAskpass`) → запуск дочерней команды, проброс кода возврата.
  - `plan.go` — `printPlan` и хелперы dry-run (`describeSection`, `renderCommand`, `shellQuote`).
  - `commands.go` — `cmdConfig` (TTY → TUI, иначе `configFallback`; после сохранения — `ReconcileStores` для default), `cmdCheck`, `cmdForget`.
  - `tui.go` — Bubble Tea-модель настройки `default`: поле `key-store` с автодополнением пути (`refreshPathSuggestions`, `deleteLastPathSegment` на `alt+backspace`, раскрытие `~` на `tab`), построчный редактор маппингов с хоткеями (`a`/`e`/`d`/`↑↓`/`tab`/`ctrl+s`/`esc`) и легендой.
  - `show.go` — `cmdShow`: read-only TUI (`showTUI`) со списком `.kdbx` (навигация `↑/↓`), `Enter` → открыть в GUI через `openInKeePassXC` (var, стабится в тестах); не-TTY → печать.

## Сборка и тесты

Зависит от `keepassxc-cli` в PATH. Go 1.26.1. TUI — на Bubble Tea (`charmbracelet/bubbletea`, `bubbles`, `lipgloss`); кэш пароля — `zalando/go-keyring` (godbus, без cgo). Сборка статическая (`CGO_ENABLED=0`).

- `just build` — собрать бинарь `./kdbx-cli` (версия из `versions.txt`).
- `just unit-test` — юнит-тесты (`go test ./...`).
- `just e2e-test` — e2e (тег `e2e`, реально создаёт/читает `.kdbx` через `keepassxc-cli`).
- `just test` — всё вместе.
- `just bump-version` — поднять `versions.txt`.

Релиз — push в `main`/`master`, тег `v<versions.txt>` (см. конвенции `dimkarp93/install`). Две площадки, одинаковые артефакты:

- GitHub Actions — `.github/workflows/release.yml`.
- Gitea Actions — `.gitea/workflows/release.yml`: без внешних actions (checkout, установка Go и публикация — шаги `run:` на shell), релиз создаётся через Gitea API.
