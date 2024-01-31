package main_test

import (
  "horellana/reddit-wallpaper-download"
  "testing"
)

func TestGenerateUrl(t *testing.T) {
  expected := "https://www.reddit.com/r/foobar.json"
  got := main.GenerateUrl("foobar")

  if got != expected {
    t.Errorf("expected %s got %s", expected, got);
  }
}

func TestGetImageOutputPath(t *testing.T) {
  expected := "/tmp/foo.png"
  got, _ := main.GenerateOutputPath("https://i.reddit.com/bar/foo.png", "/tmp")

  if got != expected {
    t.Errorf("expected %s got %s", expected, got);
  }
}
