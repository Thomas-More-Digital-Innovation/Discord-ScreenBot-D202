# Discord ScreenBot (AMX DVX-3150HD-SP)

A Discord bot and CLI utility written in Go (using [Cobra](https://github.com/spf13/cobra), [Viper](https://github.com/spf13/viper), and [DiscordGo](https://github.com/bwmarrin/discordgo)) to control and route video inputs and outputs on an **AMX Enova DVX-3150HD-SP** presentation switcher / video matrixer.

---

## Features

- **Discord Slash Commands**:
  - `/video screen:<output> input:<source>`: Route any video source to a screen output with interactive choices (including `all` to route to all screens simultaneously).
  - `/screens`: Display all configured outputs and inputs along with their hardware port mappings.
  - `/ping`: Check connectivity and response latency to both Discord Gateway and the AMX matrix switcher.
- **Friendly Aliasing**: Route video using custom aliases (e.g. `all`, `screena`, `screenb`, `laptop`, `pc`, `appletv`) or direct port numbers (e.g. `1`, `2`, `4`).
- **CLI Management**:
  - `screenbot switch <screen> <input>`: Switch inputs directly from the terminal without Discord (e.g. `screenbot switch all input1`).
  - `screenbot screens`: Inspect configured input and output mappings.
  - `screenbot ping`: Test network connection to the AMX matrixer.
  - `screenbot run`: Start the Discord bot daemon.
- **Robust Networking**: Defers Discord interactions to prevent timeouts while waiting for matrixer hardware HTTP responses.
- **Docker Ready**: Minimal multi-stage Alpine Docker image and Docker Compose support.

---

## Project Structure

```
├── code/
│   ├── cmd/                   # Cobra CLI commands (root, run, switch, ping, screens)
│   ├── internal/
│   │   ├── amx/               # AMX DVX-3150HD-SP HTTP AJAX client
│   │   ├── bot/               # Discord bot and slash command handlers
│   │   └── config/            # Viper configuration and alias resolver
│   ├── config.example.yaml    # Example configuration template
│   ├── Dockerfile             # Multi-stage Docker container
│   ├── docker-compose.yml     # Container orchestration
│   ├── Makefile               # Development shortcuts
│   ├── go.mod
│   └── main.go                # Application entrypoint
└── README.md
```

---

## Configuration

Copy `code/config.example.yaml` to `code/config.yaml`:

### Cloudflare Access / Reverse Proxy Authentication

If the AMX switcher is protected behind **Cloudflare Zero Trust / Cloudflare Access** (e.g. `https://amx.digitalinnovation.be`):
1. In Cloudflare Zero Trust: create a **Service Token** (Access -> Service Auth -> Service Tokens) and assign it to the application's policy.
2. In `config.yaml`, add the service token headers:
   ```yaml
   amx:
     host: "https://amx.digitalinnovation.be"
     headers:
       CF-Access-Client-Id: "YOUR_SERVICE_TOKEN_CLIENT_ID"
       CF-Access-Client-Secret: "YOUR_SERVICE_TOKEN_CLIENT_SECRET"
   ```
   *(Alternatively, for quick temporary testing, you can pass your browser's `Cookie: "CF_Authorization=..."`)*.

### Environment Variable Overrides

Configuration settings can also be set via environment variables:
- `SCREENBOT_DISCORD_TOKEN` or `DISCORD_TOKEN`
- `SCREENBOT_DISCORD_GUILD_ID` or `DISCORD_GUILD_ID`
- `SCREENBOT_DISCORD_CHANNEL_ID` or `DISCORD_CHANNEL_ID`
- `SCREENBOT_AMX_HOST` or `AMX_HOST`
- `SCREENBOT_AMX_TIMEOUT`

---

## Getting Started

### Prerequisites

- Go 1.24+ (or Docker)
- Network access to the AMX DVX-3150HD-SP switcher (default IP `192.168.1.73`)
- A Discord Bot Token with the `applications.commands` and `bot` scopes

### Building and Running Locally

```bash
cd code

# Run all unit tests
make test

# Build the executable
make build

# Verify your configuration mappings
./screenbot screens --config config.example.yaml

# Test network connectivity to the AMX switcher
./screenbot ping --config config.yaml

# Switch directly via CLI
./screenbot switch screena input2 --config config.yaml

# Start the Discord Bot daemon
./screenbot run --config config.yaml
```

---

## Running with Docker

```bash
cd code

# Create your production config
cp config.example.yaml config.yaml
# (edit config.yaml with your bot token)

# Launch using docker-compose
docker compose up -d
```

---

## Discord Bot Setup Guide

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications) and create a **New Application**.
2. Navigate to **Bot** -> click **Reset Token** to copy your bot token into `config.yaml`.
3. Under **OAuth2** -> **URL Generator**:
   - Select scopes: `bot` and `applications.commands`.
   - Select bot permissions: `Send Messages`, `Embed Links`, `Use Slash Commands`.
4. Copy the generated invite link to invite the bot to your server.
5. *(Optional but Recommended)* Add your Discord server ID to `guild_id` in `config.yaml` to register slash commands instantly without waiting for Discord's global propagation delay (~1 hour).

---

## AMX DVX-3150HD-SP HTTP Control Protocol

The bot communicates with the AMX DVX-3150HD-SP internal web server via its AJAX dashboard control endpoint:

- **Endpoint**: `POST /web/module/DVX-Switcher-Dashboard/com.amx.dvx/hcontrol`
- **Headers**:
  - `Content-Type: application/x-www-form-urlencoded; charset=UTF-8`
  - `X-Requested-With: XMLHttpRequest`
- **Payload**:
  ```
  set {"path":"/switcher/<OUTPUT>/video/input","value":"<INPUT>"}
  ```
  *(e.g., `set {"path":"/switcher/1/output/video/input","value":"4"}` routes input 4 to output 1)*

---

## License

[MIT License](LICENSE)
