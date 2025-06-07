-- db/init.sql

CREATE TYPE processing_status AS ENUM ('pending', 'processing', 'completed', 'failed');

CREATE TABLE IF NOT EXISTS problems (
    id SERIAL PRIMARY KEY,
    
    processing_status processing_status DEFAULT 'pending',
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    raw_input_text TEXT, 
    raw_input_file BYTEA, 
    
    -- AIによる構造抽出の結果
    exam_title VARCHAR(255),
    duration_minutes INT,
    is_open_book BOOLEAN,
    allowed_materials TEXT[],
    question_format_is_latex BOOLEAN,
    answer_format_is_latex BOOLEAN,
    major_sections JSONB,

    -- !! 修正: 問題と解答のセット全体をこのJSONBカラムに格納する
    generated_questions JSONB,

    -- トークン使用量
    structure_prompt_tokens INT,
    structure_candidates_tokens INT,
    generation_prompt_tokens INT,
    generation_candidates_tokens INT
);

CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_timestamp ON problems;
CREATE TRIGGER set_timestamp
BEFORE UPDATE ON problems
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX IF NOT EXISTS idx_problems_status ON problems(processing_status);
CREATE INDEX IF NOT EXISTS idx_problems_created_at ON problems(created_at DESC);