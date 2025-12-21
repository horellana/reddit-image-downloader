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

func ListAvailableImages(subredditUrl string, filterMature bool) ([]ImageUrl, error) {
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

  var result []ImageUrl
  suffixes := []string{".jpeg",".png",".jpg"}

  for _, child := range subredditImages.Data.Children {
    for _, suffix := range suffixes {
      if child.Data.Mature && filterMature {
	break
      }

      if strings.HasSuffix(child.Data.Url, suffix) {
	result = append(result, child)
	break
      }
    }
  }

  return result, nil
}

func GenerateOutputPath(imageUrl string, rootFolder string, subreddit string) (string, error) {
  var parsedUrl, err = url.Parse(imageUrl)

  if err != nil {
    return "", err
  }

  var urlPaths = strings.Split(parsedUrl.Path, "/")
  filename := urlPaths[len(urlPaths) - 1]
  extension := filepath.Ext(filename)
  name := strings.TrimSuffix(filename, extension)
  newFilename := name + "_" + subreddit + extension

  return path.Join(rootFolder, newFilename), nil
}

func DownloadImage(imageUrl string, rootFolder string, subreddit string) (string, error) {
  response, requestError := http.Get(imageUrl)

  if requestError != nil {
    return "", requestError
  }

  defer response.Body.Close()

  outputPath, outputPathError := GenerateOutputPath(imageUrl, rootFolder, subreddit)

  if outputPathError != nil {
    return "", outputPathError
  }

  file, fileCreateError := os.Create(outputPath)

  if fileCreateError != nil {
    return "", fileCreateError
  }

  defer file.Close()

  _, fileWriteError := io.Copy(file, response.Body)

  if fileWriteError != nil {
    return "", fileWriteError
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

    _, err := DownloadImage(imageUrl.Data.Url, rootFolder, imageUrl.Data.Subreddit)

    if err != nil {
	    slog.Error(MSG_DOWNLOAD_FAILED, "url", imageUrl.Data.Url, "error", err)
    }
	
    wg.Done()
  }
}

func DownloadSubredditWorker(subredditChannel chan string, imagesChannel chan ImageUrl, wg *sync.WaitGroup, rootFolder string, allowMature bool, sort string, timeRange string) {
  for {
    subreddit := <-subredditChannel

    var imageUrls, err = ListAvailableImages(GenerateUrl(subreddit, sort, timeRange), allowMature)

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

  flag.Parse()

  var wg sync.WaitGroup
  imagesChannel := make(chan ImageUrl, 100)
  subredditChannel := make(chan string)
  workerCount := (runtime.NumCPU() * 2) + 1

  for i := 0; i < workerCount; i++ {
    go DownloadImageWorker(imagesChannel, &wg, *outputFolderArg)
  }

  slog.Info(MSG_WORKER_IMG_STARTED, "count", workerCount)

  for i := 0; i < workerCount; i++ {
    go DownloadSubredditWorker(subredditChannel, imagesChannel, &wg, *outputFolderArg, !*matureArg, *sortArg, *timeArg)
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
