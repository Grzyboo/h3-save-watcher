# What is this project?
It uploads game saves of `HOMM3: Horn of the Abyss` onto a parser server.

## Why?
Coming soon :)

--- 

# for developers
Application is written in `Go` + `fyne` as UI framework. The minimal setup is Go installation.

## Go
Install [GoLang](https://go.dev/).
To verify installation, run `go version` in your terminal. You should see the installed version of Go.

## cross compilation (optional)
In order to be able to create cross-release builds (binaries for Windows+MacOS+Linux), you need to have the following tools installed:
- `go install fyne.io/fyne/v2/cmd/fyne@latest`
- `go install github.com/fyne-io/fyne-cross@latest`
- docker (for cross-compilation)
- make sure `fyne-cross` is accessible (in your `$PATH`). It will probably be in `~/go/bin/fyne-cross` if you installed it with `go install`.

## build project
- `./build.sh` - local build
- `RELEASE=1 ./build.sh` - release build (cross-compiles for all platforms and generates checksums)

## .env file
`.env.example` shows a template of the `.env` file. You can copy it and fill in the values.
API_USER nad API_PASSWORD are secret by default. To be released later.

> [!WARNING]
> To avoid auto-update of a locally built application, version in .env needs to contain "-SNAPSHOT", e.g. "VERSION=v0.1.0-SNAPSHOT"
