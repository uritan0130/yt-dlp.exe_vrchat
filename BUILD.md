# Build

## Requirements

- Go 1.22+ recommended
- Windows x64 target
- No external Go modules are required

## Build on Windows

From the repository root:

```powershell
go build -trimpath -ldflags="-s -w" -o yt-dlp.exe .\src
```

## Cross-build from Linux/macOS

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o yt-dlp.exe ./src
```

## Pinned dependencies

The wrapper verifies these official release artifacts internally:

- yt-dlp `2026.08.19` (`yt-dlp.exe`) SHA-256:
  `66674953fe251b89f4d08c5f0e35e0728679bd67ab3d7d05c0562af101dd3e7a`
- Deno `2.9.5` (`deno-x86_64-pc-windows-msvc.zip`) SHA-512:
  `7c4b0701e85f105b4ad000a8cab575203c5fa6e95adc47d3f14df87b8b11f90b8d2704de824d61368b4571a03ac7ef83d49dd176fee8713bfc8c9270c4a35b92`

The source uses only Go's standard library.
