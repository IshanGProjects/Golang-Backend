package factories

import (
	"encoding/json"
	"errors"

	// "fmt"
	"io/ioutil"
	// "log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// TicketmasterFactory Structure
type TicketmasterFactory struct{}

func (f *TicketmasterFactory) CreateProduct() AbstractProduct {
	return &TicketmasterProduct{
		TicketmasterApiKey:  os.Getenv("TICKETMASTER_API_KEY"),
		TicketmasterBaseUrl: "https://app.ticketmaster.com/discovery/v2",
		OpenAiApiKey:        os.Getenv("OPENAI_API_KEY"),
		OpenAiEndpoint:      "https://api.openai.com/v1/chat/completions",
	}
}

// TicketmasterProduct Structure
type TicketmasterProduct struct {
	TicketmasterApiKey  string
	TicketmasterBaseUrl string
	OpenAiApiKey        string
	OpenAiEndpoint      string
}

type TicketmasterAction struct {
	Action string
	Params map[string]string
}

func (p *TicketmasterProduct) PerformAction(request map[string]string) (map[string]interface{}, error) {
	if prompt, ok := request["prompt"]; ok {
		analyzed, err := p.analyzePromptWithLLM(prompt)
		if err != nil {
			return nil, err
		}
		return p.fetchFromApi(analyzed.Action, analyzed.Params)
	}

	action, aok := request["action"]
	params, pok := request["params"]
	if aok && pok {
		paramsMap := make(map[string]string)
		// Assume params are passed as a serialized JSON string for actions not analyzed by LLM
		err := json.Unmarshal([]byte(params), &paramsMap)
		if err != nil {
			return nil, err // Handle JSON parsing error if params need to be parsed
		}
		return p.fetchFromApi(action, paramsMap)
	}
	return nil, errors.New("invalid request: specify 'prompt' or both 'action' and 'params'")
}

func (p *TicketmasterProduct) analyzePromptWithLLM(prompt string) (*TicketmasterAction, error) {
	requestBody, _ := json.Marshal(map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 300,
	})
	req, err := http.NewRequest("POST", p.OpenAiEndpoint, strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.OpenAiApiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var respData struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, err
	}

	var action TicketmasterAction
	if err := json.Unmarshal([]byte(respData.Choices[0].Message.Content), &action); err != nil {
		return nil, err
	}

	return &action, nil
}

func (p *TicketmasterProduct) fetchFromApi(action string, params map[string]string) (map[string]interface{}, error) {
	baseURL := p.TicketmasterBaseUrl + "/" + action + ".json?"
	query := url.Values{}
	for k, v := range params {
		query.Add(k, v)
	}
	query.Add("apikey", p.TicketmasterApiKey)

	resp, err := http.Get(baseURL + query.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
