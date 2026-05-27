# QQMusic Navidrome Plugin

A Navidrome plugin that fetches artist and album artwork from QQ Music.

## Building

1. Install [TinyGo](https://tinygo.org/getting-started/install/)
2. Build the plugin:
   ```bash
   go mod tidy
   tinygo build -o qqmusic.wasm -target wasip1 -buildmode=c-shared .
   zip -j qqmusic.ndp manifest.json qqmusic.wasm
   ```

## Installing

1. Copy `qqmusic.ndp` (or `qqmusic.wasm`) to your Navidrome plugins folder (default: `<data-folder>/plugins/`)
2. Enable plugins in your `navidrome.toml`:
   ```toml
   [Plugins]
   Enabled = true

   # Add the plugin to your agents list
   Agents = "lastfm,deezer,qqmusic"
   ```

## Features

- **Artist Images**: Fetches artist profile images from QQ Music
- **Album Images**: Fetches album cover images from QQ Music
- **Multiple Sizes**: Returns images in 150px, 300px, 500px, and 800px resolutions
- **Fallback Support**: Works alongside other artwork agents (LastFM, Deezer, etc.)

## How It Works

The plugin searches QQ Music's API by artist name or album name, retrieves the media ID (mid), and constructs image URLs using QQ Music's standard `photo_new` URL format:

```
https://y.gtimg.cn/music/photo_new/T001R300M000{singer_mid}.jpg
https://y.gtimg.cn/music/photo_new/T002R300M000{album_mid}.jpg
```

## Configuration

No additional configuration is required. The plugin uses default QQ Music search API.

## Artwork Priority

To use QQ Music as an artwork source, ensure it is listed in your `Agents` configuration. The order determines priority:

```toml
Agents = "qqmusic,lastfm,deezer"  # QQ Music tried first
```

## Development

### Project Structure

```
├── main.go         # Plugin source code
├── manifest.json   # Plugin metadata
├── go.mod         # Go module definition
└── qqmusic.wasm   # Compiled WASM binary
```

### PDK Usage

```go
import "github.com/navidrome/navidrome/plugins/pdk/go/metadata"

type qqmusicPlugin struct{}

func init() {
    metadata.Register(&qqmusicPlugin{})
}

func (*qqmusicPlugin) GetArtistImages(input metadata.ArtistRequest) (*metadata.ArtistImagesResponse, error) {
    // Search QQ Music and return images
}

func (*qqmusicPlugin) GetAlbumImages(input metadata.AlbumRequest) (*metadata.AlbumImagesResponse, error) {
    // Search QQ Music and return images
}
```

## Supported Interfaces

- `metadata.ArtistImagesProvider` - Get artist images
- `metadata.AlbumImagesProvider` - Get album images

## License

MIT