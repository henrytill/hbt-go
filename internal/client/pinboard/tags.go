package pinboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

func (c *Client) GetTags(ctx context.Context) (map[string]int, error) {
	resp, err := c.makeRequest(ctx, "tags/get", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tags map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("failed to decode tags response: %w", err)
	}

	return tags, nil
}

func (c *Client) DeleteTag(ctx context.Context, tag string) error {
	params := url.Values{}
	params.Set("tag", tag)

	resp, err := c.makeRequest(ctx, "tags/delete", params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Result string `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode delete tag response: %w", err)
	}

	if result.Result != "done" {
		return fmt.Errorf("failed to delete tag: %s", result.Result)
	}

	return nil
}

func (c *Client) RenameTag(ctx context.Context, old, new string) error {
	params := url.Values{}
	params.Set("old", old)
	params.Set("new", new)

	resp, err := c.makeRequest(ctx, "tags/rename", params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Result string `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode rename tag response: %w", err)
	}

	if result.Result != "done" {
		return fmt.Errorf("failed to rename tag: %s", result.Result)
	}

	return nil
}
