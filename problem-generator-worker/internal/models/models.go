// problem-generator-worker/internal/models/models.go

package models

// ProblemStructure はAIによる構造抽出の結果を格納します。
type ProblemStructure struct {
	ExamMeta  ExamMeta         `json:"exam_meta"`
	Structure StructureSection `json:"structure"`
}
type ExamMeta struct {
	ExamTitle             string   `json:"exam_title"`
	ExamDuration          *int     `json:"exam_duration,omitempty"`
	OpenBook              bool     `json:"open_book"`
	AllowedMaterials      []string `json:"allowed_materials,omitempty"`
	QuestionFormatIsLatex bool     `json:"question_format_is_latex"`
	AnswerFormatIsLatex   bool     `json:"answer_format_is_latex"`
}
type StructureSection struct {
	MajorSections []MajorSection `json:"major_sections"`
}
type MajorSection struct {
	SectionIndex string        `json:"section_index,omitempty"`
	SectionTitle string        `json:"section_title,omitempty"`
	SubQuestions []SubQuestion `json:"sub_questions"`
}
type SubQuestion struct {
	QuestionIndex string   `json:"question_index,omitempty"`
	Topic         string   `json:"topic,omitempty"`
	Keywords      []string `json:"keywords,omitempty"`
	Difficulty    string   `json:"difficulty,omitempty"`
}

// !! 修正: AIが生成する問題と解答のJSON構造に合わせた新しいモデル
type GeneratedData struct {
	ExamMeta  GeneratedExamMeta   `json:"exam_meta"`
	Questions []GeneratedQuestion `json:"questions"`
}
type GeneratedExamMeta struct {
	ExamTitle             string `json:"exam_title"`
	OpenBook              bool   `json:"open_book"`
	QuestionFormatIsLatex bool   `json:"question_format_is_latex"`
	AnswerFormatIsLatex   bool   `json:"answer_format_is_latex"`
}
type GeneratedQuestion struct {
	QuestionIndex string   `json:"question_index"`
	Topic         string   `json:"topic"`
	Keywords      []string `json:"keywords"`
	Difficulty    string   `json:"difficulty"`
	QuestionText  string   `json:"question_text"`
	AnswerText    string   `json:"answer_text"`
}
