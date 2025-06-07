import Head from 'next/head';
import { useState, useEffect, useRef } from 'react';
import MarkdownRenderer from '../components/MarkdownRenderer';

export default function Home() {
    const [inputType, setInputType] = useState('text');
    const [inputText, setInputText] = useState('');
    const [pdfFile, setPdfFile] = useState(null);
    const fileInputRef = useRef(null);

    const [jobId, setJobId] = useState(null);
    const [jobStatus, setJobStatus] = useState('');
    const [jobResult, setJobResult] = useState(null);
    
    const [showAnswers, setShowAnswers] = useState(false);
    const [isShowingAd, setIsShowingAd] = useState(false);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState('');

    const resetState = () => {
        setJobId(null);
        setJobStatus('');
        setJobResult(null);
        setError('');
        setShowAnswers(false);
        setIsLoading(false);
    };

    // ============================================================================
    // !! 修正箇所 !!
    // ジョブ完了後/失敗後に`setJobId(null)`を呼び出し、UIの状態をリセットします。
    // ============================================================================
    useEffect(() => {
        // ポーリングを開始する条件：jobIdがあり、かつステータスが処理中であること
        if (!jobId || (jobStatus !== 'pending' && jobStatus !== 'processing')) {
            return;
        }

        const pollStatus = async () => {
            try {
                const res = await fetch(`http://localhost:8080/api/v1/problems/${jobId}/status`);
                if (!res.ok) {
                    const errData = await res.json().catch(() => ({ message: 'Status check failed.' }));
                    throw new Error(errData.message || 'Status check failed');
                }
                const data = await res.json();
                
                // ステータスが変化した場合のみ更新
                if (data.status !== jobStatus) {
                    setJobStatus(data.status);
                }

                if (data.status === 'completed') {
                    setJobResult(data.generated_output);
                    setJobId(null); // !! 修正: ジョブIDをリセットしてUIを待機状態から解放する
                } else if (data.status === 'failed') {
                    setError(data.error || 'Job processing failed.');
                    setJobId(null); // !! 修正: エラー時もジョブIDをリセットする
                }
            } catch (err) {
                setError(err.message);
                setJobStatus('failed');
                setJobId(null); // !! 修正: 通信エラー時もジョブIDをリセットする
            }
        };

        const intervalId = setInterval(pollStatus, 3000); // 3秒ごとにステータスをチェック
        
        // クリーンアップ関数：コンポーネントがアンマウントされるか、依存関係が変わる際にインターバルを停止
        return () => clearInterval(intervalId);

    }, [jobId, jobStatus]); // jobIdかjobStatusが変わるたびにこのeffectは再評価される

    const handleGenerateProblem = async (e) => {
        e.preventDefault();
        resetState();
        setIsLoading(true);

        try {
            let body;
            let headers = {};
            if (inputType === 'pdf' && pdfFile) {
                body = new FormData();
                body.append('pdfFile', pdfFile);
            } else if (inputType === 'text' && inputText) {
                body = inputText;
                headers['Content-Type'] = 'application/json';
            } else {
                throw new Error("Input is empty. Please provide text or a PDF file.");
            }

            const res = await fetch('http://localhost:8080/api/v1/generate', { method: 'POST', headers, body });
            if (res.status !== 202) {
                const errText = await res.text();
                throw new Error(errText || 'Failed to submit job to the server.');
            }

            const data = await res.json();
            setJobId(data.problem_id);
            setJobStatus('pending'); // ポーリングを開始
        } catch (err) {
            setError(err.message);
        } finally {
            setIsLoading(false);
        }
    };

    const handleShowAnswers = async () => {
        setIsShowingAd(true);
        await new Promise(resolve => setTimeout(resolve, 3000));
        setIsShowingAd(false);
        setShowAnswers(true);
    };

    return (
        <div className="container">
            <Head>
                <title>EduMint - AI Problem Generator</title>
                <meta name="description" content="Generate university-level problems from text or PDF using AI" />
                <link rel="icon" href="/favicon.ico" />
            </Head>

            <header className="header">
                <h1>EduMint 問題生成プラットフォーム</h1>
            </header>

            <main className="main-content">
                <section className="section">
                    <h2>1. 問題の元となる情報を入力</h2>
                    <div className="input-type-selector">
                        <button onClick={() => setInputType('text')} className={inputType === 'text' ? 'active' : ''}>テキスト入力</button>
                        <button onClick={() => setInputType('pdf')} className={inputType === 'pdf' ? 'active' : ''}>PDFアップロード</button>
                    </div>

                    <form onSubmit={handleGenerateProblem} className="form">
                        {inputType === 'text' ? (
                            <textarea
                                id="problem-text-input" name="problem-text"
                                className="textarea"
                                value={inputText}
                                onChange={(e) => setInputText(e.target.value)}
                                placeholder="講義ノートや過去問のテキストをここに貼り付け..."
                                rows={10}
                                disabled={isLoading || !!jobId} // !! 修正: jobIdの存在をbooleanに変換
                            />
                        ) : (
                            <div className="file-input-area">
                                <button type="button" onClick={() => fileInputRef.current.click()} className="button file-button" disabled={isLoading || !!jobId}>
                                  ファイルを選択
                                </button>
                                <input id="pdf-file-input" name="pdf-file"
                                    type="file" ref={fileInputRef} onChange={(e) => setPdfFile(e.target.files[0])} accept=".pdf" style={{ display: 'none' }} />
                                {pdfFile && <span className="file-name">{pdfFile.name}</span>}
                            </div>
                        )}
                        <button type="submit" disabled={isLoading || !!jobId} className="button generate-button">
                            {isLoading ? '投入中...' : (jobId ? '処理中' : '問題を生成')}
                        </button>
                    </form>
                </section>
                
                { (isLoading || (jobId && jobStatus !== 'completed' && jobStatus !== 'failed')) &&
                    <div className="status-box">
                        <p>ステータス: {isLoading ? 'ジョブをサーバーに送信中...' : `処理中 (${jobStatus})`}</p>
                        <p>処理が完了すると、結果が自動的に表示されます。このページを離れても処理は続行されます。</p>
                    </div>
                }
                
                { error && <p className="error-message">{error}</p> }

                {jobResult && jobStatus === 'completed' && (
                    <section className="section result-display-section">
                        <h2>2. 生成された問題と解答</h2>
                        
                        <h3>{jobResult.exam_meta?.exam_title || '生成された問題'}</h3>

                        {jobResult.questions && jobResult.questions.length > 0 ? (
                          jobResult.questions.map((q, index) => (
                            <div key={index} className="result-box">
                                <h4>問題 {q.question_index} (トピック: {q.topic || 'N/A'})</h4>
                                <div className="markdown-content">
                                    <MarkdownRenderer markdownContent={q.question_text} />
                                </div>
                                
                                {showAnswers && (
                                    <div className="answer-section">
                                        <h5>解答</h5>
                                        <div className="markdown-content">
                                            <MarkdownRenderer markdownContent={q.answer_text} />
                                        </div>
                                    </div>
                                )}
                            </div>
                          ))
                        ) : (
                          <p>問題が見つかりませんでした。</p>
                        )}
                        
                        { !showAnswers && (jobResult.questions && jobResult.questions.length > 0) &&
                            <div className="show-answer-area">
                                <button onClick={handleShowAnswers} disabled={isShowingAd} className="button answer-button">
                                    {isShowingAd ? '広告表示中...' : 'すべての解答を表示する'}
                                </button>
                            </div>
                        }
                        
                        { isShowingAd && (
                            <div className="ad-placeholder">
                                <p>広告が表示されています...</p>
                                <p>(しばらくお待ちください)</p>
                            </div>
                        )}
                    </section>
                )}
            </main>

            <style jsx>{`
                .container { max-width: 900px; margin: 0 auto; padding: 1rem; font-family: sans-serif; }
                .header { text-align: center; padding: 1rem 0; border-bottom: 1px solid #e0e0e0; margin-bottom: 1.5rem; }
                .section { background-color: #fff; margin-bottom: 1.5rem; padding: 1.5rem; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.06); }
                .form { display: flex; flex-direction: column; gap: 1rem; }
                .button { padding: 0.75rem 1.5rem; border-radius: 6px; border: none; font-size: 1rem; cursor: pointer; transition: background-color 0.2s; }
                .button:disabled { background-color: #ccc; cursor: not-allowed; }
                .generate-button { background-color: #0070f3; color: white; }
                .textarea { width: 100%; min-height: 150px; padding: 0.5rem; font-size: 1rem; border: 1px solid #ccc; border-radius: 4px; }
                .error-message { color: #e74c3c; background: #fbeae5; padding: 1rem; border-radius: 4px; margin: 1rem 0; white-space: pre-wrap; }
                .status-box { background: #eaf5ff; border: 1px solid #99caff; padding: 1rem; border-radius: 4px; margin: 1rem 0; }
                .result-box { margin-top: 1.5rem; padding: 1.5rem; border: 1px solid #e0e0e0; border-radius: 4px; }
                .markdown-content { background-color: #f9f9f9; padding: 1rem; border-radius: 4px; }
                .show-answer-area { text-align: center; margin: 2rem 0; }
                .answer-button { background-color: #f5a623; color: white; }
                .ad-placeholder { text-align: center; padding: 2rem; border: 2px dashed #ccc; margin: 1rem 0; }
                .input-type-selector { display: flex; margin-bottom: 1rem; }
                .input-type-selector button { flex: 1; padding: 0.5rem; border: 1px solid #ccc; background: #f0f0f0; cursor: pointer; }
                .input-type-selector button.active { background: #0070f3; color: white; border-color: #0070f3; }
                .file-input-area { padding: 1rem; border: 2px dashed #ccc; border-radius: 4px; }
                .answer-section { margin-top: 1rem; padding-top: 1rem; border-top: 1px solid #eee; }
            `}</style>
        </div>
    );
}