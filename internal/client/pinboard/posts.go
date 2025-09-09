package pinboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/henrytill/hbt-go/internal/pinboard"
)

type GetAllPostsOptions struct {
	Tag     []string
	Start   int
	Results int
	FromDt  time.Time
	ToDt    time.Time
	Meta    bool
}

func (c *Client) GetAllPosts(ctx context.Context, opts *GetAllPostsOptions) ([]pinboard.Post, error) {
	params := url.Values{}

	if opts != nil {
		if len(opts.Tag) > 0 {
			for _, tag := range opts.Tag {
				if tag != "" {
					params.Add("tag", tag)
				}
			}
		}
		if opts.Start > 0 {
			params.Set("start", strconv.Itoa(opts.Start))
		}
		if opts.Results > 0 {
			params.Set("results", strconv.Itoa(opts.Results))
		}
		if !opts.FromDt.IsZero() {
			params.Set("fromdt", opts.FromDt.Format(time.RFC3339))
		}
		if !opts.ToDt.IsZero() {
			params.Set("todt", opts.ToDt.Format(time.RFC3339))
		}
		if opts.Meta {
			params.Set("meta", "yes")
		}
	}

	resp, err := c.makeRequest(ctx, "posts/all", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var posts []pinboard.Post
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, fmt.Errorf("failed to decode posts response: %w", err)
	}

	return posts, nil
}

func (c *Client) GetRecentPosts(ctx context.Context, count int, tags []string, meta bool) ([]pinboard.Post, error) {
	params := url.Values{}
	if count > 0 {
		params.Set("count", strconv.Itoa(count))
	}
	for _, tag := range tags {
		if tag != "" {
			params.Add("tag", tag)
		}
	}
	if meta {
		params.Set("meta", "yes")
	}

	resp, err := c.makeRequest(ctx, "posts/recent", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Posts []pinboard.Post `json:"posts"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode recent posts response: %w", err)
	}

	return result.Posts, nil
}

func (c *Client) GetPosts(ctx context.Context, tags []string, dt, urlParam string, meta bool) ([]pinboard.Post, error) {
	params := url.Values{}
	for _, tag := range tags {
		if tag != "" {
			params.Add("tag", tag)
		}
	}
	if dt != "" {
		params.Set("dt", dt)
	}
	if urlParam != "" {
		params.Set("url", urlParam)
	}
	if meta {
		params.Set("meta", "yes")
	}

	resp, err := c.makeRequest(ctx, "posts/get", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Posts []pinboard.Post `json:"posts"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode posts response: %w", err)
	}

	return result.Posts, nil
}

type AddPostOptions struct {
	Extended string
	Tags     string
	Dt       time.Time
	Replace  *bool
	Shared   *bool
	ToRead   *bool
}

func (c *Client) AddPost(ctx context.Context, urlParam, description string, opts *AddPostOptions) error {
	params := url.Values{}
	params.Set("url", urlParam)
	params.Set("description", description)

	if opts != nil {
		if opts.Extended != "" {
			params.Set("extended", opts.Extended)
		}
		if opts.Tags != "" {
			params.Set("tags", opts.Tags)
		}
		if !opts.Dt.IsZero() {
			params.Set("dt", opts.Dt.Format(time.RFC3339))
		}
		if opts.Replace != nil {
			if *opts.Replace {
				params.Set("replace", "yes")
			} else {
				params.Set("replace", "no")
			}
		}
		if opts.Shared != nil {
			if *opts.Shared {
				params.Set("shared", "yes")
			} else {
				params.Set("shared", "no")
			}
		}
		if opts.ToRead != nil {
			if *opts.ToRead {
				params.Set("toread", "yes")
			}
		}
	}

	resp, err := c.makeRequest(ctx, "posts/add", params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		ResultCode string `json:"result_code"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode add post response: %w", err)
	}

	if result.ResultCode != "done" {
		return fmt.Errorf("failed to add post: %s", result.ResultCode)
	}

	return nil
}

func (c *Client) DeletePost(ctx context.Context, urlParam string) error {
	params := url.Values{}
	params.Set("url", urlParam)

	resp, err := c.makeRequest(ctx, "posts/delete", params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		ResultCode string `json:"result_code"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode delete post response: %w", err)
	}

	if result.ResultCode != "done" {
		return fmt.Errorf("failed to delete post: %s", result.ResultCode)
	}

	return nil
}

func (c *Client) GetPostsDates(ctx context.Context, tags []string) (map[string]int, error) {
	params := url.Values{}
	for _, tag := range tags {
		if tag != "" {
			params.Add("tag", tag)
		}
	}

	resp, err := c.makeRequest(ctx, "posts/dates", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Dates map[string]int `json:"dates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode posts dates response: %w", err)
	}

	return result.Dates, nil
}

func (c *Client) GetUpdate(ctx context.Context) (time.Time, error) {
	resp, err := c.makeRequest(ctx, "posts/update", nil)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	var result struct {
		UpdateTime string `json:"update_time"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return time.Time{}, fmt.Errorf("failed to decode update response: %w", err)
	}

	updateTime, err := time.Parse(time.RFC3339, result.UpdateTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse update time: %w", err)
	}

	return updateTime, nil
}

func (c *Client) SuggestTags(ctx context.Context, urlParam string) ([]string, []string, error) {
	params := url.Values{}
	params.Set("url", urlParam)

	resp, err := c.makeRequest(ctx, "posts/suggest", params)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	var result []struct {
		Popular     []string `json:"popular"`
		Recommended []string `json:"recommended"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("failed to decode suggest response: %w", err)
	}

	if len(result) == 0 {
		return []string{}, []string{}, nil
	}

	return result[0].Popular, result[0].Recommended, nil
}
