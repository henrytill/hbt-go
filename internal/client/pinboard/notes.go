package pinboard

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/henrytill/hbt-go/internal/pinboard"
)

type Note = pinboard.Note

func (c *Client) ListNotes(ctx context.Context) ([]Note, error) {
	resp, err := c.makeRequest(ctx, "notes/list", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Notes []Note `json:"notes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode notes list response: %w", err)
	}

	return result.Notes, nil
}

func (c *Client) GetNote(ctx context.Context, noteID string) (*Note, error) {
	endpoint := fmt.Sprintf("notes/%s", noteID)

	resp, err := c.makeRequest(ctx, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var note Note
	if err := json.NewDecoder(resp.Body).Decode(&note); err != nil {
		return nil, fmt.Errorf("failed to decode note response: %w", err)
	}

	return &note, nil
}
