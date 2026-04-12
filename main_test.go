package main_test

import (
  "horellana/reddit-wallpaper-download"
  "testing"
)

func TestGenerateUrl(t *testing.T) {
  expected := "https://www.reddit.com/r/foobar/top.json?t=all"
  got := main.GenerateUrl("foobar", "top", "all")

  if got != expected {
    t.Errorf("expected %s got %s", expected, got);
  }
}

func TestGetImageOutputPath(t *testing.T) {
  expected := "/tmp/1080p/foo_bar.png"
  got, _ := main.GenerateOutputPath("https://i.reddit.com/bar/foo.png", "/tmp", "bar", 1920, 1080)

  if got != expected {
    t.Errorf("expected %s got %s", expected, got);
  }
}

func TestGetResolutionCategory(t *testing.T) {
	tests := []struct {
		width    int
		height   int
		expected string
	}{
		{3840, 2160, "4k"},
		{4000, 3000, "4k"},
		{2560, 1440, "2k"},
		{3000, 2000, "2k"},
		{1920, 1080, "1080p"},
		{2000, 1200, "1080p"},
		{1280, 720, "other"},
		{0, 0, "other"},
	}

	for _, tt := range tests {
		got := main.GetResolutionCategory(tt.width, tt.height)
		if got != tt.expected {
			t.Errorf("GetResolutionCategory(%d, %d) = %s; want %s", tt.width, tt.height, got, tt.expected)
		}
	}
}

func TestCheckResolution(t *testing.T) {
	tests := []struct {
		width    int
		height   int
		minRes   string
		expected bool
		isValid  bool
	}{
		{1920, 1080, "1080p", true, true},
		{3840, 2160, "1080p", true, true},
		{1280, 720, "1080p", false, true},
		{2560, 1440, "2k", true, true},
		{1920, 1080, "2k", false, true},
		{3840, 2160, "4k", true, true},
		{2560, 1440, "4k", false, true},
		{0, 0, "1080p", false, true},
		{0, 0, "none", true, true},
		{1920, 1080, "invalid", false, false},
	}

	for _, tt := range tests {
		meets, valid := main.CheckResolution(tt.width, tt.height, tt.minRes)
		if valid != tt.isValid {
			t.Errorf("CheckResolution(%d, %d, %s) valid = %v; want %v", tt.width, tt.height, tt.minRes, valid, tt.isValid)
		}
		if meets != tt.expected {
			t.Errorf("CheckResolution(%d, %d, %s) meets = %v; want %v", tt.width, tt.height, tt.minRes, meets, tt.expected)
		}
	}
}

func TestFilterImages(t *testing.T) {
	children := []main.ImageUrl{
		{Data: struct {
			Title     string `json:"title"`
			Subreddit string `json:"subreddit"`
			Mature    bool   `json:"over_18"`
			Url       string `json:"url"`
			Preview   struct {
				Images []struct {
					Source struct {
						Width  int `json:"width"`
						Height int `json:"height"`
					} `json:"source"`
				} `json:"images"`
			} `json:"preview"`
		}{Url: "test.jpg", Preview: struct {
			Images []struct {
				Source struct {
					Width  int `json:"width"`
					Height int `json:"height"`
				} `json:"source"`
			} `json:"images"`
		}{Images: []struct {
			Source struct {
				Width  int `json:"width"`
				Height int `json:"height"`
			} `json:"source"`
		}{{Source: struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		}{Width: 1920, Height: 1080}}}}}},
		{Data: struct {
			Title     string `json:"title"`
			Subreddit string `json:"subreddit"`
			Mature    bool   `json:"over_18"`
			Url       string `json:"url"`
			Preview   struct {
				Images []struct {
					Source struct {
						Width  int `json:"width"`
						Height int `json:"height"`
					} `json:"source"`
				} `json:"images"`
			} `json:"preview"`
		}{Url: "small.png", Preview: struct {
			Images []struct {
				Source struct {
					Width  int `json:"width"`
					Height int `json:"height"`
				} `json:"source"`
			} `json:"images"`
		}{Images: []struct {
			Source struct {
				Width  int `json:"width"`
				Height int `json:"height"`
			} `json:"source"`
		}{{Source: struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		}{Width: 1280, Height: 720}}}}}},
	}

	got := main.FilterImages(children, false, "1080p")
	if len(got) != 1 {
		t.Errorf("Expected 1 image, got %d", len(got))
	}
}

func TestGetMinFileSize(t *testing.T) {
	tests := []struct {
		category string
		expected int64
	}{
		{"4k", 800000},
		{"2k", 400000},
		{"1080p", 200000},
		{"other", 100000},
		{"unknown", 100000},
	}

	for _, tt := range tests {
		got := main.GetMinFileSize(tt.category)
		if got != tt.expected {
			t.Errorf("GetMinFileSize(%s) = %d; want %d", tt.category, got, tt.expected)
		}
	}
}
