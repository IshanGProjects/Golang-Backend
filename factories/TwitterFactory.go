package factories

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// TwitterFactory struct
type TwitterFactory struct{}

// CreateProduct creates a new TwitterProduct
func (f *TwitterFactory) CreateProduct() AbstractProduct {
	return &TwitterProduct{
		TwitterProductBaseUrl: "http://localhost:8080/search",
	}
}

// TwitterProduct struct
type TwitterProduct struct {
	TwitterProductBaseUrl string
}

// Ensure TwitterProduct implements AbstractProduct
var _ AbstractProduct = (*TwitterProduct)(nil)

func (p *TwitterProduct) PerformAction(data map[string]string) (map[string]interface{}, error) {
	prompt, ok := data["prompt"]
	if !ok || prompt == "" {
		return nil, fmt.Errorf("missing 'prompt' key")
	}

	// Ask LLM to extract just the query
	queryParams, err := AnalyzeTwitterPromptWithLLM(prompt)
	if err != nil {
		return nil, err
	}

	query, ok := queryParams["query"]
	if !ok || query == "" {
		return nil, fmt.Errorf("LLM did not return a valid 'query'")
	}

	return p.performHTTPRequest(query)
}

// performHTTPRequest performs the GET request with query and date range
func (p *TwitterProduct) performHTTPRequest(query string) (map[string]interface{}, error) {
	now := time.Now().UTC()
	since := now.AddDate(0, -1, 0).Format("2006-01-02")
	until := now.Format("2006-01-02")

	queryParams := url.Values{}
	queryParams.Set("f", "tweets")
	queryParams.Set("since", since)
	queryParams.Set("until", until)
	queryParams.Set("q", query)

	fullURL := fmt.Sprintf("%s?%s", p.TwitterProductBaseUrl, queryParams.Encode())
	fmt.Println("The full Twitter search URL is:", fullURL)

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var result interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("error decoding JSON response: %v", err)
	}

	switch res := result.(type) {
	case map[string]interface{}:
		return res, nil
	case []interface{}:
		if len(res) > 0 {
			if first, ok := res[0].(map[string]interface{}); ok {
				return first, nil
			}
			return nil, errors.New("first element is not a JSON object")
		}
		return nil, errors.New("empty result array")
	default:
		return nil, errors.New("unexpected response format")
	}
}

func AnalyzeTwitterPromptWithLLM(prompt string) (map[string]string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	endpoint := "https://api.openai.com/v1/chat/completions"

	requestBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "system", "content": "You extract query strings for social media searches from user prompts."},
			{"role": "system", "content": "Return a JSON object with only one key: 'query'. No actions."},
			{"role": "user", "content": fmt.Sprintf("Given this prompt: '%s', what should the Twitter search query be? Respond with: {\"query\": \"...\"}", prompt)},
		},
		"max_tokens": 100,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response data: %v", err)
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(responseData, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal LLM response: %v", err)
	}
	if len(response.Choices) == 0 {
		return nil, errors.New("no LLM response")
	}

	// Strip markdown
	content := strings.TrimSpace(response.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var result map[string]string
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to decode LLM JSON: %v\nContent: %s", err, content)
	}
	return result, nil
}
