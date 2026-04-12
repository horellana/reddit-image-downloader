# Reddit Wallpaper Downloader

## Project Overview

`reddit-wallpaper-download` is a high-performance CLI tool written in Go designed to download images from specified subreddits. It utilizes Go's concurrency features (goroutines and channels) to efficiently fetch image URLs and download files in parallel. The tool also includes a deduplication feature that removes identical files based on MD5 checksums.

## Key Features

*   **Concurrent Downloading:** Spawns multiple workers to handle subreddit parsing and image downloading simultaneously.
*   **Duplicate Removal:** Automatically detects and removes duplicate images in the output folder using MD5 hashing.
*   **Filename Tagging:** Appends the subreddit name to the filename (e.g. `image_SpacePorn.jpg`) to preserve context.
*   **Filtering:** Filters for common image formats (`.jpg`, `.jpeg`, `.png`), supports an optional mature content filter, and uses a file size heuristic to reject low-quality or upscaled images based on their resolution.

## Building and Running

### Prerequisites

*   Go 1.21.6 or later

### Build Command

```bash
go build -o reddit-wallpaper-download
```

### Usage

Run the compiled binary with the following flags:

```bash
./reddit-wallpaper-download [flags]
```

#### Flags

*   `-subreddits`: Comma-separated list of subreddits to download from. (Default: "wallpapers,WQHD_Wallpaper")
*   `-folder`: Path to the directory where images will be saved. (Default: "/tmp")
*   `-mature`: Boolean flag to allow mature content. (Default: true)
*   `-min-res`: Minimum resolution for downloaded wallpapers. Options: `1080p`, `2k`, `4k`, `none`. (Default: "1080p")

**Example:**

```bash
./reddit-wallpaper-download -subreddits="EarthPorn,SpacePorn" -folder="./wallpapers" -mature=false
```

## Architecture

The application logic is contained within `main.go`.

1.  **Workers:** The program starts a pool of "Subreddit Downloaders" and "Image Downloaders" based on the available CPU cores.
2.  **Pipeline:**
    *   Subreddit workers fetch JSON data from Reddit, filter for valid image URLs, and push them to an `imagesChannel`.
    *   Image workers consume from `imagesChannel` and download the files to the specified disk location.
3.  **Cleanup:** After all downloads complete, the `RemoveDuplicateFiles` function scans the output directory and deletes duplicate files based on content hash.
