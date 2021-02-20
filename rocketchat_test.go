package rocketapi

import (
  "os"
  "net/http"
  "testing"
)

var (
  chat Chat
)

func TestMain(m *testing.M) {
  setup()
  code := m.Run()
  shutdown()
  os.Exit(code)
}

func TestGetRoom(t *testing.T) {
  rooms, err := chat.GetRoom()
  if err != nil {
    t.Fatal(err)
  }
  if len(rooms.Update) == 0 {
    t.Fatal("GetRoom failed")
  }
}

func TestGetIMList(t *testing.T) {
  imListResp, err := chat.GetIMList()
  if err != nil {
    t.Fatal(err)
  }
  if len(imListResp.IMs) == 0 {
    t.Fatal("GetIMList failed")
  }
}

func TestSetStatus(t *testing.T) {
  err := chat.SetStatus("online", "WAZAAAAAAAA")
  if err != nil {
    t.Fatal(err)
  }
}

func TestPostMessage(t *testing.T) {
  data := map[string]string{
    "channel": "waza",
    "text": "a simple message",
  }
  err := chat.PostMessage(data)
  if err != nil {
    t.Fatal(err)
  }
}

func setup() {
  username := os.Getenv("ROCKET_USER")
  password := os.Getenv("ROCKET_PASSWORD")
  baseURL := os.Getenv("ROCKET_URL")
  chat = NewChat(&http.Client{}, baseURL)
  if err := chat.Login(username, password); err != nil {
    panic(err)
  }
}

func shutdown() {
}
