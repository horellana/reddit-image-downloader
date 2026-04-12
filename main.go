package main

import (
  "io"
  "os"
  "log/slog"
  "flag"
  "sync"
  "path"
  "strings"
  "runtime"
  "net/url"
  "net/http"
  "errors"
  "crypto/md5"
  "encoding/hex"
  "path/filepath"
  "encoding/json"
)

const (
	MSG_STARTUP                    = "STARTING_REDDIT_WALLPAPER_DOWNLOAD"
	MSG_WORKER_IMG_STARTED         = "STARTED_IMAGE_DOWNLOADER_WORKERS"
	MSG_WORKER_SUB_STARTED         = "STARTED_SUBREDDIT_DOWNLOADER_WORKERS"
	MSG_WORKER_GOT_IMAGE           = "WORKER_GOT_IMAGE"
	MSG_DOWNLOAD_FAILED            = "FAILED_TO_DOWNLOAD_IMAGE"
	MSG_SUBREDDIT_LIST_FAILED      = "COULD_NOT_LIST_IMAGES_FOR_SUBREDDIT"
	MSG_REMOVING_DUPLICATE         = "REMOVING_DUPLICATE_FILE"
	MSG_ERROR_REMOVING_FILE        = "ERROR_REMOVING_FILE"
)

type ImageUrl struct {
  Data struct {
    Title string `json:"title"`
    Subreddit string `json:"subreddit"`
    Mature bool `json:"over_18"`
    Url string `json:"url"`
    Preview struct {
        Images []struct {
            Source struct {
                Width int `json:"width"`
                Height int `json:"height"`
            } `json:"source"`
        } `json:"images"`
    } `json:"preview"`
  } `json:"data"`
}

type SubredditImages struct {
  Kind string `json:"kind"`
  Data struct {
    Children []ImageUrl `json:"children"`
  } `json:"data"`
}

func GenerateUrl(subreddit string, sort string, timeRange string) string {
  if sort == "" {
    sort = "top"
  }
  
  baseUrl := "https://www.reddit.com/r/" + subreddit + "/" + sort + ".json"

  if sort == "top" && timeRange != "" {
    return baseUrl + "?t=" + timeRange
  }

  return baseUrl
}

func CheckResolution(width int, height int, minRes string) (bool, bool) {
	switch minRes {
	case "1080p":
		return width >= 1920 && height >= 1080, true
	case "2k":
		return width >= 2560 && height >= 1440, true
	case "4k":
		return width >= 3840 && height >= 2160, true
	case "none", "":
		return true, true
	default:
		return false, false
	}
}

func FilterImages(children []ImageUrl, filterMature bool, minRes string) []ImageUrl {
	var result []ImageUrl
	suffixes := []string{".jpeg", ".png", ".jpg"}

	for _, child := range children {
		if child.Data.Mature && filterMature {
			continue
		}

		hasCorrectSuffix := false
		for _, suffix := range suffixes {
			if strings.HasSuffix(strings.ToLower(child.Data.Url), suffix) {
				hasCorrectSuffix = true
				break
			}
		}

		if !hasCorrectSuffix {
			continue
		}

		if len(child.Data.Preview.Images) > 0 {
			width := child.Data.Preview.Images[0].Source.Width
			height := child.Data.Preview.Images[0].Source.Height
			meetsRes, _ := CheckResolution(width, height, minRes)
			if !meetsRes {
				continue
			}
		} else if minRes != "" && minRes != "none" {
			// Skip if min resolution requested but metadata is missing
			continue
		}

		result = append(result, child)
	}

	return result
}

func ListAvailableImages(subredditUrl string, filterMature bool, minRes string) ([]ImageUrl, error) {
	var response, getRequestError = http.Get(subredditUrl)

	if getRequestError != nil {
		return nil, getRequestError
	}

	defer response.Body.Close()

	var subredditImages SubredditImages
	var jsonDecodeError = json.NewDecoder(response.Body).Decode(&subredditImages)

	if jsonDecodeError != nil {
		return nil, jsonDecodeError
	}

	return FilterImages(subredditImages.Data.Children, filterMature, minRes), nil
}

func GetResolutionCategory(width int, height int) string {
	if width >= 3840 && height >= 2160 {
		return "4k"
	}
	if width >= 2560 && height >= 1440 {
		return "2k"
	}
	if width >= 1920 && height >= 1080 {
		return "1080p"
	}
	return "other"
}

func GetMinFileSize(width int, height int) int64 {
	if width == 0 || height == 0 {
		return 100000 // 100 KB
	}
	pixels := int64(width) * int64(height)
	return pixels / 10
}

func GenerateOutputPath(imageUrl string, rootFolder string, subreddit string, width int, height int) (string, error) {
  var parsedUrl, err = url.Parse(imageUrl)

  if err != nil {
    return "", err
  }

  var urlPaths = strings.Split(parsedUrl.Path, "/")
  filename := urlPaths[len(urlPaths) - 1]
  extension := filepath.Ext(filename)
  name := strings.TrimSuffix(filename, extension)
  newFilename := name + "_" + subreddit + extension

  resolutionCategory := GetResolutionCategory(width, height)

  return path.Join(rootFolder, resolutionCategory, newFilename), nil
}

func DownloadImage(imageUrl string, rootFolder string, subreddit string, width int, height int) (string, error) {
  response, requestError := http.Get(imageUrl)

  if requestError != nil {
    return "", requestError
  }

  defer response.Body.Close()

  minSize := GetMinFileSize(width, height)

  if response.ContentLength > 0 && response.ContentLength < minSize {
      return "", errors.New("image rejected: file size too small for resolution (likely low quality or upscaled)")
  }

  outputPath, outputPathError := GenerateOutputPath(imageUrl, rootFolder, subreddit, width, height)

  if outputPathError != nil {
    return "", outputPathError
  }

  // Ensure resolution directory exists
  dir := filepath.Dir(outputPath)
  if err := os.MkdirAll(dir, 0755); err != nil {
    return "", err
  }

  file, fileCreateError := os.Create(outputPath)

  if fileCreateError != nil {
    return "", fileCreateError
  }

  defer file.Close()

  bytesWritten, fileWriteError := io.Copy(file, response.Body)

  if fileWriteError != nil {
    return "", fileWriteError
  }

  if bytesWritten < minSize {
      os.Remove(outputPath)
      return "", errors.New("image rejected: downloaded file size too small for resolution (likely low quality or upscaled)")
  }

  return outputPath, nil
}

func GetFileMD5(filePath string) (string, error) {
  file, err := os.Open(filePath)
  if err != nil {
    return "", err
  }
  defer file.Close()

  hash := md5.New()
  if _, err := io.Copy(hash, file); err != nil {
    return "", err
  }

  hashInBytes := hash.Sum(nil)
  hashString := hex.EncodeToString(hashInBytes)

  return hashString, nil
}

func RemoveDuplicateFiles(folderPath string) error {
  fileChecksumMap := make(map[string][]string)

  err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    }
    if !info.IsDir() {
      md5sum, err := GetFileMD5(path)
      if err != nil {
	return err
      }
      fileChecksumMap[md5sum] = append(fileChecksumMap[md5sum], path)
    }
    return nil
  })
  if err != nil {
    return err
  }

  for _, paths := range fileChecksumMap {
    if len(paths) > 1 {
      // Remove duplicates by keeping only the first occurrence
      paths = paths[1:]
      for _, path := range paths {
	slog.Info(MSG_REMOVING_DUPLICATE, "path", path)
	err := os.Remove(path)
	if err != nil {
	  slog.Error(MSG_ERROR_REMOVING_FILE, "path", path, "error", err)
	}
      }
    }
  }

  return nil
}

func DownloadImageWorker(imagesChannel chan ImageUrl, wg *sync.WaitGroup, rootFolder string) {
  for {
    imageUrl := <-imagesChannel
    slog.Info(MSG_WORKER_GOT_IMAGE, "subreddit", imageUrl.Data.Subreddit, "url", imageUrl.Data.Url)

    width := 0
    height := 0
    if len(imageUrl.Data.Preview.Images) > 0 {
      width = imageUrl.Data.Preview.Images[0].Source.Width
      height = imageUrl.Data.Preview.Images[0].Source.Height
    }

    _, err := DownloadImage(imageUrl.Data.Url, rootFolder, imageUrl.Data.Subreddit, width, height)

    if err != nil {
	    slog.Error(MSG_DOWNLOAD_FAILED, "url", imageUrl.Data.Url, "error", err)
    }
	
    wg.Done()
  }
}

func DownloadSubredditWorker(subredditChannel chan string, imagesChannel chan ImageUrl, wg *sync.WaitGroup, rootFolder string, allowMature bool, sort string, timeRange string, minRes string) {
  for {
    subreddit := <-subredditChannel

    var imageUrls, err = ListAvailableImages(GenerateUrl(subreddit, sort, timeRange), allowMature, minRes)

    if err != nil {
      slog.Error(MSG_SUBREDDIT_LIST_FAILED, "subreddit", subreddit, "error", err)
      wg.Done()
      continue
    }

    for _, imageUrl := range imageUrls {
      wg.Add(1)
      imagesChannel <- imageUrl
    }

    wg.Done()
  }
}

func main() {
  handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        if a.Key == slog.TimeKey {
	    return slog.Attr{
		Key:   a.Key,
		Value: slog.StringValue(a.Value.Time().Format("15:04:05")),
	    }
        }
        return a
    },
  })
  slog.SetDefault(slog.New(handler))

  slog.Info(MSG_STARTUP, "version", 1)
  subredditsArg := flag.String("subreddits", "wallpapers,WQHD_Wallpaper", "Comma separated list of subreddit names")
  outputFolderArg := flag.String("folder", "/tmp", "Path where to download images")
  matureArg := flag.Bool("mature", true, "Allow mature content")
  sortArg := flag.String("sort", "top", "Sort order (top, hot, new, rising)")
  timeArg := flag.String("time", "all", "Time range for top sort (hour, day, week, month, year, all)")
  minResArg := flag.String("min-res", "1080p", "Minimum resolution (1080p, 2k, 4k, none)")

  flag.Parse()

  // Validate minResArg
  if _, isValid := CheckResolution(0, 0, *minResArg); !isValid {
    slog.Error("Invalid -min-res value. Choose from: 1080p, 2k, 4k, none")
    os.Exit(1)
  }

  var wg sync.WaitGroup
  imagesChannel := make(chan ImageUrl, 100)
  subredditChannel := make(chan string)
  workerCount := (runtime.NumCPU() * 2) + 1

  for i := 0; i < workerCount; i++ {
    go DownloadImageWorker(imagesChannel, &wg, *outputFolderArg)
  }

  slog.Info(MSG_WORKER_IMG_STARTED, "count", workerCount)

  for i := 0; i < workerCount; i++ {
    go DownloadSubredditWorker(subredditChannel, imagesChannel, &wg, *outputFolderArg, !*matureArg, *sortArg, *timeArg, *minResArg)
  }

  slog.Info(MSG_WORKER_SUB_STARTED, "count", workerCount)


  subreddits := strings.Split(*subredditsArg, ",")

  for _, subreddit := range subreddits {
    wg.Add(1)
    subredditChannel <- subreddit
  }

  wg.Wait()

  RemoveDuplicateFiles(*outputFolderArg)
}
