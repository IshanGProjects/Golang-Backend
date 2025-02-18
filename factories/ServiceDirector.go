package factories

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type ServiceDirector struct {
	Factories     map[string]AbstractFactory
	OpenAIService *OpenAIService
}

type Product interface {
	PerformAction(data map[string]string) (map[string]interface{}, error)
}

func NewServiceDirector() *ServiceDirector {
	sd := &ServiceDirector{
		Factories:     make(map[string]AbstractFactory),
		OpenAIService: NewOpenAIService(),
	}
	sd.Factories["Ticketing"] = &TicketmasterFactory{}
	return sd
}

// ServiceResponse structure for channel communication
type ServiceResponse struct {
	Service string
	Data    interface{}
	Error   string
}

func (sd *ServiceDirector) ProcessPrompt(w http.ResponseWriter, r *http.Request) {
	var requestBody map[string]string
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	prompt, exists := requestBody["prompt"]
	if !exists {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	// Call OpenAI service with prompt
	analysisResults, err := sd.OpenAIService.AnalyzePrompt(prompt) // Assuming AnalyzePrompt now correctly takes a string prompt and returns results and an error
	if err != nil {
		log.Println("Error processing prompt:", err)
		http.Error(w, "Failed to analyze the prompt", http.StatusInternalServerError)
		return
	}

	var wg sync.WaitGroup
	resultsChan := make(chan *ServiceResponse, len(analysisResults))

	for _, result := range analysisResults {
		wg.Add(1)
		go func(result AnalysisResult) {
			defer wg.Done()
			if result.Applicability < 90 {
				resultsChan <- &ServiceResponse{
					Service: result.Service,
					Data:    nil,
					Error:   fmt.Sprintf("Applicability below threshold (%d%%)", result.Applicability),
				}
				return
			}

			factory, ok := sd.Factories[result.Service]
			if !ok {
				resultsChan <- &ServiceResponse{
					Service: result.Service,
					Data:    nil,
					Error:   "Factory not found for service",
				}
				return
			}

			product := factory.CreateProduct()
			rawData, err := product.PerformAction(map[string]string{"prompt": prompt})
			if err != nil {
				resultsChan <- &ServiceResponse{
					Service: result.Service,
					Data:    nil,
					Error:   err.Error(),
				}
				return
			}

			resultsChan <- &ServiceResponse{
				Service: result.Service,
				Data:    rawData,
				Error:   "",
			}
		}(result)
	}

	wg.Wait()
	close(resultsChan)

	serviceResponses := make([]*ServiceResponse, 0)
	for res := range resultsChan {
		serviceResponses = append(serviceResponses, res)
	}

	// Compile all responses into a single JSON output
	responseData, err := json.Marshal(map[string]interface{}{
		"message":   "Processed all services",
		"responses": serviceResponses,
	})
	if err != nil {
		http.Error(w, "Failed to marshal response data", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseData)
}
