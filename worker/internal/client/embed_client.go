package client

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type EmbedClient struct {
	baseURL string
	http    *http.Client
}

func NewEmbedClient(baseURL string) *EmbedClient {
	return &EmbedClient{baseURL: baseURL, http: &http.Client{}}
}

type EmbedRequest struct {
	Kind   string `json:"kind"`   // "image" | "text"
	Prompt string `json:"prompt"` // "catalog_image" | "catalog_text" — NEVER "query_*" from this service
	// exactly one of the below is set
	ImageBase64 string `json:"image_base64,omitempty"`
	Text        string `json:"text,omitempty"`
}

type EmbedResponse struct {
	Vector []float32 `json:"vector"`
}

// Embed calls the embedding server. Callers MUST pass a "catalog_*"
// prompt — the worker only ever embeds catalog content, never queries —
// to preserve query/document asymmetry (see embedding-server).
func (c *EmbedClient) Embed(req EmbedRequest) (*EmbedResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Post(c.baseURL+"/embed", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
