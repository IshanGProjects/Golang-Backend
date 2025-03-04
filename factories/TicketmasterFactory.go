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

// PerformAction method to use LLM for action determination
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
	actionDetails := TicketmasterAction{
		Action:     action,
		Parameters: params,
	}

	return p.performHTTPRequest(actionDetails)
}

func (p *TicketmasterProduct) performHTTPRequest(tma TicketmasterAction) (map[string]interface{}, error) {
	// Make sure the base URL is correct and ends without a slash
	baseURL := "https://app.ticketmaster.com/discovery/v2"

	// Construct the endpoint URL by appending the action and ".json" properly
	endpoint := fmt.Sprintf("%s/%s.json", baseURL, tma.Action)

	// Add the action parameters to the endpoint if they exist
	println("the number of params is: ", len(tma.Parameters))

	// Parse the URL to check for errors
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %v", err)
	}

	// Add the API key and other parameters to the query
	q := u.Query()
	q.Set("apikey", p.TicketmasterApiKey)
	for k, v := range tma.Parameters {
		q.Add(k, v)
	}

	u.RawQuery = q.Encode()

	fmt.Println("The full URL is: ", u.String())

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
	Action     string            `json:"action"`
	Parameters map[string]string `json:"parameters"`
}

// Helper function to convert interface{} to string safely
func toString(value interface{}) string {
	switch v := value.(type) {
	case float64:
		// Convert numeric values to string
		return fmt.Sprintf("%v", v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// AnalyzePromptWithLLM uses an LLM to analyze the prompt and suggest Ticketmaster actions
func AnalyzePromptWithLLM(prompt string) (*TicketmasterAction, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	print("the api key is: ", apiKey)
	endpoint := "https://api.openai.com/v1/chat/completions"

	requestBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a system that dervies API actions and query paramters based on user prompts. Please return only a json object with the action and parameters."},
			{"role": "system", "content": "Query param with date must be of valid format YYYY-MM-DDTHH:mm:ssZ {example: 2020-08-01T14:00:00Z }"},
			{"role": "user", "content": fmt.Sprintf("Given the user's request: '%s', determine the most appropriate Ticketmaster API action and parameters. Return a JSON object with the action and parameters. Consider valid actions such as attractions, classifications, events, venues. Include details on how to use the following query parameters effectively: \n- id (Filter entities by its id)\n- keyword (Keyword to search on)\n- attractionId (Filter by attraction id)\n- venueId (Filter by venue id)\n- postalCode (Filter by postal code / zipcode)\n- latlong (Filter events by latitude and longitude; deprecated)\n- radius (Radius of the area for event search)\n- unit (Unit of the radius, e.g., miles, km)\n- source (Filter entities by source name, e.g., ticketmaster, universe, frontgate)\n- locale (Locale in ISO code format)\n- marketId, startDateTime, endDateTime (Filter events by market, start and end dates)\n- includeTBA, includeTBD (Include events with dates to be announced or defined)\n- size, page (Pagination options)\n- sort (Sorting order of the search results, e.g., 'name,asc', 'date,desc')\n- onsaleStartDateTime, onsaleEndDateTime (Filter events by onsale start and end dates)\n- city, countryCode, stateCode (Filter by geographical location)\n- classificationName, classificationId (Filter by type of event, like genre or segment)\n- includeFamily (Include family-friendly classifications)\n- promoterId, genreId, subGenreId, typeId, subTypeId (Filter by various IDs related to event categorization)\n- geoPoint (Filter events by geoHash)\n- includeSpellcheck (Include spell check suggestions in response)", prompt)},
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

	var intermediate struct {
		Action     string                 `json:"action"`
		Parameters map[string]interface{} `json:"parameters"`
	}
	print("The content is: ", response.Choices[0].Message.Content)

	if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &intermediate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal action from content: %v", err)
	}

	// Convert map[string]interface{} to map[string]string
	params := make(map[string]string)
	for key, value := range intermediate.Parameters {
		params[key] = toString(value)
	}

	action := TicketmasterAction{
		Action:     intermediate.Action,
		Parameters: params,
	}

	return &action, nil
}
