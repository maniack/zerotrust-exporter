package magicwan

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type variables struct {
	AccountTag    string `json:"accountTag"`
	DatetimeStart string `json:"datetimeStart"`
	DatetimeEnd   string `json:"datetimeEnd"`
}

func doGraphQLRequest(ctx context.Context, token string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.cloudflare.com/client/v4/graphql",
		bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("authorization failed: status %d", resp.StatusCode)
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respBytes, nil
}
