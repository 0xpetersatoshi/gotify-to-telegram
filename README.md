# gotify-to-telegram

This plugin routes [Gotify](https://gotify.net/) messages to [Telegram](https://telegram.org/).

## Features

- Support for forwarding messages to multiple telegram bots/chat ids
- Configurable message formatting options per bot
- Configuration via environment variables or yaml config file via Gotify UI

## Installation

### Download pre-built plugins

You can download pre-built plugins for your specific architecture and gotify version from the
[releases page](https://github.com/0xpetersatoshi/gotify-to-telegram/releases).

### Build from source

Clone the repository and run:

```bash
make GOTIFY_VERSION="v2.6.1" FILE_SUFFIX="for-gotify-v2.6.1" build
```

> **Note**: Specify the `GOTIFY_VERSION` you want to build the plugin for.

### Gotify Setup

Copy the plugin shared object file into your Gotify plugins directory (configured as `pluginsdir` in your Gotify
config file). Further documentation can be found [here](https://gotify.net/docs/plugin-deploy#deploying).

## Configuration

### Prequisites

There are four required configuration settings needed to start using this plugin:

1. A Telegram bot token
2. A Telegram chat id
3. A Gotify server url
4. A Gotify server client token

#### Telegram

By default, all gotify messages will be sent to this default bot, though you can configure multiple different bots and
specify which gotify messages are routed to which bot. You can read
[this](https://sendpulse.com/knowledge-base/chatbot/telegram/create-telegram-chatbot#create-bot) for more info on how
to create a telegram bot.

#### Gotify

Additionally, the plugin needs the gotify server url and a client token to be able to create a websocket connection to
the gotify server and listen for new messages. A client token can be created in the Gotify UI from the "Clients" tab.

### Getting Started

You can configure the plugin in one of the following ways:

1. Only using environment variables (limited configuration options)
2. Only using the yaml editor from the Gotify UI (full configuration options) accessible from the Plugins > Details >
   Configurer section
3. Using both environment variables and the yaml editor with environment variables taking precedence over any values set
   in the yaml editor. You can later decide to ignore the environment variables and only use the yaml editor by either
   unsetting the environment variables or setting the option `ignore_env_vars: true` in the yaml editor.

#### Environment Variables

The plugin can be configured using environment variables. All variables are prefixed with `TG_PLUGIN__`
(note the double underscore!).

##### Logging Settings

| Variable               | Type   | Default  | Description                                  |
| ---------------------- | ------ | -------- | -------------------------------------------- |
| `TG_PLUGIN__LOG_LEVEL` | string | `"info"` | Log level (`debug`, `info`, `warn`, `error`) |

##### Gotify Server Settings

| Variable                         | Type   | Default                 | Description                          |
| -------------------------------- | ------ | ----------------------- | ------------------------------------ |
| `TG_PLUGIN__GOTIFY_URL`          | string | `"http://localhost:80"` | URL of your Gotify server (required) |
| `TG_PLUGIN__GOTIFY_CLIENT_TOKEN` | string | `""`                    | Client token from Gotify (required)  |

##### Telegram Bot Settings

| Variable                                | Type   | Default | Description                                 |
| --------------------------------------- | ------ | ------- | ------------------------------------------- |
| `TG_PLUGIN__TELEGRAM_DEFAULT_BOT_TOKEN` | string | `""`    | Default Telegram bot token (required)       |
| `TG_PLUGIN__TELEGRAM_DEFAULT_CHAT_IDS`  | string | `""`    | Comma-separated list of chat IDs (required) |

##### Message Formatting Settings

| Variable                                | Type    | Default        | Description                                  |
| --------------------------------------- | ------- | -------------- | -------------------------------------------- |
| `TG_PLUGIN__MESSAGE_INCLUDE_APP_NAME`   | boolean | `false`        | Include Gotify app name in the message title |
| `TG_PLUGIN__MESSAGE_INCLUDE_TIMESTAMP`  | boolean | `false`        | Include timestamp                            |
| `TG_PLUGIN__MESSAGE_INCLUDE_EXTRAS`     | boolean | `false`        | Include message extras                       |
| `TG_PLUGIN__MESSAGE_PARSE_MODE`         | string  | `"MarkdownV2"` | Message parse mode                           |
| `TG_PLUGIN__MESSAGE_INCLUDE_PRIORITY`   | boolean | `false`        | Show priority indicators emojis              |
| `TG_PLUGIN__MESSAGE_PRIORITY_THRESHOLD` | integer | `0`            | Priority indicator threshold                 |

##### Priority Indicators

When `TG_PLUGIN__MESSAGE_INCLUDE_PRIORITY` is enabled, messages include these indicator emojis based on priority:

- 🔴 Critical Priority (≥8)
- 🟠 High Priority (≥6)
- 🟡 Medium Priority (≥4)
- 🟢 Low Priority (<4)

##### Example Configuration

```env
# Logging
TG_PLUGIN__LOG_LEVEL=debug

# Gotify Server
TG_PLUGIN__GOTIFY_URL="http://gotify.example.com"
TG_PLUGIN__GOTIFY_CLIENT_TOKEN="ABC123..."

# Telegram Settings
TG_PLUGIN__TELEGRAM_DEFAULT_BOT_TOKEN="123456:ABC-DEF..."
TG_PLUGIN__TELEGRAM_DEFAULT_CHAT_IDS="123456789,987654321"

# Message Formatting
TG_PLUGIN__MESSAGE_INCLUDE_APP_NAME=true
TG_PLUGIN__MESSAGE_INCLUDE_TIMESTAMP=true
TG_PLUGIN__MESSAGE_INCLUDE_EXTRAS=false
TG_PLUGIN__MESSAGE_INCLUDE_PRIORITY=true
TG_PLUGIN__MESSAGE_PRIORITY_THRESHOLD=5

```

#### Yaml configuration

You can also configure the plugin using the yaml editor from the Gotify UI. This unlocks more granular configuration
including specifying multiple telegram bots, differing formatting options for each bot, and specific Gotify application
IDs that should be routed to a specific bot.

Here is an example yaml configuration:

```yaml
settings:
  ignore_env_vars: false
  log_options:
    log_level: debug
  gotify_server:
    url: http://localhost:80
    client_token: CzV6.mP4r3r1yoA
    websocket:
      handshake_timeout: 10
  telegram:
    default_bot_token: 123456789:ABC-DEF-GHI-JKL-MNO
    default_chat_ids:
      - "123456789"
      - "987654321"
    bots:
      example_bot:
        token: 987654321:XYZ-ABC-DEF-GHI-JKL-MNO
        chat_ids:
          - "445566778"
          - "223344556"
        gotify_app_ids:
          - 10
          - 23
        message_format_options:
          include_app_name: true
          include_timestamp: true
          include_extras: false
          parse_mode: MarkdownV2
          include_priority: false
          priority_threshold: 0
      another_bot:
        token: 678901234:JKL-MNO-PQR-STU-VWX
        chat_ids:
          - "889900112"
        gotify_app_ids:
          - 5
          - 6
          - 7
        message_format_options:
          include_app_name: false
          include_timestamp: true
          include_extras: false
          parse_mode: MarkdownV2
          include_priority: true
          priority_threshold: 4
    default_message_format_options:
      include_app_name: false # example: [Jellyseer] Movie Request Approved
      include_timestamp: false
      include_extras: false
      parse_mode: MarkdownV2
      include_priority: false
      priority_threshold: 0
```

In this example, there are two additional bots configured: `example_bot` and `another_bot`. Both bots have different
message formatting options. Messages from gotify application IDs 5, 6, and 7 will be sent to the `another_bot` and
messages from gotify application IDs 10 and 23 will be sent to the `example_bot`. All other messages will be sent to
the default bot.

## Development

You can run and test this plugin in a docker container by running:

```bash

# if you are on an arm machine
make test-plugin-arm64

# if you are on a x86 machine
make test-plugin-amd64
```

This will build the shared objects file for your architecture and spin up a gotify docker container with the plugin
loaded onto it.

> **NOTE**: The gotify docker container uses the default username and password (admin/admin).

Additionally, you can run tests with:

```bash
make test
```
