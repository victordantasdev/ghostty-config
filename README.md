# ghostty-config

A CLI for configuring [Ghostty](https://ghostty.org) — pick GLSL shaders and color themes from a curated list, audition them live in the running terminal, and have your selections written back to `~/.config/ghostty/config` automatically. Written in Go on top of [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss), and [Cobra](https://github.com/spf13/cobra).


https://github.com/user-attachments/assets/b26f4d18-5539-43b2-9726-401752a854d9

## What it does

Ghostty supports `custom-shader` entries (GLSL post-processing effects) and `theme` entries (color palettes), but tuning them by hand means editing the config file, saving, and reloading repeatedly to compare results. `ghostty-config` replaces that loop with a single CLI that:

- **Discovers everything that is available**: it scans your local shader directory, your user themes directory, and the themes shipped inside the Ghostty application bundle on macOS. On first launch it also seeds your shader and user-theme directories with a generous starter pack of community shaders (CRT effects, glow, gradients, fireworks, starfields, matrix-style scenes, cursor warps, ripples, blazes, tails, etc.) and a `vesper` theme, extracted from binaries embedded in the program.
- **Lets you compose a shader pipeline**: in the Shaders screen you build a *global* shader chain (zero or more non-cursor GLSL files, applied in order so each shader's output feeds into the next) and pick a *cursor* shader (a single file whose name contains `cursor`). Ghostty chains both, so cursor effects compose on top of the global pipeline.
- **Lets you pair a light and a dark theme**: in the Themes screen you assign one theme to the light slot and one theme to the dark slot. Ghostty uses the appropriate slot based on the system appearance.
- **Previews changes live**: as you move through the list, each new highlight is written to the config and Ghostty is asked to reload, so the terminal you are reading this from updates instantly. Pressing Enter commits the highlighted combination; pressing Esc, `q`, or Ctrl+C restores whatever was active before you opened the CLI, so you can experiment freely without losing your previous setup.
- **Edits your config safely**: managed keys (`theme` and `custom-shader`) are written into a clearly marked section at the bottom of the file, separated by the line `# The following settings are managed by ghostty-config. Do not edit by hand.` Any earlier occurrences of those keys outside the managed section are commented out rather than deleted, and a `-bkp` copy of your original config is created on the first write.
- **Triggers a Ghostty reload after every change**: on macOS this is done by sending `Cmd+Shift+,` (Ghostty's default `reload_config` keystroke) via `osascript`. On other platforms — or whenever you prefer a different reload mechanism — you can pass an explicit shell command, or disable automatic reloads entirely.

## Requirements

- Go 1.26 or newer (to build from source).
- Ghostty installed and configured. On macOS the default behavior expects Ghostty's bundled themes at `/Applications/Ghostty.app/Contents/Resources/ghostty/themes`; the path can be overridden.
- macOS, if you want automatic reloads out of the box. On Linux and other platforms you must pass `--reload-command` or `--no-reload`.

## Installation

### Download a prebuilt binary

Each tagged release publishes self-contained binaries for **macOS (Intel and Apple Silicon)** and **Linux (amd64 and arm64)** on the [Releases page](https://github.com/victordantasdev/ghostty-config/releases). Shaders and themes are embedded in the binary via `//go:embed` and seeded to `~/.config/ghostty/` on first launch — no extra files needed.

Pick the archive that matches your platform (`darwin_arm64`, `darwin_x86_64`, `linux_arm64`, `linux_x86_64`), extract, and install:

```sh
# macOS arm64 example — adjust the URL for your platform and the latest tag
VERSION=v0.1.0
curl -sSL "https://github.com/victordantasdev/ghostty-config/releases/download/${VERSION}/ghostty-config_${VERSION#v}_darwin_arm64.tar.gz" \
  | tar -xz ghostty-config
sudo install -m 755 ghostty-config /usr/local/bin/ghostty-config
rm ghostty-config
```

Or, without `sudo`, into a user-owned directory on your `$PATH`:

```sh
mkdir -p ~/.local/bin
install -m 755 ghostty-config ~/.local/bin/ghostty-config
# make sure ~/.local/bin is on your PATH (zsh):
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

On macOS, the binaries are not codesigned or notarized, so on first launch Gatekeeper will block them with *"cannot be opened because the developer cannot be verified."* Clear the quarantine attribute once:

```sh
xattr -d com.apple.quarantine "$(command -v ghostty-config)"
```

Verify the install:

```sh
ghostty-config --version
```

### Build a local binary

```sh
make build
```

Produces `./dist/ghostty-config` in the project root. `make dev` builds and immediately runs it.

### Install with `go install`

```sh
go install ./cmd/ghostty-config
```

This places a `ghostty-config` binary in your `$GOBIN` (defaults to `$GOPATH/bin` or `$HOME/go/bin`). Make sure that directory is on your `PATH`.

### Other useful Make targets

- `make fmt` — `go fmt ./...`
- `make vet` — `go vet ./...`
- `make tidy` — `go mod tidy`
- `make clean` — remove the built binary
- `make help` — print the target list

## Running

Once installed, just run:

```sh
ghostty-config
```

You will land on a top-level menu with two entries: **Shaders** and **Themes**. Pick one with the arrow keys (or `j`/`k`, or the number `1`/`2`) and press Enter to open it. Press Esc, `q`, or Ctrl+C from the menu to quit.

The CLI uses Bubble Tea's alt-screen mode, so on exit your terminal scrollback is preserved exactly as it was before launch.

## The Shaders screen

The screen has two slots, displayed side by side:

- **Global (multi-select)**. Zero or more `.glsl` files whose name does *not* contain `cursor`. Order matters: every selected shader gets its own `custom-shader = ...` line in the config, and Ghostty chains them top-to-bottom — each shader's output becomes the next one's input. A bracket marker like `[1]`, `[2]` next to an entry shows its position in the chain.
- **Cursor (single-select)**. A single `.glsl` file whose name contains `cursor` (for example `cursor_warp.glsl`, `ripple_cursor.glsl`, `rectangle_boom_cursor.glsl`, `in-game-crt-cursor.glsl`). A `None` entry at the top of the list removes the cursor shader entirely.

All globals are written first, then the cursor shader, so the cursor effect always composes on top of the global pipeline.

### Keybindings (Shaders)

- `Tab` / `Shift+Tab` (or `Left`/`Right`, `h`/`l`): switch between the Global and Cursor slots.
- `Up`/`Down` or `j`/`k`: move the highlight inside the active slot.
  - In the Cursor slot, moving the highlight also previews the shader immediately.
  - In the Global slot, moving only navigates. Use `Space` (or `x`) to toggle whether the highlighted shader is part of the pipeline.
- `Space` / `x`: in the Global slot, add or remove the highlighted shader from the chain. The bracket index `[N]` next to the entry shows its position.
- `g` / `Home`: jump to the first entry of the active filtered list.
- `G` / `End`: jump to the last entry of the active filtered list.
- `/`: open the substring filter for the active slot.
  - Type to refine; matching is case-insensitive substring on the file name.
  - `Up`/`Down` navigates the matches while you are still typing.
  - `Backspace` deletes one character; `Ctrl+U` clears the whole query.
  - `Enter` applies the filter and returns to normal navigation.
  - `Esc` clears the filter and exits search mode.
- `Enter`: commit the current combination (global pipeline + cursor) and return to the main menu.
- `Esc`, `q`, or `Ctrl+C`: cancel and restore the shaders that were active when the screen was opened.

An empty Global selection means no global `custom-shader` line is written. Selecting `None` in the Cursor slot removes the cursor `custom-shader` line.

## The Themes screen

The screen has two slots, Light and Dark, both backed by the same combined theme list (your user themes directory plus Ghostty's bundled themes). Each entry shows whether it came from your user folder or from the Ghostty app bundle, so you can disambiguate themes with the same name.

### Keybindings (Themes)

- `Tab` / `Shift+Tab` (or `Left`/`Right`, `h`/`l`): switch between the Light and Dark slots.
- `Up`/`Down` or `j`/`k`: move the highlight. The highlighted theme is previewed live in the terminal as you move.
- `g` / `Home`, `G` / `End`: jump to first/last entry.
- `/`: open the substring filter (same behavior as in the Shaders screen).
- `Enter`: commit the current Light + Dark pair and return to the main menu.
- `Esc`, `q`, or `Ctrl+C`: cancel and restore the themes that were active when the screen was opened.

## CLI flags

```sh
ghostty-config --help
```

| Flag                 | Default                                                       | Purpose                                                                  |
|----------------------|---------------------------------------------------------------|--------------------------------------------------------------------------|
| `-c`, `--config`     | `~/.config/ghostty/config`                                    | Path to the Ghostty configuration file.                                  |
| `-s`, `--shader-dir` | `~/.config/ghostty/shaders`                                   | Directory containing `.glsl` shaders.                                    |
| `--user-theme-dir`   | `~/.config/ghostty/themes`                                    | Directory of user-defined themes.                                        |
| `--system-theme-dir` | `/Applications/Ghostty.app/Contents/Resources/ghostty/themes` | Directory of themes shipped with Ghostty.                                |
| `--no-reload`        | off                                                           | Write the config but do not trigger a Ghostty reload.                    |
| `--reload-command`   | empty                                                         | Shell command to run after each change instead of the macOS keystroke.   |

The `~` prefix is expanded automatically. Any of the directories that does not exist on first launch is created and seeded with the bundled assets.

## Live reload behavior

After every config write the program asks Ghostty to reload itself.

- On **macOS** (default): the program runs `osascript` to send `Cmd+Shift+,`, which is Ghostty's stock `reload_config` keybinding. If that keystroke fails — typically because the Accessibility permission has not been granted to the controlling terminal — the error is surfaced in the CLI status bar.
  - To grant Accessibility access on macOS, open *System Settings → Privacy & Security → Accessibility* and enable the terminal app you launch `ghostty-config` from (or Ghostty itself if you run it inside Ghostty).
- On **non-macOS systems**: the built-in path will fail with a clear error message. Use `--reload-command` or `--no-reload`.
- `--reload-command "<sh -c snippet>"`: run an arbitrary shell command after every change. Useful for window-manager-driven reload bindings, for example `--reload-command 'pkill -SIGUSR2 ghostty'` (replace with whatever reload mechanism your Ghostty setup uses).
- `--no-reload`: write the config but never reload. Pair this with manually triggering reload in Ghostty.

## How the config file is modified

`ghostty-config` only manages two keys: `theme` and `custom-shader`. Both are written into a managed section at the end of the config file, marked by:

```
# The following settings are managed by ghostty-config. Do not edit by hand.
```

Behavior in detail:

1. On the first write, the original file is copied to `<config>-bkp` (if a backup does not already exist).
2. Any pre-existing `theme = ...` or `custom-shader = ...` lines that live *above* the managed marker are commented out — never deleted — so you can recover their values if needed.
3. The managed block is rewritten from scratch on each change, in a stable order (themes first, then shaders), with one value per line.
4. If you commit an empty selection (no globals, `None` cursor, no themes), the corresponding lines are simply omitted from the managed block.

The rest of your config file is left untouched.

## Bundled assets

The program embeds a starter shader pack and a `vesper` theme using Go's `embed` package. On launch, if the target directory does not exist, it is created and populated with these files. Existing directories are never overwritten — drop in your own `.glsl` files or themes and they will appear in the lists immediately on the next launch.

## Project layout

```
.
├── assets.go                    embed.FS of bundled shaders/themes
├── cmd/ghostty-config/main.go   entry point
├── internal/
│   ├── app/                     Bubble Tea root model + main menu
│   ├── cli/                     Cobra command and option resolution
│   ├── ghostty/                 config I/O, managed-block logic, reload
│   ├── shader/                  Shaders screen (global + cursor slots)
│   ├── theme/                   Themes screen (light + dark slots)
│   └── ui/                      shared styles and screen messages
├── shaders/                     bundled GLSL starter pack
├── themes/                      bundled themes (vesper)
├── go.mod
├── go.sum
└── Makefile
```

## Troubleshooting

- **The CLI complains that reloading failed.** On macOS this almost always means Accessibility permission has not been granted to the terminal app. See the reload section above. As an immediate workaround, pass `--no-reload` and reload Ghostty by hand.
- **My selection was lost when I quit.** Esc, `q`, and Ctrl+C all restore the shaders/themes active when the screen was opened. Use `Enter` to commit before exiting a screen.
- **My custom shaders do not show up.** Make sure the files have a `.glsl` extension and live under `--shader-dir` (default `~/.config/ghostty/shaders`). Cursor shaders are identified by having `cursor` in the file name.
- **My custom themes do not show up.** Drop the theme file into `--user-theme-dir` (default `~/.config/ghostty/themes`). Ghostty's bundled themes are loaded from `--system-theme-dir` and are read-only.
- **I want to recover the original config.** A backup is kept at `<config-path>-bkp` from the first time `ghostty-config` modified the file.
