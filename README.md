# Reddit Wallpaper Downloader

A high-performance command-line tool written in Go to efficiently download wallpapers and images from your favorite subreddits.

## Features

*   **Concurrent Downloading:** Utilizes Go routines and channels to download multiple images in parallel, maximizing bandwidth usage.
*   **Smart Filtering:** Automatically filters for standard image formats (`.jpeg`, `.png`, `.jpg`), handles mature content settings, and allows filtering by minimum resolution (1080p, 2k, 4k).
*   **Filename Tagging:** Appends the source subreddit name to the filename (e.g., `image_SpacePorn.jpg`) to easily identify the source and prevent name collisions.
*   **Deduplication:** Includes a built-in post-processing step to detect and remove duplicate files based on MD5 checksums, keeping your wallpaper collection clean.
*   **Systemd Integration:** Comes with ready-to-use systemd service and timer files for scheduling automatic background downloads.

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

## Automation with Systemd

This project includes systemd unit files to allow you to run the downloader automatically on a schedule (e.g., every 15 minutes).

1.  **Edit the Service File:**
    Open `wallpaper-downloader.service` and update the paths to match your installation:
    *   `WorkingDirectory`: Path to where the binary is located.
    *   `ExecStart`: Full path to the binary with your desired flags.
    *   `User`: Your system username.

2.  **Install Units:**
    Copy the files to your systemd user directory (create it if it doesn't exist):
    ```bash
    mkdir -p ~/.config/systemd/user/
    cp wallpaper-downloader.service wallpaper-downloader.timer ~/.config/systemd/user/
    ```

3.  **Enable and Start:**
    ```bash
    systemctl --user daemon-reload
    systemctl --user enable --now wallpaper-downloader.timer
    ```

## License

[GPLv3](LICENSE)
