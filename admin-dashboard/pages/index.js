import Head from 'next/head';
import { useState, useEffect } from 'react';

export default function AdminHome() {
  const [history, setHistory] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const fetchHistory = async () => {
      try {
        setIsLoading(true);
        const response = await fetch('http://localhost:8080/api/v1/admin/history');
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();
        setHistory(data);
      } catch (err) {
        setError(err.message);
      } finally {
        setIsLoading(false);
      }
    };

    fetchHistory();
    // Poll for updates every 10 seconds
    const intervalId = setInterval(fetchHistory, 10000);
    return () => clearInterval(intervalId);
  }, []);

  const getStatusClass = (status) => {
    switch (status) {
      case 'completed': return 'status-completed';
      case 'failed': return 'status-failed';
      case 'processing': return 'status-processing';
      case 'pending': return 'status-pending';
      default: return '';
    }
  };

  return (
    <div className="container">
      <Head>
        <title>Admin Dashboard - EduMint</title>
      </Head>
      <header className="header">
        <h1>管理者ダッシュボード</h1>
        <p>生成ジョブの履歴 (自動更新)</p>
      </header>
      <main>
        {isLoading && history.length === 0 && <p>履歴を読み込み中...</p>}
        {error && <p className="error">エラー: {error}</p>}
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>試験名</th>
                <th>作成日時</th>
                <th>ステータス</th>
                <th>合計トークン</th>
                <th>エラー</th>
              </tr>
            </thead>
            <tbody>
              {history.map((item) => (
                <tr key={item.id}>
                  <td>{item.id}</td>
                  <td>{item.exam_title || '(タイトル未設定)'}</td>
                  <td>{new Date(item.created_at).toLocaleString()}</td>
                  <td><span className={`status-badge ${getStatusClass(item.processing_status)}`}>{item.processing_status}</span></td>
                  <td>{item.total_tokens.toLocaleString()}</td>
                  <td title={item.error_message}>{item.error_message.substring(0, 50)}{item.error_message.length > 50 ? '...' : ''}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </main>

      <style jsx>{`
        .container { max-width: 1400px; margin: 0 auto; padding: 1rem; }
        .header { text-align: center; margin-bottom: 2rem; border-bottom: 1px solid #dee2e6; padding-bottom: 1rem;}
        .table-container { overflow-x: auto; }
        table { width: 100%; border-collapse: collapse; font-size: 0.9rem; }
        th, td { border: 1px solid #dee2e6; padding: 0.75rem; text-align: left; white-space: nowrap; }
        th { background-color: #f8f9fa; }
        tr:nth-child(even) { background-color: #f8f9fa; }
        .error { color: #dc3545; }
        .status-badge { display: inline-block; padding: 0.25em 0.6em; font-size: 75%; font-weight: 700; line-height: 1; text-align: center; white-space: nowrap; vertical-align: baseline; border-radius: 0.375rem; color: #fff; }
        .status-completed { background-color: #28a745; }
        .status-failed { background-color: #dc3545; }
        .status-processing { background-color: #007bff; }
        .status-pending { background-color: #6c757d; }
      `}</style>
    </div>
  );
}