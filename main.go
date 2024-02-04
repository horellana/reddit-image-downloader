package main

import (
  "io"
  "os"
  "log"
  "fmt"
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

func GenerateUrl(subreddit string) string {
  return fmt.Sprintf("https://www.reddit.com/r/%s.json", subreddit)
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

func GenerateOutputPath(imageUrl string, rootFolder string) (string, error) {
  var parsedUrl, err = url.Parse(imageUrl)

  if err != nil {
    return "", err
  }

  var urlPaths = strings.Split(parsedUrl.Path, "/")

  return path.Join(rootFolder, urlPaths[len(urlPaths) - 1]), nil
}

func DownloadImage(imageUrl string, rootFolder string) (string, error) {
  response, requestError := http.Get(imageUrl)

  if requestError != nil {
    return "", requestError
  }

  defer response.Body.Close()

  outputPath, outputPathError := GenerateOutputPath(imageUrl, rootFolder)

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
	fmt.Printf("Removing duplicate file: %s\n", path)
	err := os.Remove(path)
	if err != nil {
	  fmt.Printf("Error removing file %s: %v\n", path, err)
	}
      }
    }
  }

  return nil
}

func DownloadImageWorker(imagesChannel chan ImageUrl, wg *sync.WaitGroup, rootFolder string) {
  for {
    imageUrl := <-imagesChannel
    fmt.Printf("Worker got image: (%s) %s\n", imageUrl.Data.Subreddit, imageUrl.Data.Url)

    wg.Add(1)
    DownloadImage(imageUrl.Data.Url, rootFolder)
    wg.Done()
  }
}

func DownloadSubredditWorker(subredditChannel chan string, imagesChannel chan ImageUrl, wg *sync.WaitGroup, rootFolder string, allowMature bool) {
  for {
    subreddit := <-subredditChannel

    var imageUrls, err = ListAvailableImages(GenerateUrl(subreddit), allowMature)

    if err != nil {
      log.Printf("Could not list images for subreddit: %s, error: %s", subreddit, err);
      wg.Done()
      continue
    }

    for _, imageUrl := range imageUrls {
      imagesChannel <- imageUrl
    }

    wg.Done()
  }
}

func main() {
  subredditsArg := flag.String("subreddits", "wallpapers,WQHD_Wallpaper", "Comma separated list of subreddit names")
  outputFolderArg := flag.String("folder", "/tmp", "Path where to download images")
  matureArg := flag.Bool("mature", true, "Allow mature content")

  flag.Parse()

  var wg sync.WaitGroup
  imagesChannel := make(chan ImageUrl)
  subredditChannel := make(chan string)
  workerCount := (runtime.NumCPU() * 2) + 1

  for i := 0; i < workerCount; i++ {
    go DownloadImageWorker(imagesChannel, &wg, *outputFolderArg)
  }

  fmt.Printf("Started %d image downloader workers\n", workerCount)

  for i := 0; i < workerCount; i++ {
    go DownloadSubredditWorker(subredditChannel, imagesChannel, &wg, *outputFolderArg, !*matureArg)
  }

  fmt.Printf("Started %d subreddit downloader workers\n", workerCount)

  subreddits := strings.Split(*subredditsArg, ",")

  for _, subreddit := range subreddits {
    wg.Add(1)
    subredditChannel <- subreddit
  }

  wg.Wait()

  RemoveDuplicateFiles(*outputFolderArg)
}
