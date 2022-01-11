package rocketapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type (
	userInfoKey string
	// UserInfo type
	UserInfo map[string]string
)

var (
	// UserInfoKey user info key variable.
	UserInfoKey = userInfoKey("user_info")
)

// Logger logger interface
type Logger interface {
	Debugf(string, ...interface{})
	Info(...interface{})
	Warn(...interface{})
}

type defaultLogger struct{}

func (d defaultLogger) Debugf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func (d defaultLogger) Info(v ...interface{}) {
	fmt.Println(v...)
}

func (d defaultLogger) Warn(v ...interface{}) {
	fmt.Println(v...)
}

// Chat chat structure.
type Chat struct {
	Client  *http.Client
	BaseURL string
	Logger  Logger
	Token   string
	UserID  string
}

// NewChat init new chat.
func NewChat(client *http.Client, baseURL string) Chat {
	return Chat{
		Client:  client,
		BaseURL: baseURL,
		Logger:  &defaultLogger{},
	}
}

// SetLogger set logger
func (chat Chat) SetLogger(logger Logger) {
	chat.Logger = logger
}

// GetUserInfo Get user info key
func GetUserInfo(ctx context.Context) UserInfo {
	return ctx.Value(UserInfoKey).(UserInfo)
}

type errorResponse struct {
	Success   bool   `json:"success"`
	Error     string `json:"error"`
	ErrorType string `json:"errorType"`
}

// PostMessageResponse post message response
type PostMessageResponse struct {
	Success bool `json:"success"`
	Timestamp int64 `json:"ts"`
	Channel string `json:"channel"`
	Message struct {
		Alias string
		Message string `json:"msg"`
		ParseURLs bool `json:"parseUrls"`
		Groupable bool `json:"groupable"`
		Timestamp string `json:"ts"`
		User struct {
			ID string `json:"_id"`
			Username string `json:"username"`
		} `json:"u"`
		RoomID string `json:"rid"`
		UpdatedAt string `json:"_updatedAt"`
		ID string `json:"_id"`
	}
}

type loginResponse struct {
	Status string `json:"status"`
	Data   struct {
		UserID    string `json:"userId"`
		AuthToken string `json:"authToken"`
		Me        struct {
			ID string `json:"_id"`
		} `json:"me"`
	} `json:"data"`
}

type getChannelsResponse struct {
	Channels []struct {
		ID   string `json:"_id"`
		Name string `json:"name"`
		Msgs int    `json:"msgs"`
	} `json:"channels"`
	Status bool `json:"success"`
	Total  int  `json:"total"`
	Count  int  `json:"count"`
	Offset int  `json:"offset"`
}

// RoomsGetResponse get rooms response
type RoomsGetResponse struct {
	Update []struct {
		ID      string `json:"_id"`
		Name    string `json:"name"`
		Default bool   `json:"default"`
	} `json:"update"`
	Status string `json:"status"`
}

// ChannelsHistoryResponse get channels history response
type ChannelsHistoryResponse struct {
	Messages []Message `json:"messages"`
	Success  bool      `json:"success"`
}

// Message message structure
type Message struct {
	ID        string `json:"_id"`
	Msg       string `json:"msg"`
	Ts        string `json:"ts"`
	UpdatedAt string `json:"_updatedAt"`
	U         struct {
		ID       string `json:"_id"`
		Username string `json:"username"`
	} `json:"u"`
	RID string `json:"rid"`
}

type discussionGetResponse struct{}

type loginErrResponse struct {
	Status  string `json:"status"`
	Error   int    `json:"error"`
	Message string `json:"message"`
}

// Login login on chat
func (chat *Chat) Login(username, password string) error {
	body := map[string]string{
		"username": username,
		"password": password,
	}
	jsonValue, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/v1/login", chat.BaseURL),
		bytes.NewBuffer(jsonValue),
	)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := chat.Client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	loginResp := loginResponse{}

	if res.StatusCode != 200 {
		loginErrResp := loginErrResponse{}
		err = json.NewDecoder(res.Body).Decode(&loginErrResp)
		if err != nil {
			return err
		}
		return fmt.Errorf("Error status: %d, Message: %s", loginErrResp.Error, loginErrResp.Message)
	}
	err = json.NewDecoder(res.Body).Decode(&loginResp)
	if err != nil {
		return err
	}
	chat.UserID = loginResp.Data.UserID
	chat.Token = loginResp.Data.AuthToken
	return nil
}

type logoutResponse struct {
	Status string `json:"status"`
	Data   struct {
		Message string `json:"message"`
	} `json:"data"`
}

// Logout Logout of chat
func (chat *Chat) Logout() error {
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/v1/logout", chat.BaseURL),
		nil,
	)
	if err != nil {
		return err
	}
	res, err := chat.Client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	logoutResp := logoutResponse{}
	err = json.NewDecoder(res.Body).Decode(&logoutResp)
	if err != nil {
		return err
	}
	if logoutResp.Status == "error" {
		return fmt.Errorf("fail to logout")
	}
	return nil
}

func (chat Chat) getDiscussion(roomID string) (discussionGetResponse, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/v1/chat.getDiscussions?roomId=%s", chat.BaseURL, roomID),
		nil,
	)
	if err != nil {
		return discussionGetResponse{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Auth-Token", chat.Token)
	req.Header.Add("X-User-id", chat.UserID)
	res, err := chat.Client.Do(req)
	if err != nil {
		return discussionGetResponse{}, err
	}

	defer res.Body.Close()
	discussionGetResp := discussionGetResponse{}

	err = json.NewDecoder(res.Body).Decode(&discussionGetResp)
	if err != nil {
		return discussionGetResponse{}, err
	}

	return discussionGetResp, nil
}

func index(arr []string, data string) int {
	for idx, item := range arr {
		if item == data {
			return idx
		}
	}
	return -1
}

type currentRoom struct {
	channels []string
	ims      []string
}

func (chat Chat) getCurrentRoom() (currentRoom, error) {
	roomsResp, err := chat.GetRoom()
	if err != nil {
		return currentRoom{}, err
	}
	channels := make([]string, 0)
	for _, item := range roomsResp.Update {
		channels = append(channels, item.ID)
	}
	ims, err := chat.GetIMList()
	if err != nil {
		return currentRoom{}, err
	}
	imms := make([]string, 0)
	for _, im := range ims.IMs {
		imms = append(imms, im.ID)
	}

	return currentRoom{
		channels,
		imms,
	}, nil
}

// GetIncomingMessage Fetch incoming message
func (chat Chat) GetIncomingMessage(
	sleepTime time.Duration,
) <-chan Message {
	msgChan := make(chan Message)
	go func() {
		now := time.Now()
		lastMessageID := []string{}
		maxSize := 50
		for {
			current, err := chat.getCurrentRoom()
			if err != nil {
				chat.Logger.Warn(err)
			}
			chat.Logger.Debugf("Current room: %v", current)
			for _, im := range current.ims {
				resp, err := chat.getIMHistory(
					im,
					"",
					now.Format(time.RFC3339),
					true,
				)
				if err == nil {
					for _, msg := range resp.Messages {
						chat.Logger.Debugf("msg: %s", msg)
						if msg.U.ID != chat.UserID && index(lastMessageID, msg.ID) == -1 {
							msgChan <- msg
							lastMessageID = append(lastMessageID, msg.ID)
							if len(lastMessageID) > maxSize {
								lastMessageID = lastMessageID[maxSize/2:]
							}
						}
					}
				}
			}
			for _, channel := range current.channels {
				resp, err := chat.getChannelsHistory(
					channel,
					"",
					now.Format(time.RFC3339),
					true,
				)
				if err == nil {
					for _, msg := range resp.Messages {
						chat.Logger.Debugf("msg: %s", msg)
						if msg.U.ID != chat.UserID && index(lastMessageID, msg.ID) == -1 {
							msgChan <- msg
							lastMessageID = append(lastMessageID, msg.ID)
							if len(lastMessageID) > maxSize {
								lastMessageID = lastMessageID[maxSize/2:]
							}
						}
					}
				}
			}
			now = time.Now()
			time.Sleep(sleepTime)
		}
	}()

	return msgChan
}

func (chat Chat) getChannelsHistory(
	roomID string,
	latest string,
	oldest string,
	unreads bool,
) (ChannelsHistoryResponse, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/v1/channels.history", chat.BaseURL),
		nil,
	)
	if err != nil {
		return ChannelsHistoryResponse{}, err
	}
	query := req.URL.Query()
	query.Add("roomId", roomID)
	if latest != "" {
		query.Add("latest", latest)
	}
	if oldest != "" {
		query.Add("oldest", oldest)
	}
	if unreads {
		query.Add("unreads", "true")
	} else {
		query.Add("unreads", "false")
	}
	req.URL.RawQuery = query.Encode()
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Auth-Token", chat.Token)
	req.Header.Add("X-User-id", chat.UserID)
	res, err := chat.Client.Do(req)
	if err != nil {
		return ChannelsHistoryResponse{}, err
	}

	defer res.Body.Close()
	if res.StatusCode >= 400 && res.StatusCode < 500 {
		errorResp := errorResponse{}
		err = json.NewDecoder(res.Body).Decode(&errorResp)
		if err != nil {
			return ChannelsHistoryResponse{}, err
		}
		return ChannelsHistoryResponse{}, fmt.Errorf(
			"status code: %d %s: %s",
			res.StatusCode,
			errorResp.ErrorType,
			errorResp.Error,
		)
	}

	channelsHistoryResp := ChannelsHistoryResponse{}

	err = json.NewDecoder(res.Body).Decode(&channelsHistoryResp)
	if err != nil {
		return ChannelsHistoryResponse{}, err
	}
	if !channelsHistoryResp.Success {
		return ChannelsHistoryResponse{}, fmt.Errorf("fail to getChannelsHistory")
	}

	return channelsHistoryResp, nil
}

func (chat Chat) getIMHistory(
	roomID string,
	latest string,
	oldest string,
	unreads bool,
) (ChannelsHistoryResponse, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/v1/im.history", chat.BaseURL),
		nil,
	)
	if err != nil {
		return ChannelsHistoryResponse{}, err
	}
	query := req.URL.Query()
	query.Add("roomId", roomID)
	if latest != "" {
		query.Add("latest", latest)
	}
	if oldest != "" {
		query.Add("oldest", oldest)
	}
	if unreads {
		query.Add("unreads", "true")
	} else {
		query.Add("unreads", "false")
	}
	req.URL.RawQuery = query.Encode()
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Auth-Token", chat.Token)
	req.Header.Add("X-User-id", chat.UserID)
	res, err := chat.Client.Do(req)
	if err != nil {
		return ChannelsHistoryResponse{}, err
	}

	defer res.Body.Close()
	if res.StatusCode >= 400 && res.StatusCode < 500 {
		errorResp := errorResponse{}
		err = json.NewDecoder(res.Body).Decode(&errorResp)
		if err != nil {
			return ChannelsHistoryResponse{}, err
		}
		return ChannelsHistoryResponse{}, fmt.Errorf(
			"status code: %d %s: %s",
			res.StatusCode,
			errorResp.ErrorType,
			errorResp.Error,
		)
	}

	channelsHistoryResp := ChannelsHistoryResponse{}

	err = json.NewDecoder(res.Body).Decode(&channelsHistoryResp)
	if err != nil {
		return ChannelsHistoryResponse{}, err
	}
	if !channelsHistoryResp.Success {
		return ChannelsHistoryResponse{}, fmt.Errorf("fail to getChannelsHistory")
	}

	return channelsHistoryResp, nil
}

func (chat Chat) getChannels() (getChannelsResponse, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/v1/channels.list", chat.BaseURL),
		nil,
	)
	if err != nil {
		return getChannelsResponse{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Auth-Token", chat.Token)
	req.Header.Add("X-User-id", chat.UserID)
	res, err := chat.Client.Do(req)
	if err != nil {
		return getChannelsResponse{}, err
	}

	defer res.Body.Close()
	getChannelsResp := getChannelsResponse{}

	err = json.NewDecoder(res.Body).Decode(&getChannelsResp)
	if err != nil {
		return getChannelsResponse{}, err
	}

	return getChannelsResp, nil
}

// GetRoom get rooms
func (chat Chat) GetRoom() (RoomsGetResponse, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/v1/rooms.get", chat.BaseURL),
		nil,
	)
	if err != nil {
		return RoomsGetResponse{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Auth-Token", chat.Token)
	req.Header.Add("X-User-id", chat.UserID)
	res, err := chat.Client.Do(req)
	if err != nil {
		return RoomsGetResponse{}, err
	}

	defer res.Body.Close()
	roomsGetResp := RoomsGetResponse{}

	err = json.NewDecoder(res.Body).Decode(&roomsGetResp)
	if err != nil {
		return RoomsGetResponse{}, err
	}

	return roomsGetResp, nil
}

// SetStatus Set user status
func (chat Chat) SetStatus(message, status string) error {
	body := map[string]string{
		"message": message,
		"status":  status,
	}
	jsonValue, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST",
		fmt.Sprintf(
			"%s/api/v1/users.setStatus",
			chat.BaseURL,
		),
		bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Auth-Token", chat.Token)
	req.Header.Add("X-User-id", chat.UserID)
	resp, err := chat.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// PostMessage post message on chat
func (chat Chat) PostMessage(body map[string]string) (PostMessageResponse, error) {
	jsonValue, err := json.Marshal(body)
	if err != nil {
		return PostMessageResponse{}, err
	}
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/v1/chat.postMessage", chat.BaseURL),
		bytes.NewBuffer(jsonValue),
	)
	if err != nil {
		return PostMessageResponse{}, err
	}
	req.Header.Add("X-Auth-Token", chat.Token)
	req.Header.Add("X-User-id", chat.UserID)
	req.Header.Add("Content-Type", "application/json")
	resp, err := chat.Client.Do(req)
	if err != nil {
		return PostMessageResponse{}, err
	}
	defer resp.Body.Close()

	postMessageResp := PostMessageResponse{}
	err = json.NewDecoder(resp.Body).Decode(&postMessageResp)
	if err != nil {
		return PostMessageResponse{}, err
	}

	return postMessageResp, nil
}

// GetIMMessages get im messages
func (chat Chat) GetIMMessages(
	username string,
) (ChannelsHistoryResponse, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/v1/im.messages", chat.BaseURL),
		nil,
	)
	if err != nil {
		return ChannelsHistoryResponse{}, err
	}
	query := req.URL.Query()
	query.Add("username", username)
	req.URL.RawQuery = query.Encode()
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Auth-Token", chat.Token)
	req.Header.Add("X-User-id", chat.UserID)
	res, err := chat.Client.Do(req)
	if err != nil {
		return ChannelsHistoryResponse{}, err
	}

	defer res.Body.Close()
	if res.StatusCode >= 400 && res.StatusCode < 500 {
		errorResp := errorResponse{}
		err = json.NewDecoder(res.Body).Decode(&errorResp)
		if err != nil {
			return ChannelsHistoryResponse{}, err
		}
		return ChannelsHistoryResponse{}, fmt.Errorf(
			"status code: %d %s: %s",
			res.StatusCode,
			errorResp.ErrorType,
			errorResp.Error,
		)
	}

	channelsHistoryResp := ChannelsHistoryResponse{}

	err = json.NewDecoder(res.Body).Decode(&channelsHistoryResp)
	if err != nil {
		return ChannelsHistoryResponse{}, err
	}
	if !channelsHistoryResp.Success {
		return ChannelsHistoryResponse{}, fmt.Errorf("fail to getChannelsHistory")
	}

	return channelsHistoryResp, nil
}

// ImList im list structure
type ImList struct {
	IMs []struct {
		ID   string `json:"_id"`
		Msgs int    `json:"msgs"`
	} `json:"ims"`
	Success bool `json:"success"`
}

// GetIMList get im list
func (chat Chat) GetIMList() (ImList, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/v1/im.list", chat.BaseURL),
		nil,
	)
	if err != nil {
		return ImList{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Auth-Token", chat.Token)
	req.Header.Add("X-User-id", chat.UserID)
	res, err := chat.Client.Do(req)
	if err != nil {
		return ImList{}, err
	}

	defer res.Body.Close()
	if res.StatusCode >= 400 && res.StatusCode < 500 {
		errorResp := errorResponse{}
		err = json.NewDecoder(res.Body).Decode(&errorResp)
		if err != nil {
			return ImList{}, err
		}
		return ImList{}, fmt.Errorf(
			"status code: %d %s: %s",
			res.StatusCode,
			errorResp.ErrorType,
			errorResp.Error,
		)
	}

	channelsHistoryResp := ImList{}

	err = json.NewDecoder(res.Body).Decode(&channelsHistoryResp)
	if err != nil {
		return ImList{}, err
	}
	if !channelsHistoryResp.Success {
		return ImList{}, fmt.Errorf("fail to getChannelsHistory")
	}

	return channelsHistoryResp, nil
}
