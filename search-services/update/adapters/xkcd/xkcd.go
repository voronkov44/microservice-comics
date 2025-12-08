package xkcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"yadro.com/course/update/core"
)

type Client struct {
	log    *slog.Logger
	client http.Client
	url    string
}

func NewClient(url string, timeout time.Duration, log *slog.Logger) (*Client, error) {
	if url == "" {
		return nil, fmt.Errorf("empty base url specified")
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	return &Client{
		client: http.Client{Timeout: timeout},
		log:    log,
		url:    strings.TrimRight(url, "/"),
	}, nil
}

type res struct {
	Num        int    `json:"num"`
	Img        string `json:"img"`
	Title      string `json:"title"`
	Alt        string `json:"alt"`
	Transcript string `json:"transcript"`
}

func (c Client) Get(ctx context.Context, id int) (core.XKCDInfo, error) {
	u := fmt.Sprintf("%s/%d/info.0.json", c.url, id)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)

	r, err := c.client.Do(req)
	if err != nil {
		return core.XKCDInfo{}, err
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			c.log.Debug("failed to close response body in Get", "id", id, "error", err)
		}
	}()

	switch r.StatusCode {
	case http.StatusOK:
		var x res
		if err := json.NewDecoder(r.Body).Decode(&x); err != nil {
			return core.XKCDInfo{}, err
		}
		desc := strings.TrimSpace(x.Transcript)

		return core.XKCDInfo{
			ID:          x.Num,
			URL:         x.Img,
			Title:       x.Title,
			Alt:         x.Alt,
			Description: desc,
		}, nil
	case http.StatusNotFound:
		return core.XKCDInfo{}, core.ErrNotFound
	default:
		return core.XKCDInfo{}, fmt.Errorf("xkcd %d: http %d", id, r.StatusCode)
	}
}

func (c Client) LastID(ctx context.Context) (int, error) {
	u := fmt.Sprintf("%s/info.0.json", c.url)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)

	r, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			c.log.Debug("failed to close response body in LastID", "error", err)
		}
	}()

	if r.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("xkcd latest: http %d", r.StatusCode)
	}

	var x res
	if err := json.NewDecoder(r.Body).Decode(&x); err != nil {
		return 0, err
	}
	if x.Num <= 0 {
		return 0, fmt.Errorf("xkcd latest: invalid num %d", x.Num)
	}
	return x.Num, nil
}
