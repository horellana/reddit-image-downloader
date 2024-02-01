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
  "net/url"
  "net/http"
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

func downloadImages(images []ImageUrl, rootFolder string) {
  var wg sync.WaitGroup

  for j := 0; j < len(images); j++ {
    wg.Add(1)

    go func(image ImageUrl, rootFolder string) {
      defer wg.Done()

      _, err := DownloadImage(image.Data.Url, rootFolder)

      if err != nil {
	log.Printf("Could not download image: (r/%s) %s, error: %s", image.Data.Subreddit, image.Data.Url, err)
      } else {
	log.Printf("Downloaded (r/%s) %s", image.Data.Subreddit, image.Data.Url)
      }
    }(images[j], rootFolder)
  }

  wg.Wait()
}

func main() {
  subredditsArg := flag.String("subreddits", "wallpapers,WQHD_Wallpaper", "Comma separated list of subreddit names")
  outputFolderArg := flag.String("folder", "/tmp", "Path where to download images")
  matureArg := flag.Bool("mature", true, "Allow mature content")

  flag.Parse()

  subreddits := strings.Split(*subredditsArg, ",")

  var wg sync.WaitGroup
  for i := 0; i < len(subreddits); i++ {
    wg.Add(1)

    go func(subreddit string) {
      defer wg.Done()
      var imageUrls, err = ListAvailableImages(GenerateUrl(subreddit), !*matureArg)

      if err != nil {
	log.Printf("Could not list images for subreddit: %s, error: %s", subreddit, err);
	return
      }

      downloadImages(imageUrls, *outputFolderArg)
    }(subreddits[i])
  }

  wg.Wait()
}
