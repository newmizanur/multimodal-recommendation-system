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

type embedReq struct {
	Kind        string `json:"kind"`
	Prompt      string `json:"prompt"`
	ImageBase64 string `json:"image_base64,omitempty"`
	Text        string `json:"text,omitempty"`
}
type embedResp struct {
	Vector []float32 `json:"vector"`
}

// EmbedText/EmbedImage always pass "query_*" prompts, not "catalog_*" —
// see query/document asymmetry in embedding-server.
func (c *EmbedClient) EmbedText(text string) ([]float32, error) {
	return c.call(embedReq{Kind: "text", Prompt: "query_text", Text: text})
}

func (c *EmbedClient) EmbedImage(imageBase64 string) ([]float32, error) {
	return c.call(embedReq{Kind: "image", Prompt: "query_image", ImageBase64: imageBase64})
}

func (c *EmbedClient) call(req embedReq) ([]float32, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Post(c.baseURL+"/embed", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out embedResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Vector, nil
}
