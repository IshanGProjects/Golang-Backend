package factories

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// TicketmasterFactory struct
type TicketmasterFactory struct{}

// CreateProduct method for TicketmasterFactory
func (f *TicketmasterFactory) CreateProduct() AbstractProduct {
	return &TicketmasterProduct{
		TicketmasterApiKey:  os.Getenv("TICKETMASTER_API_KEY"),
		TicketmasterBaseUrl: "https://app.ticketmaster.com/discovery/v2",
	}
}

// TicketmasterProduct struct
type TicketmasterProduct struct {
	TicketmasterApiKey  string
	TicketmasterBaseUrl string
}

// PerformAction method revised to use LLM for action determination
func (p *TicketmasterProduct) PerformAction(data map[string]string) (map[string]interface{}, error) {
	// Check if the prompt is provided for LLM analysis
	prompt, exists := data["prompt"]
	if exists {
		// Analyze the prompt to determine the action and parameters
		actionDetails, err := AnalyzePromptWithLLM(prompt)

		if err != nil {
			return nil, fmt.Errorf("error analyzing prompt with LLM: %v", err)
		}

		if actionDetails != nil {
			fmt.Printf("The analyzed action is: Action: %s, Params: %v\n", actionDetails.Action, actionDetails.Params)
		}

		// Proceed with the determined action and parameters
		return p.performHTTPRequest(*actionDetails)
	}

	// Fallback to directly using provided action if no prompt analysis is needed
	action, ok := data["action"]
	if !ok {
		return nil, fmt.Errorf("action key is required in data or prompt for analysis")
	}

	// Extract additional parameters and perform the HTTP request using TicketmasterAction struct
	params := make(map[string]string)
	for k, v := range data {
		if k != "action" && k != "prompt" { // Exclude action and prompt keys
			params[k] = v
		}
	}
	actionDetails := TicketmasterAction{
		Action: action,
		Params: params,
	}

	return p.performHTTPRequest(actionDetails)
}

func (p *TicketmasterProduct) performHTTPRequest(tma TicketmasterAction) (map[string]interface{}, error) {
	// Make sure the base URL is correct and ends without a slash
	baseURL := "https://app.ticketmaster.com/discovery/v2" // Ensure this does not end with a slash

	// Construct the endpoint URL by appending the action and ".json" properly
	endpoint := fmt.Sprintf("%s/%s.json", baseURL, tma.Action)

	// Parse the URL to check for errors
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %v", err)
	}

	// Add the API key and other parameters to the query
	q := u.Query()
	q.Set("apikey", p.TicketmasterApiKey)
	for k, v := range tma.Params {
		q.Add(k, v)
	}

	u.RawQuery = q.Encode()

	// Check URL before making the request
	fmt.Println("Final URL being requested:", u.String())

	// Make the HTTP GET request
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Decode the JSON response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding JSON response: %v", err)
	}

	return result, nil
}

// TicketmasterAction contains the action and parameters required for the Ticketmaster API
type TicketmasterAction struct {
	Action string            `json:"action"`
	Params map[string]string `json:"params"`
}

// AnalyzePromptWithLLM uses an LLM to analyze the prompt and suggest Ticketmaster actions
func AnalyzePromptWithLLM(prompt string) (*TicketmasterAction, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	print("the api key is: ", apiKey)
	endpoint := "https://api.openai.com/v1/chat/completions"

	requestBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a system that dervies API actions based on user prompts."},
			{"role": "user", "content": fmt.Sprintf("Given the user's request: '%s', determine the most appropriate Ticketmaster API action and parameters. Return a JSON object with the action and parameters. Consider valid actions such as: attractions, attractionDetails, classifications, events, venues.", prompt)},
		},
		"max_tokens": 150,
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

	fmt.Printf("The response data is: %s\n", responseData)

	type Response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	var response Response
	if err := json.Unmarshal(responseData, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if len(response.Choices) == 0 || response.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("no response or empty content from LLM")
	}

	var action TicketmasterAction
	if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &action); err != nil {
		return nil, fmt.Errorf("failed to unmarshal action: %v", err)
	}

	fmt.Printf("The analyzed action is: Action: %s, Params: %v\n", action.Action, action.Params)
	return &action, nil
}
