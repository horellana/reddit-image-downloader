# Reddit Wallpaper Downloader

A high-performance command-line tool written in Go to efficiently download wallpapers and images from your favorite subreddits.

## Features

*   **Concurrent Downloading:** Utilizes Go routines and channels to download multiple images in parallel, maximizing bandwidth usage.
*   **Smart Filtering:** Automatically filters for standard image formats (`.jpeg`, `.png`, `.jpg`), handles mature content settings, and allows filtering by minimum resolution (1080p, 2k, 4k).
*   **Filename Tagging:** Appends the source subreddit name to the filename (e.g., `image_SpacePorn.jpg`) to easily identify the source and prevent name collisions.
*   **Deduplication:** Includes a built-in post-processing step to detect and remove duplicate files based on MD5 checksums, keeping your wallpaper collection clean.

## Prerequisites

*   [Go](https://go.dev/dl/) (version 1.21.6 or later recommended)

## Installation & Building

1.  Clone the repository:
    ```bash
    git clone <repository-url>
    cd reddit-wallpaper-download
    ```

2.  Build the binary:
    ```bash
    go build -o reddit-wallpaper-download
    ```

## Usage

Run the downloaded binary with the following flags to customize your download:

```bash
./reddit-wallpaper-download [flags]
```

### Available Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-subreddits` | Comma-separated list of subreddits to scrape. | `wallpapers,WQHD_Wallpaper` |
| `-folder` | Destination directory for downloaded images. | `/tmp` |
| `-mature` | Allow mature (over 18) content. Set to `false` to block. | `true` |
| `-sort` | Sort order for reddit posts (`top`, `hot`, `new`, `rising`). | `top` |
| `-time` | Time range for `top` sort (`hour`, `day`, `week`, `month`, `year`, `all`). | `all` |
| `-min-res` | Minimum resolution filter (`1080p`, `2k`, `4k`, `none`). | `1080p` |

### Example

Download 4K wallpapers from specific subreddits, sorting by the top posts of the month:

```bash
./reddit-wallpaper-download -subreddits="SpacePorn,EarthPorn" -folder="./wallpapers" -min-res="4k" -sort="top" -time="month"
```

## License

[GPLv3](LICENSE)
