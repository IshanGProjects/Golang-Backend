package factories

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type AnalysisResult struct {
	Service       string `json:"service"`
	Applicability string `json:"applicability"`
}

type OpenAIService struct {
	APIKey string
}

func NewOpenAIService() *OpenAIService {
	// Load .env file
	err := godotenv.Load() // This will load the .env file in the same directory as the main.go file
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set.")
	}
	return &OpenAIService{
		APIKey: apiKey,
	}
}

func (o *OpenAIService) AnalyzePrompt(prompt string) ([]AnalysisResult, error) {
	jsonData := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{
				"role": "user",
				"content": fmt.Sprintf(`Given the prompt, "%s" generate a JSON array ranking how applicable each service is for this prompt. Use the format:
				[
  {
	"service": "Ticketing",
	"applicability": "XX"
  },
  {
	"service": "Accommodations",
	"applicability": "XX"
  },
  {
	"service": "Restaurants",
	"applicability": "XX"
  }
]

- "Applicability" reflects the relevance of each service for fulfilling the user's goal.
- Rank each service from 0%% (irrelevant) to 100%% (highly relevant).
- In the JSON object, don't include the percent symbol in the applicability value.
- For context: The "Ticketing" service provides tickets to events, "Accommodations" helps with travel accommodations, and "Restaurants" suggests nearby dining options.
Return only the JSON object as a string`, prompt),
			},
		},
		"max_tokens":  100,
		"temperature": 0.5,
	}

	jsonValue, err := json.Marshal(jsonData)
	if err != nil {
		return nil, fmt.Errorf("error marshaling json data: %w", err)
	}

	request, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", strings.NewReader(string(jsonValue)))
	if err != nil {
		return nil, fmt.Errorf("error creating new request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+o.APIKey)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer response.Body.Close()

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var respData map[string]interface{}
	if err := json.Unmarshal(data, &respData); err != nil {
		return nil, fmt.Errorf("error unmarshaling response data: %w", err)
	}

	return o.FilterOpenAIResponse(respData)
}

func (o *OpenAIService) FilterOpenAIResponse(data map[string]interface{}) ([]AnalysisResult, error) {
	choices, ok := data["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("error: 'choices' is not a slice or is empty")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error: first item in 'choices' is not a map")
	}

	messageMap, ok := choice["message"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error: 'message' is not a map")
	}

	content, ok := messageMap["content"].(string)
	if !ok {
		return nil, fmt.Errorf("error: 'content' is not a string")
	}

	// Directly unmarshal JSON string
	var services []AnalysisResult
	err := json.Unmarshal([]byte(content), &services)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling 'content' into services: %v", err)
	}

	return services, nil
}
