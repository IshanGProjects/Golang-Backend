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
	// Make sure the base URL is correct and ends without a slash
	baseURL := "https://api.content.tripadvisor.com/api/v1/location/search"

	// Construct the endpoint URL by appending the action and ".json" properly
	endpoint := fmt.Sprintf("%s/%s.json", baseURL, tra.Action)

	// Add the action parameters to the endpoint if they exist
	println("the number of params is: ", len(tra.Parameters))

	// Parse the URL to check for errors
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %v", err)
	}

	// Add the API key and other parameters to the query
	q := u.Query()
	q.Set("apikey", p.TripadvisorProductApiKey)
	for k, v := range tra.Parameters {
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

// TripadvisorAction contains the action and parameters required for the Tripadvisor API
type TripadvisorAction struct {
	Action     string            `json:"action"`
	Parameters map[string]string `json:"parameters"`
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
			{"role": "user", "content": fmt.Sprintf(`Given the user's request: '%s', determine the most appropriate Tripadvisor API action and parameters. Return a JSON object with the action and parameters. 
Consider valid actions such as:
- searchQuery (string, required): Text to use for searching based on the name of the location.
- category (string): Filters result set based on property type. Valid options are 'hotels', 'attractions', 'restaurants', and 'geos'.
- phone (string): Phone number to filter the search results by (this can be in any format with spaces and dashes but without the '+' sign at the beginning).
- address (string): Address to filter the search results by.
- latLong (string): Latitude/Longitude pair to scope down the search around a specific point, e.g., '42.3455,-71.10767'.
- radius (number > 0): Length of the radius from the provided latitude/longitude pair to filter results.
- radiusUnit (string): Unit for length of the radius. Valid options are 'km', 'mi', 'm' (km=kilometers, mi=miles, m=meters).
- language (string, defaults to 'en'): The language in which to return results (e.g., 'en' for English or 'es' for Spanish) from the list of supported languages.

Include details on how to use the following query parameters effectively:
- id: Filter entities by its id.
- keyword: Keyword to search on.
- attractionId: Filter by attraction id.
- venueId: Filter by venue id.
- postalCode: Filter by postal code / zipcode.
- latlong: Filter events by latitude and longitude (deprecated).
- radius: Radius of the area for event search.
- unit: Unit of the radius, e.g., miles, km.
- source: Filter entities by source name, e.g., ticketmaster, universe, frontgate.
- locale: Locale in ISO code format.
- marketId, startDateTime, endDateTime: Filter events by market, start and end dates.
- includeTBA, includeTBD: Include events with dates to be announced or defined.
- size, page: Pagination options.
- sort: Sorting order of the search results, e.g., 'name,asc', 'date,desc'.
- onsaleStartDateTime, onsaleEndDateTime: Filter events by onsale start and end dates.
- city, countryCode, stateCode: Filter by geographical location.
- classificationName, classificationId: Filter by type of event, like genre or segment.
- includeFamily: Include family-friendly classifications.
- promoterId, genreId, subGenreId, typeId, subTypeId: Filter by various IDs related to event categorization.
- geoPoint: Filter events by geoHash.
- includeSpellcheck: Include spell check suggestions in response.`, prompt)},
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
	content := strings.TrimSpace(response.Choices[0].Message.Content)
	print("The content is: ", content)

	if err := json.Unmarshal([]byte(content), &intermediate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal action from content: %v", err)
	}

	params := make(map[string]string)
	for key, value := range intermediate.Parameters {
		params[key] = toString(value)
	}

	action := TripadvisorAction{
		Action:     intermediate.Action,
		Parameters: params,
	}

	return &action, nil
}
