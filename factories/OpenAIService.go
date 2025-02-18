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
	Applicability int    `json:"applicability"`
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
				"role":    "user",
				"content": fmt.Sprintf(`Given the prompt, "%s" generate a JSON array ranking how applicable each service is for this prompt.`, prompt),
			},
		},
		"max_tokens":  100,
		"temperature": 0.5,
	}

	jsonValue, err := json.Marshal(jsonData)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", strings.NewReader(string(jsonValue)))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+o.APIKey)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var respData map[string]interface{}
	if err := json.Unmarshal(data, &respData); err != nil {
		return nil, err
	}

	return o.FilterOpenAIResponse(respData), nil
}

func (o *OpenAIService) FilterOpenAIResponse(data map[string]interface{}) []AnalysisResult {
	choices := data["choices"].([]interface{})
	message := choices[0].(map[string]interface{})["message"].(string)

	var services []AnalysisResult
	if err := json.Unmarshal([]byte(message), &services); err != nil {
		log.Println("Error unmarshaling services:", err)
		return nil // Depending on requirements, might return error
	}

	threshold := 50
	var filteredResults []AnalysisResult
	for _, service := range services {
		if service.Applicability >= threshold {
			filteredResults = append(filteredResults, service)
		}
	}
	return filteredResults
}
