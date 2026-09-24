# kdbx-cli

`kdbx-cli` — обёртка для CLI-утилит, которая достаёт секреты из зашифрованного `.kdbx`-хранилища и передаёт их дочерней команде: через переменные окружения, её stdin, файл или askpass-хелпер.

Зачем: многие утилиты требуют передачи токенов через env-переменные. Делать это вручную из shell неудобно и небезопасно — секрет попадает в историю команд и в plaintext-файлы. `kdbx-cli` решает это: значение секрета берётся из зашифрованного `.kdbx`, а сам вызов выглядит так:

```sh
kdbx-cli --config ~/.config/kdbx-cli/install_secrets -- install user/repo
```

В строке вызова секрета нет — история чистая. Секрет существует только в окружении дочернего процесса.

Это тот же паттерн, что у `op run -- cmd` (1Password CLI), `envchain`, `aws-vault exec`, `sops exec-env` — но self-hosted, на базе `keepassxc`.

## Требования

Нужна установленная `keepassxc-cli` (входит в KeePassXC):

```sh
# Debian/Ubuntu
sudo apt install keepassxc
# Arch Linux
sudo pacman -S keepassxc
# macOS
brew install keepassxc
```

## Установка

Через установщик [`dimkarp93/install`](https://github.com/dimkarp93/install):

```sh
github_install.sh dimkarp93/kdbx-cli
# или одной строкой:
curl -fsSL https://raw.githubusercontent.com/dimkarp93/install/master/install.sh | sh -s -- dimkarp93/kdbx-cli
```

## Использование

```
kdbx-cli [--config <path>] [--key-store <path>] [--secrets=name:env,...]
         [--stdin=name,...] [--stdin-keep-open] [--secret-file=name,...]
         [--askpass=name] [--dry-run] -- <cmd> [args...]
kdbx-cli config [--config <path>]
kdbx-cli version | --version | -v
```

Всё после `--` — команда, которую надо запустить. Секцию конфига выбирает **базовое имя** первого слова команды (`install` в примере; для `/usr/bin/sudo` это будет `sudo`); если такой секции нет, используется `default`. Флаги должны идти **до** `--`.

Флаги:

- `--config <path>` — путь до конфига. По умолчанию `~/.config/kdbx-cli/default`.
- `--key-store <path>` — полный путь до `.kdbx`-файла. Перетирает значение из конфига.
- `--secrets=name:env,...` — маппинг секретов на env-переменные. Сливается поверх конфига.
- `--stdin=name,...` — записать секреты в stdin команды, по строке на секрет, в указанном порядке, и закрыть stdin.
- `--stdin-keep-open` — не закрывать stdin после секретов, а пробросить туда остаток своего stdin.
- `--secret-file=name,...` — отдать секрет файлом и подставить путь к нему вместо плейсхолдера `{{name}}` в команде.
- `--askpass=name` — отдавать секрет через askpass-хелпер (для команд, которые читают только с терминала).
- `--dry-run` — не запускать команду и не трогать хранилище: показать разрешённый план (см. ниже).

При запуске `kdbx-cli` спрашивает пароль от хранилища (ввод скрыт, читается с `/dev/tty`). Отмена — `Ctrl+C` / `Ctrl+D`.

### Пример

```sh
kdbx-cli --key-store ~/secrets/tokens.kdbx --secrets=GITHUB_TOKEN:GH_TOKEN -- gh repo list
```

`kdbx-cli` прочитает запись с заголовком `GITHUB_TOKEN` из `tokens.kdbx`, положит её пароль в переменную `GH_TOKEN` и запустит `gh repo list`.

### Предпросмотр: `--dry-run`

С флагом `--dry-run` команда не выполняется, пароль не запрашивается и `.kdbx` не читается. Выводится разрешённый план: какой конфиг используется, какая секция применена, активные маппинги и итоговая команда (значения секретов заменены плейсхолдером `<secret from Title>`):

```sh
$ kdbx-cli --dry-run -- install user/repo
Dry run — the command will NOT be executed.

Config file:  /home/user/.config/kdbx-cli/default
Tool:         install
Section:      "install" (merged over "default")
Key-store:    ~/.config/kdbx-cli/store.kdbx

Mappings (env ← secret):
  GH_TOKEN  ← GITHUB_TOKEN
  NPM_TOKEN ← NPM_TOKEN

Command:
  GH_TOKEN=<secret from GITHUB_TOKEN> NPM_TOKEN=<secret from NPM_TOKEN> install user/repo
```

## Конфиг

Конфиг — JSON-объект с полями `sections` (набор секций по именам тулз плюс `default`) и необязательным `cached` (см. [Кэширование пароля](#кэширование-пароля)). В каждой секции:

- `key-store` — полный путь до `.kdbx`-файла;
- `secrets` — маппинг `имя_в_хранилище: имя_env`;
- `stdin` — список секретов, которые пишутся в stdin команды (аналог `--stdin`);
- `stdin-keep-open` — `true`, если после секретов stdin не закрывать (аналог `--stdin-keep-open`);
- `files` — список секретов, отдаваемых файлом через плейсхолдер `{{Title}}` (аналог `--secret-file`);
- `askpass` — один секрет, отдаваемый через askpass-хелпер (аналог `--askpass`).

Любой канал доставки настраивается и флагом, и конфигом — флаги лишь перекрывают конфиг, отдельного «только-CLI» канала нет. Подробности — в разделе [Способы доставки секрета](#способы-доставки-секрета).

```json
{
  "sections": {
    "default": {
      "key-store": "~/.config/kdbx-cli/store.kdbx",
      "secrets": { "GITHUB_TOKEN": "GH_TOKEN" }
    },
    "install": {
      "secrets": { "NPM_TOKEN": "NPM_TOKEN" }
    }
  },
  "cached": { "enabled": true, "ttl": "10m" }
}
```

**Слияние:** `default` — это база. Секция тулзы накладывается сверху: переопределяет `key-store` (если задан) и добавляет/переопределяет маппинги секретов. Для примера выше команда `install` получит общий `key-store` из `default` и оба секрета — `GITHUB_TOKEN` и `NPM_TOKEN`. Флаги `--key-store` / `--secrets` перетирают результат.

### Как ищется секрет

«Имя секрета» в конфиге — это **заголовок (Title)** записи в `.kdbx`, значением подставляется поле **Password** этой записи. Если в хранилище несколько записей с одинаковым Title, укажите полный путь `Группа/Подгруппа/Title`:

```json
"secrets": { "web/API_KEY": "API_KEY" }
```

## Способы доставки секрета

Далеко не каждая утилита читает секрет из окружения. `kdbx-cli` умеет четыре канала; их можно комбинировать в одной секции.

### `secrets` — переменные окружения

Канал по умолчанию, описан выше.

### `stdin` — запись в stdin команды

Секреты пишутся в stdin дочерней команды по строке на секрет в указанном порядке, после чего stdin закрывается (именно этого ждут `--password-stdin`-флаги). Список, а не одно значение: часть утилит спрашивает пароль дважды.

```sh
kdbx-cli --stdin=ghcr-token -- docker login ghcr.io -u me --password-stdin
kdbx-cli --stdin=gh-token   -- gh auth login --with-token
kdbx-cli --stdin=vault-root -- vault login -
kdbx-cli --stdin=new-pw,new-pw -- keepassxc-cli db-create -p ~/new.kdbx
```

`--stdin-keep-open` нужен, когда команда читает пароль из stdin, но потом ждёт там же данные:

```sh
kdbx-cli --stdin=sudo-pw --stdin-keep-open -- sudo -S tee /etc/foo.conf < local.conf
```

Секрет с переводом строки в этом режиме передать нельзя — `kdbx-cli` завершится с ошибкой.

### `files` — секрет как файл

Секрет кладётся в анонимный файл в памяти (`memfd`), дочерняя команда получает его как `/dev/fd/N`, а `kdbx-cli` подставляет этот путь вместо плейсхолдера `{{Title}}` в аргументах. На файловой системе секрет не появляется. Файл можно читать сколько угодно раз.

```sh
kdbx-cli --secret-file=restic-repo -- restic -r sftp:backup:/b --password-file '{{restic-repo}}' snapshots
kdbx-cli --secret-file=mysql-ini   -- mysql --defaults-extra-file='{{mysql-ini}}' mydb
kdbx-cli --secret-file=gpg-pass    -- gpg --batch --pinentry-mode loopback --passphrase-file '{{gpg-pass}}' -d file.gpg
```

Плейсхолдер берите в одинарные кавычки, чтобы его не тронул шелл. Если плейсхолдера нет в команде, `kdbx-cli` завершится с ошибкой, а не запустит команду молча.

### `askpass` — для команд, которые читают только с терминала

`ssh`, `sudo`, `git` при запросе пароля открывают `/dev/tty` напрямую — запись в stdin им не поможет. Штатный обход — askpass-хелпер. `kdbx-cli` создаёт такой хелпер и выставляет `SSH_ASKPASS`, `SSH_ASKPASS_REQUIRE=force`, `SUDO_ASKPASS`, `GIT_ASKPASS`, `GIT_TERMINAL_PROMPT=0`, `RESTIC_PASSWORD_COMMAND`, `BORG_PASSCOMMAND`.

```sh
kdbx-cli --askpass=ssh-key-pass -- ssh -T git@github.com
kdbx-cli --askpass=gitlab-pat   -- git push origin master
kdbx-cli --askpass=sudo-pw      -- sudo -A apt update
kdbx-cli --askpass=restic-pw    -- restic -r sftp:backup:/b snapshots
```

`sudo` смотрит на `SUDO_ASKPASS` только с флагом `-A` — его нужно указать самому.

### В конфиге

```json
{
  "sections": {
    "default": { "key-store": "~/.config/kdbx-cli/store.kdbx" },
    "docker":  { "stdin": ["ghcr-token"] },
    "sudo":    { "stdin": ["sudo-pw"], "stdin-keep-open": true },
    "restic":  { "files": ["restic-repo"] },
    "ssh":     { "askpass": "ssh-key-pass" }
  }
}
```

Каналы комбинируются в одной секции — например, `restic` может получать `B2_ACCOUNT_KEY` из окружения, пароль репозитория файлом, а пароль ssh-ключа через askpass:

```json
"restic": {
  "secrets": { "B2_KEY": "B2_ACCOUNT_KEY" },
  "files":   ["restic-pw"],
  "askpass": "ssh-key-pass"
}
```

#### Приоритет

Значения накладываются в порядке `default` → секция тулзы → флаги, по таким правилам:

| Поле | Как накладывается |
| --- | --- |
| `key-store` | заменяется, если задано непустым |
| `secrets` | дополняется поэлементно (ключ-коллизия — побеждает более поздний слой) |
| `stdin`, `files` | заменяются целиком: порядок строк значим, склейка списков дала бы неожиданный результат |
| `stdin-keep-open` | включается, если `true` хотя бы на одном слое |
| `askpass` | заменяется, если задан непустым |

Отсюда следствия, о которых легко забыть:

- `--stdin=a --stdin=b` в одном вызове накапливаются в `a,b`, но вместе они заменяют весь список `stdin` из конфига, а не дополняют его. То же с `--secret-file`.
- Выключить `stdin-keep-open` или `askpass`, заданные в `default`, из секции тулзы или флагом нельзя — флагов `--no-stdin-keep-open` / `--no-askpass` нет. Такие настройки держите в секции конкретной тулзы, а не в `default`.
- `--secrets` не отключает маппинги из конфига, а только добавляет свои и переопределяет одноимённые.

Проверить итог, не трогая хранилище и не вводя пароль, можно через `--dry-run` — он печатает блоки `Stdin`, `Files` и `Askpass` вместе с применённой секцией:

```sh
$ kdbx-cli --dry-run -- restic -r sftp:b:/b --password-file '{{restic-pw}}' snapshots
...
Section:      "restic" (merged over "default")

Mappings (env ← secret):
  B2_ACCOUNT_KEY ← B2_KEY

Files (placeholder ← secret):
  {{restic-pw}} ← restic-pw

Askpass:      ssh-key-pass
              SSH_ASKPASS, SUDO_ASKPASS, GIT_ASKPASS, RESTIC_PASSWORD_COMMAND, BORG_PASSCOMMAND

Command:
  B2_ACCOUNT_KEY=<secret from B2_KEY> restic -r sftp:b:/b --password-file '<file with restic-pw>' snapshots
```

`kdbx-cli check` и `kdbx-cli show` учитывают секреты всех четырёх каналов.

## Команда `config`

`kdbx-cli config` интерактивно настраивает секцию `default` (путь до `key-store` и маппинг секретов) и записывает конфиг:

```sh
kdbx-cli config
# или в произвольный файл:
kdbx-cli config --config ~/.config/kdbx-cli/install_secrets
```

В терминале открывается TUI:

- если конфиг уже существует, поля **предзаполняются** текущими значениями;
- поле `key-store` поддерживает **автодополнение пути**: `Tab` — дополнить (и раскрыть `~` в полный путь), `↑/↓` — перебрать варианты в текущем каталоге, `Alt+Backspace` — удалить последний сегмент пути (до `/`);
- маппинги секретов показываются **построчно** и редактируются хоткеями: `a` — добавить, `e` — изменить, `d` — удалить, `↑/↓` — выбор, `Tab` — переключиться между полем пути и списком, `Ctrl+S` — сохранить, `Esc` — отмена. Легенда хоткеев всегда видна внизу.

Если стандартный ввод/вывод не является терминалом (пайп, скрипт), `config` переключается на простой построчный ввод без TUI.

TUI редактирует только секцию `default` и только `key-store`, маппинги `secrets` и кэш пароля. Каналы `stdin`, `files`, `askpass` и секции других тулз через него не настраиваются — их правят прямо в JSON-файле конфига; при сохранении из TUI они сохраняются как есть, ничего не затирается.

После сохранения `config` проверяет хранилище секции `default`:

- если файла `.kdbx` нет — предлагает создать его (запросит новый пароль);
- затем сверяет, что все указанные `Title` есть в хранилище; отсутствующие выводит списком и предлагает добавить как пустые записи.

С флагом `-y` все недостающее (файл, секреты) создаётся без вопросов.

## Команда `check`

`kdbx-cli check` проверяет, что во всех `.kdbx`-файлах, на которые ссылается конфиг (по всем секциям, с учётом слияния с `default`), есть записи со всеми нужными `Title`. Отсутствующие выводятся **сгруппированно по файлам**, после чего предлагается добавить их как пустые записи.

```sh
kdbx-cli check
kdbx-cli check --config ~/.config/kdbx-cli/install_secrets
kdbx-cli check -y          # добавить все недостающие записи без подтверждения
```

Пример вывода:

```
Missing secrets:
  /home/user/.config/kdbx-cli/store.kdbx
    - NPM_TOKEN
  /home/user/work/deploy.kdbx  (key-store does not exist)
    - AWS_KEY
```

Для каждого `.kdbx` запрашивается пароль (нужен для чтения и для добавления записей).

## Команда `show`

`kdbx-cli show` показывает текущий конфиг: путь до файла и список `.kdbx`-хранилищ с их маппингами (`name → env`). По хранилищам можно перемещаться стрелками `↑/↓`, а по `Enter` выбранный файл открывается в **GUI KeePassXC** (бинарь `keepassxc`).

```sh
kdbx-cli show
kdbx-cli show --config ~/.config/kdbx-cli/install_secrets
```

- Отсутствующие на диске файлы помечаются `(missing)`; открыть их нельзя.
- `q` / `Esc` — выход.
- Если ввод/вывод не терминал, `show` просто печатает конфиг без навигации.

## Кэширование пароля

Внутри одного вызова `kdbx-cli` пароль на базу спрашивается один раз. Чтобы не вводить его повторно **между** вызовами в рамках сессии, можно включить кэш в конфиге:

```json
"cached": { "enabled": true, "ttl": "10m" }
```

- `enabled` — включить кэш (по умолчанию выключен).
- `ttl` — время жизни записи (формат Go `time.ParseDuration`: `30s`, `10m`, `2h`; по умолчанию `10m`).

Пароль сохраняется в **OS-keyring** через Secret Service (gnome-keyring / KWallet), управляется только через конфиг (нет CLI-флагов и env). Сбросить кэш: `kdbx-cli forget` (очищает записи для всех `.kdbx` из конфига).

**Чем это управляется и риски:**

- Кэшируется **master-пароль** `.kdbx` — он открывает **всю** базу, а не только инжектируемые переменные. Это более ценная цель, чем окружение дочернего процесса.
- В keyring пароль зашифрован на диске и расшифрован в памяти демона на время login-сессии; просмотреть/отозвать можно в **seahorse** («Пароли и ключи»).
- Базовый Secret Service **не имеет** разграничения по приложениям: любой процесс под тем же пользователем может прочитать запись (та же граница доверия, что и `/proc/<pid>/environ`).
- Если Secret Service недоступен (headless, SSH, нет session-bus) — кэш просто не работает, пароль спрашивается как обычно (команда не падает).

## Ограничения безопасности

Секрет никогда не попадает в командную строку (`/proc/<pid>/cmdline` читается любым процессом) и не пишется на диск в открытом виде: файловый режим использует анонимный файл в памяти, askpass — FIFO в каталоге с правами `0700`.

Но в режиме `secrets` `kdbx-cli` инжектит секреты в окружение дочернего процесса, а env-переменные процесса доступны через `/proc/<pid>/environ` тому же пользователю (и root). Это общий компромисс всего класса инструментов (`op run`, `envchain`, `aws-vault`) — но это радикально безопаснее, чем хранить секреты в истории shell или в plaintext-файлах. Если нужна защита от чтения окружения соседними процессами того же пользователя — этот подход (как и аналоги) не подходит.

## Разработка

```sh
make build       # собрать ./kdbx-cli
make unit-test   # юнит-тесты
make e2e-test    # e2e (нужна keepassxc-cli)
make test        # всё вместе
```
