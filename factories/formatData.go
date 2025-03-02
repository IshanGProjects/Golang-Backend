package factories

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

type CombinedData struct {
	Service string      `json:"service"`
	Data    interface{} `json:"data"`
}

func FormatData(service string, combinedData []CombinedData) ([]interface{}, error) {
	var parsedActivities []interface{}
	openAiEndpoint := "https://api.openai.com/v1/chat/completions"
	openAiApiKey := os.Getenv("OPENAI_API_KEY")

	if len(combinedData) == 0 {
		log.Println("No data provided for combined formatting.")
		return parsedActivities, nil
	}

	jsonData, err := json.Marshal(combinedData)
	if err != nil {
		return nil, fmt.Errorf("error marshaling data: %v", err)
	}

	//Trimming data to avoid hitting the token limit
	if len(jsonData) > 10000 { // Example threshold, adjust based on needs and token count
		jsonData = jsonData[:10000] // Trim data; consider more intelligent trimming based on content
	}

	correctedData := bytes.ReplaceAll(jsonData, []byte("`"), []byte("'"))
	correctedDataString := string(correctedData)

	requestBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a data extraction assistant that processes raw JSON data from multiple services. Extract activities in a standardized format...",
			},
			{
				"role": "user",
				"content": fmt.Sprintf("Format the following combined raw data into the standardized activity format, where these fields make up a json file:\n\n%s.\n"+
					"Extract activities in a standardized format:\n"+
					"- image: URL or image data for the activity.\n"+
					"- activity_name: Name or title of the activity.\n"+
					"- time: Time or duration of the activity (if available).\n"+
					"- date: Date of the activity (if available).\n"+
					"- location: Location of the activity.\n"+
					"- details: Key highlights or details about the activity.\n"+
					"- link: url to more information about the activity.",
					correctedDataString),
			},
		},
		"max_tokens":  1500,
		"temperature": 0.3,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request body: %v", err)
	}

	req, err := http.NewRequest("POST", openAiEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+openAiApiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(responseData, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response data: %v", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices available in the response")
	}

	llmOutput := response.Choices[0].Message.Content

	// Clean up the output if necessary
	cleanedOutput := strings.Replace(llmOutput, " ", "", -1)

	// Remove Markdown backticks and trim spaces
	cleanedOutput = strings.Trim(cleanedOutput, "` \n")

	// Remove `json` prefix if present
	cleanedOutput = strings.TrimPrefix(cleanedOutput, "json")

	cleanedOutput = strings.ReplaceAll(cleanedOutput, "\n", "")
	cleanedOutput = strings.ReplaceAll(cleanedOutput, "\\n", "")

	var formattedData map[string]interface{}
	if err := json.Unmarshal([]byte(cleanedOutput), &formattedData); err != nil {
		return nil, fmt.Errorf("error parsing formatted data: %v", err)
	}

	for _, activity := range formattedData {
		parsedActivities = append(parsedActivities, activity)
	}

	return parsedActivities, nil
}
