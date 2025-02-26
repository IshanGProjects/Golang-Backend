package factories

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
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

type ServiceResponse struct {
	Service string      `json:"service"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error"`
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

	analysisResults, err := sd.OpenAIService.AnalyzePrompt(prompt)
	if err != nil {
		log.Println("Error processing prompt:", err)
		http.Error(w, "Failed to analyze the prompt", http.StatusInternalServerError)
		return
	}

	//Testing the response for the result types
	for _, result := range analysisResults {
		log.Printf("Service: %s, Applicability: %s%%\n", result.Service, result.Applicability)
	}

	var serviceResponses []ServiceResponse

	for _, result := range analysisResults {
		service := result.Service
		applicabilityInt, err := strconv.Atoi(result.Applicability)

		if err != nil {
			log.Printf("Error converting applicability to integer: %v", err)
		}

		if applicabilityInt < 90 {
			log.Printf("Skipping service: %s (Applicability: %d%%)\n", service, applicabilityInt)
			serviceResponses = append(serviceResponses, ServiceResponse{
				Service: service,
				Data:    nil,
				Error:   fmt.Sprintf("Applicability below threshold (%d%%)", applicabilityInt),
			})
			continue
		}

		factory, exists := sd.Factories[service]
		if !exists {
			errMsg := fmt.Sprintf("Factory not found for service: %s", service)
			log.Println(errMsg)
			serviceResponses = append(serviceResponses, ServiceResponse{
				Service: service,
				Data:    nil,
				Error:   errMsg,
			})
			continue
		}

		product := factory.CreateProduct()
		rawData, err := product.PerformAction(map[string]string{"prompt": prompt})
		if err != nil {
			log.Printf("Error processing service %s: %v\n", service, err)
			serviceResponses = append(serviceResponses, ServiceResponse{
				Service: service,
				Data:    nil,
				Error:   err.Error(),
			})
			continue
		}

		serviceResponses = append(serviceResponses, ServiceResponse{
			Service: service,
			Data:    rawData,
		})
	}

	respData, _ := json.Marshal(serviceResponses)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}
