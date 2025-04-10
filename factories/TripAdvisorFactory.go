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
)

// TripAdvisor struct
type TripAdvisorFactory struct{}

// CreateProduct method for TripAdvisorFactory
func (f *TripAdvisorFactory) CreateProduct() AbstractProduct {
	apiKey := os.Getenv("TRIPADVISOR_API_KEY")
	if apiKey == "" {
		panic("TRIPADVISOR_API_KEY environment variable is not set")
	}
	return &TripAdvisorProduct{
		TripadvisorProductApiKey:  apiKey,
		TripadvisorProductBaseUrl: "https://api.content.tripadvisor.com/api/v1/location/search",
	}
}

// TripAdvisorProduct struct
type TripAdvisorProduct struct {
	TripadvisorProductApiKey  string
	TripadvisorProductBaseUrl string
}

// Ensure TripAdvisorProduct implements AbstractProduct
var _ AbstractProduct = (*TripAdvisorProduct)(nil)

// PerformAction method to use LLM for action determination
func (p *TripAdvisorProduct) PerformAction(data map[string]string) (map[string]interface{}, error) {
	// Check if the prompt is provided for LLM analysis
	prompt, exists := data["prompt"]
	if exists {
		// Analyze the prompt to determine the action and parameters
		actionDetails, err := AnalyzeTripAdvisorPromptWithLLM(prompt)

		if err != nil {
			return nil, fmt.Errorf("error analyzing prompt with LLM: %v", err)
		}

		if actionDetails != nil {
			fmt.Printf("The analyzed action is: Action: %s, Params: %v\n", actionDetails.Action, actionDetails.Parameters)
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
	actionDetails := TripadvisorAction{
		Action:     action,
		Parameters: params,
	}

	return p.performHTTPRequest(actionDetails)
}

func (p *TripAdvisorProduct) performHTTPRequest(tra TripadvisorAction) (map[string]interface{}, error) {
	// Construct the endpoint URL
	baseURL := p.TripadvisorProductBaseUrl
	endpoint := fmt.Sprintf("%s?%s", baseURL, "key="+p.TripadvisorProductApiKey)

	// Create URL from string
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %v", err)
	}

	// Prepare the query parameters
	q := u.Query()
	for k, v := range tra.Parameters {
		// Ensure the correct parameter names are used as expected by the TripAdvisor API
		if k == "query" || k == "keyword" { // Handle both 'query' and 'keyword' as 'searchQuery'
			q.Set("searchQuery", v) // Set 'searchQuery'
		} else {
			q.Set(k, v)
		}
	}

	// Encode the parameters and update the URL
	u.RawQuery = q.Encode()
	fmt.Println("The full URL is: ", u.String())

	// Make the HTTP GET request
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	// Attempt to unmarshal into a generic interface first to inspect the data type
	var result interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("error decoding JSON response: %v", err)
	}

	// Handle both possible data types of the response
	switch res := result.(type) {
	case map[string]interface{}:
		// The response is a single JSON object (map)
		return res, nil
	case []interface{}:
		// The response is a JSON array
		if len(res) > 0 {
			if firstElem, ok := res[0].(map[string]interface{}); ok {
				return firstElem, nil
			}
			return nil, errors.New("first element of the array is not a JSON object")
		}
		return nil, errors.New("JSON array is empty")
	default:
		return nil, errors.New("JSON response is neither an array nor an object")
	}
}

// TripadvisorAction contains the action and parameters required for the Tripadvisor API
type TripadvisorAction struct {
	Action     string            `json:"action"`
	Parameters map[string]string `json:"parameters"`
}

// Helper function to convert map[string]interface{} to map[string]string
func convertToStringMap(input map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for key, value := range input {
		result[key] = fmt.Sprintf("%v", value)
	}
	return result
}

// AnalyzePromptWithLLM uses an LLM to analyze the prompt and suggest Ticketmaster actions
func AnalyzeTripAdvisorPromptWithLLM(prompt string) (*TripadvisorAction, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	print("the api key is: ", apiKey)
	endpoint := "https://api.openai.com/v1/chat/completions"

	requestBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a system that derives API actions and query parameters based on user prompts. Please return only a JSON object with the action and parameters."},
			{"role": "system", "content": "Query parameters with dates must be in the valid format YYYY-MM-DDTHH:mm:ssZ (example: 2020-08-01T14:00:00Z)."},
			{"role": "user", "content": "Remove any part of the query that is related to Tickets or accommodations."},
			{"role": "user", "content": fmt.Sprintf(`Given the user's request: '%s', determine the most appropriate Tripadvisor API action and parameters. Return a JSON object with the action and parameters. 
Consider valid actions such as:
- searchQuery (string): Search for locations based on a query string.
- category (string): Filters result set based on property type. Valid options are 'hotels', 'attractions', 'restaurants', and 'geos'.
- phone (string): Phone number to filter the search results by (this can be in any format with spaces and dashes but without the '+' sign at the beginning).
- address (string): Address to filter the search results by.
- latLong (string): Latitude/Longitude pair to scope down the search around a specific point, e.g., '42.3455,-71.10767'.
- radius (number > 0): Length of the radius from the provided latitude/longitude pair to filter results.
- radiusUnit (string): Unit for length of the radius. Valid options are 'km', 'mi', 'm' (km=kilometers, mi=miles, m=meters).
- language (string, defaults to 'en'): The language in which to return results (e.g., 'en' for English or 'es' for Spanish) from the list of supported languages.
- includeSpellcheck: Include spell check suggestions in response.`, prompt)},
			{"role": "system", "content": "response_format={ \"type\": \"json_object\" }"},
		},
		"max_tokens": 500,
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

	var intermediate interface{}
	content := strings.TrimSpace(response.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	println("The content IS: ", content)

	// Attempt to unmarshal into a generic interface
	if err := json.Unmarshal([]byte(content), &intermediate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal action from content: %v", err)
	}

	// Check if the intermediate is an array or a map
	switch v := intermediate.(type) {
	case []interface{}:
		if len(v) > 0 {
			firstElement, ok := v[0].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("unexpected array element type")
			}
			intermediate = firstElement
		} else {
			return nil, fmt.Errorf("empty array in response")
		}
	case map[string]interface{}:
		// Already a map, no changes needed
	default:
		return nil, fmt.Errorf("unexpected response format: neither array nor map")
	}

	// Check if the content is an array or a map
	var action TripadvisorAction
	switch v := intermediate.(type) {
	case []interface{}:
		// Handle array case (e.g., take the first element if applicable)
		if len(v) > 0 {
			firstElement, ok := v[0].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("unexpected array element type")
			}
			parametersMap, ok := firstElement["parameters"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("parameters field is not of expected type map[string]interface{}")
			}
			action = TripadvisorAction{
				Action:     firstElement["action"].(string),
				Parameters: convertToStringMap(parametersMap),
			}
		} else {
			return nil, fmt.Errorf("empty array in response")
		}
	case map[string]interface{}:
		// Handle map case
		action = TripadvisorAction{
			Action:     v["action"].(string),
			Parameters: convertToStringMap(v["parameters"].(map[string]interface{})),
		}
	default:
		return nil, fmt.Errorf("unexpected response format")
	}

	params := make(map[string]string)
	typedIntermediate, ok := intermediate.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected type for intermediate, expected map[string]interface{}")
	}

	for key, value := range typedIntermediate["parameters"].(map[string]interface{}) {
		params[key] = toString(value)
	}

	typedIntermediate, ok = intermediate.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected type for intermediate, expected map[string]interface{}")
	}

	action = TripadvisorAction{
		Action:     typedIntermediate["action"].(string),
		Parameters: params,
	}

	return &action, nil
}
