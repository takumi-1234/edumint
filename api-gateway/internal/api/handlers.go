package api

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/your-username/edumint/api-gateway/internal/queue"
)

const GENERATION_QUEUE = "problem_generation_queue"

type Handler struct {
	DB          *sql.DB
	QueueClient *queue.Client
}

// ProblemHistoryItem defines the structure for the admin dashboard's history view.
type ProblemHistoryItem struct {
	ID                         int       `json:"id"`
	ExamTitle                  string    `json:"exam_title"`
	CreatedAt                  time.Time `json:"created_at"`
	ProcessingStatus           string    `json:"processing_status"`
	ErrorMessage               string    `json:"error_message"`
	StructurePromptTokens      int       `json:"structure_prompt_tokens"`
	StructureCandidatesTokens  int       `json:"structure_candidates_tokens"`
	GenerationPromptTokens     int       `json:"generation_prompt_tokens"`
	GenerationCandidatesTokens int       `json:"generation_candidates_tokens"`
	TotalTokens                int       `json:"total_tokens"`
}

// GenerateProblemHandler accepts a user request, creates a job entry in the DB, and queues it.
func (h *Handler) GenerateProblemHandler(w http.ResponseWriter, r *http.Request) {
	var problemID int
	var err error

	// Create a job entry in the database based on the input type.
	if r.Header.Get("Content-Type") == "application/json" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		err = h.DB.QueryRow(`INSERT INTO problems (raw_input_text) VALUES ($1) RETURNING id`, string(body)).Scan(&problemID)
	} else { // multipart/form-data
		r.ParseMultipartForm(32 << 20) // 32MB limit
		file, _, err := r.FormFile("pdfFile")
		if err != nil {
			http.Error(w, "Invalid file in form data", http.StatusBadRequest)
			return
		}
		defer file.Close()
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Failed to read uploaded file", http.StatusInternalServerError)
			return
		}
		err = h.DB.QueryRow(`INSERT INTO problems (raw_input_file) VALUES ($1) RETURNING id`, fileBytes).Scan(&problemID)
	}

	if err != nil {
		log.Printf("Error creating job in DB: %v", err)
		http.Error(w, "Failed to create job", http.StatusInternalServerError)
		return
	}

	// Publish the job ID to the RabbitMQ queue.
	jobPayload := map[string]int{"problem_id": problemID}
	jobBytes, _ := json.Marshal(jobPayload)
	if err := h.QueueClient.Publish(GENERATION_QUEUE, jobBytes); err != nil {
		log.Printf("Error publishing job to queue: %v", err)
		// Attempt to mark the job as failed in the DB.
		h.DB.Exec(`UPDATE problems SET processing_status = 'failed', error_message = $1 WHERE id = $2`, "Failed to queue job", problemID)
		http.Error(w, "Failed to queue job for processing", http.StatusInternalServerError)
		return
	}

	log.Printf("Job with ID %d has been successfully queued.", problemID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 Accepted: Request received, processing will happen asynchronously.
	json.NewEncoder(w).Encode(map[string]int{"problem_id": problemID})
}

// GetProblemStatusHandler allows the frontend to poll for the result of a job.
func (h *Handler) GetProblemStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid Problem ID", http.StatusBadRequest)
		return
	}

	var status string
	var errorMessage sql.NullString
	var generatedQuestions []byte // JSONBをバイトスライスとして受け取る

	query := `SELECT processing_status, error_message, generated_questions FROM problems WHERE id = $1`
	err = h.DB.QueryRow(query, id).Scan(&status, &errorMessage, &generatedQuestions)

	if err == sql.ErrNoRows {
		http.Error(w, "Problem not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error querying problem status for ID %d: %v", id, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{"problem_id": id, "status": status}
	if status == "completed" {
		// バイトスライスをjson.RawMessageに変換して、JSONとしてそのままフロントに渡す
		response["generated_output"] = json.RawMessage(generatedQuestions)
	} else if status == "failed" {
		response["error"] = errorMessage.String
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetHistoryHandler provides data for the admin dashboard.
func (h *Handler) GetHistoryHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT 
			id, 
			exam_title,
			created_at,
			processing_status,
			error_message,
			structure_prompt_tokens,
			structure_candidates_tokens,
			generation_prompt_tokens,
			generation_candidates_tokens,
			(COALESCE(structure_prompt_tokens, 0) + COALESCE(structure_candidates_tokens, 0) +
			 COALESCE(generation_prompt_tokens, 0) + COALESCE(generation_candidates_tokens, 0)) as total_tokens
		FROM problems
		ORDER BY created_at DESC
		LIMIT 100;
	`
	rows, err := h.DB.Query(query)
	if err != nil {
		http.Error(w, "Failed to retrieve history", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var history []ProblemHistoryItem
	for rows.Next() {
		var item ProblemHistoryItem
		// NULLを許容する型でDBからの値を受け取る
		var examTitle, errMsg sql.NullString
		var s_prompt, s_cand, g_prompt, g_cand, total sql.NullInt64

		if err := rows.Scan(
			&item.ID, &examTitle, &item.CreatedAt, &item.ProcessingStatus, &errMsg,
			&s_prompt, &s_cand, &g_prompt, &g_cand, &total,
		); err != nil {
			log.Printf("Error scanning history row: %v", err)
			continue // エラーが発生した行はスキップ
		}

		// 値を安全に代入
		item.ExamTitle = examTitle.String
		item.ErrorMessage = errMsg.String
		item.StructurePromptTokens = int(s_prompt.Int64)
		item.StructureCandidatesTokens = int(s_cand.Int64)
		item.GenerationPromptTokens = int(g_prompt.Int64)
		item.GenerationCandidatesTokens = int(g_cand.Int64)
		item.TotalTokens = int(total.Int64)

		history = append(history, item)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating history rows: %v", err)
		http.Error(w, "Failed to process history results", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}
