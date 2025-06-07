package processor

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/your-username/edumint/problem-generator-worker/internal/services/gemini"
	"github.com/your-username/edumint/problem-generator-worker/internal/storage"
)

type Processor struct {
	StorageService *storage.Service
	GeminiService  *gemini.Service
}

// ProcessJob orchestrates the entire problem generation process for a single job.
func (p *Processor) ProcessJob(body []byte) {
	var jobData map[string]int
	if err := json.Unmarshal(body, &jobData); err != nil {
		log.Printf("Error unmarshalling job data: %v", err)
		return
	}
	problemID := jobData["problem_id"]
	log.Printf("Processing job for problem ID: %d", problemID)

	// Helper function to handle errors and update the DB status.
	handleError := func(err error, stage string) {
		errorMsg := fmt.Sprintf("failed at stage '%s': %v", stage, err)
		log.Printf("Error processing job %d: %s", problemID, errorMsg)
		p.StorageService.UpdateStatus(problemID, "failed", errorMsg)
	}

	// 1. Update job status to 'processing'
	if err := p.StorageService.UpdateStatus(problemID, "processing", ""); err != nil {
		handleError(err, "update_status_processing")
		return
	}

	// 2. Retrieve input data from DB
	inputParts, err := p.StorageService.GetInputData(problemID)
	if err != nil {
		handleError(err, "get_input_data")
		return
	}

	// 3. Call Gemini for structure extraction
	problemStructure, structureTokens, err := p.GeminiService.ExtractStructure(inputParts...)
	if err != nil {
		handleError(err, "extract_structure")
		return
	}

	// 4. Call Gemini for problem generation
	generated, generationTokens, err := p.GeminiService.GenerateProblem(problemStructure)
	if err != nil {
		handleError(err, "generate_problem")
		return
	}

	// 5. Save the successful result to the database
	if err := p.StorageService.SaveResult(problemID, problemStructure, generated, structureTokens, generationTokens); err != nil {
		handleError(err, "save_result")
		return
	}

	// 6. Final status update to 'completed'
	if err := p.StorageService.UpdateStatus(problemID, "completed", ""); err != nil {
		handleError(err, "update_status_completed")
	}

	log.Printf("Successfully processed job for problem ID: %d", problemID)
}
