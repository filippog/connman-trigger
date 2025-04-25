# connman-trigger

`connman-trigger` is a small utility that listens for ConnMan network service
state changes over D-Bus and executes user-defined scripts in response to
connectivity events.

## Features

- Maps ConnMan states to script actions:
  - `ready`, `online` → **up** (run "up" scripts)
  - `idle`, `offline` → **down** (run "down" scripts)
- Retrieves current network SSID and connection type (e.g., `wifi`)
- Executes executable scripts in specified directories, in lexicographical order
- Exports the following environment variables to each script:
  - `NETWORK_STATE`: current state (`ready`, `online`, `idle`, or `offline`)
  - `NETWORK_SSID`: SSID or `unknown`
  - `CONNECTION_TYPE`: connection type or `unknown`

## Installation

- Requires Go 1.20 or later
- Clone the repository and build:
  ```sh
  git clone https://github.com/filippog/connman-trigger.git
  cd connman-trigger
  make
  ```

## Usage

```sh
./connman-trigger -p <up-scripts-dir> -p <down-scripts-dir>
```

- `-p <path>`: directory containing executable scripts to run on state changes. (repeated)
- Scripts run with the environment variables listed above.


## Colophon

This project is licensed under the Apache License 2.0. See the
[LICENSE](LICENSE) file for details. It has been written with help from
[codex](https://github.com/openai/codex).
