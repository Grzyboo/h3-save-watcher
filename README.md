# Prepare for local development
## install go, fyne
- `go install fyne.io/fyne/v2/cmd/fyne@latest`
- `go install github.com/fyne-io/fyne-cross@latest`
## add go bin to path
`export PATH="$HOME/go/bin:$PATH"`

# Build project
- `./build.sh` - local build
- `RELEASE=1 ./build.sh` - release build (cross-compiles for all platforms and generates checksums)
