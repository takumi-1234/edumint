// problem-generator-worker/internal/storage/storage_service.go
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"github.com/lib/pq"
	"github.com/your-username/edumint/problem-generator-worker/internal/models"
)

type Service struct{ DB *sql.DB }

func (s *Service) UpdateStatus(id int, status, errMsg string) error {
	query := `UPDATE problems SET processing_status = $1, error_message = $2 WHERE id = $3`
	_, err := s.DB.Exec(query, status, errMsg, id)
	return err
}

func (s *Service) GetInputData(id int) ([]genai.Part, error) {
	var text sql.NullString
	var file []byte
	err := s.DB.QueryRow(`SELECT raw_input_text, raw_input_file FROM problems WHERE id = $1`, id).Scan(&text, &file)
	if err != nil {
		return nil, fmt.Errorf("could not query input data for id %d: %w", id, err)
	}
	if text.Valid && text.String != "" {
		return []genai.Part{genai.Text(text.String)}, nil
	}
	if len(file) > 0 {
		return []genai.Part{genai.Blob{MIMEType: "application/pdf", Data: file}}, nil
	}
	return nil, fmt.Errorf("no input data found for problem id %d", id)
}

// !! 修正: 引数とクエリを新しいデータ構造に合わせる
func (s *Service) SaveResult(id int, ps *models.ProblemStructure, gp *models.GeneratedData, st, gt *genai.UsageMetadata) error {
	majorSectionsJSON, err := json.Marshal(ps.Structure.MajorSections)
	if err != nil {
		return err
	}

	generatedQuestionsJSON, err := json.Marshal(gp)
	if err != nil {
		return err
	}

	var duration sql.NullInt64
	if ps.ExamMeta.ExamDuration != nil {
		duration.Valid = true
		duration.Int64 = int64(*ps.ExamMeta.ExamDuration)
	}

	query := `UPDATE problems SET
		exam_title = $1, duration_minutes = $2, is_open_book = $3, allowed_materials = $4,
		question_format_is_latex = $5, answer_format_is_latex = $6, major_sections = $7,
		generated_questions = $8,
		structure_prompt_tokens = $9, structure_candidates_tokens = $10,
		generation_prompt_tokens = $11, generation_candidates_tokens = $12
		WHERE id = $13`

	_, err = s.DB.Exec(query,
		ps.ExamMeta.ExamTitle, duration, ps.ExamMeta.OpenBook, pq.StringArray(ps.ExamMeta.AllowedMaterials),
		ps.ExamMeta.QuestionFormatIsLatex, ps.ExamMeta.AnswerFormatIsLatex, majorSectionsJSON,
		generatedQuestionsJSON,
		st.PromptTokenCount, st.CandidatesTokenCount,
		gt.PromptTokenCount, gt.CandidatesTokenCount, id,
	)
	return err
}
