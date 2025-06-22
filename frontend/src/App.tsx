// frontend/src/App.tsx
import { useState, useEffect } from 'react';
import './App.css';
import { Memo } from './types'; // 作成した型定義をインポート

function App() {
  const [memos, setMemos] = useState<Memo[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  // 環境変数からAPIのベースURLを取得 (Viteの場合 import.meta.env.VITE_API_URL)
  // ローカル開発時は .env ファイルで VITE_API_URL=http://localhost:3000 (Goサーバーのポート) のように設定
  const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:3000';


  useEffect(() => {
    const fetchMemos = async () => {
      setLoading(true);
      setError(null);
      try {
        // TODO: 認証トークンをヘッダーに含める処理を追加する
        const token = localStorage.getItem('token'); // 例: トークンをlocalStorageから取得
        const headers: HeadersInit = { 'Content-Type': 'application/json' };
        if (token) {
          headers['Authorization'] = `Bearer ${token}`;
        }

        const response = await fetch(`${API_BASE_URL}/api/memos`, {
           headers: headers,
        });

        if (!response.ok) {
          if (response.status === 401) {
            // トークンが無効または期限切れの場合、トークンをクリアしてエラー表示
            localStorage.removeItem('token');
            throw new Error('認証が必要です。再度ログインしてください。');
          }
          const errorData = await response.json().catch(() => ({ message: `APIエラー: ${response.status} ${response.statusText}` }));
          throw new Error(errorData.message || `APIエラー: ${response.status} ${response.statusText}`);
        }
        const data: Memo[] = await response.json();
        setMemos(data);
      } catch (err) {
        if (err instanceof Error) {
          setError(err.message);
        } else {
          setError('不明なエラーが発生しました。');
        }
        setMemos([]); // エラー時はメモリストをクリア
      } finally {
        setLoading(false);
      }
    };

    fetchMemos();
    // TODO: 認証状態が変更されたら再取得する依存関係を追加 (例: token)
  }, [API_BASE_URL]);

  return (
    <div className="App">
      <header className="App-header">
        <h1>Fast Memos</h1>
        {/* TODO: ログイン/ログアウトUI */}
      </header>
      <main>
        <h2>メモ一覧</h2>
        {/* TODO: メモ作成フォーム */}
        {loading && <p>読み込み中...</p>}
        {error && <p style={{ color: 'red' }}>エラー: {error}</p>}
        {!loading && !error && memos.length === 0 && <p>メモはありません。作成してください。</p>}
        {!loading && !error && memos.length > 0 && (
          <ul>
            {memos.map((memo) => (
              <li key={memo.ID} style={{ border: '1px solid #ccc', margin: '10px', padding: '10px' }}>
                <h3>{memo.Title}</h3>
                <p>{memo.Content}</p>
                <small>作成日時: {new Date(memo.CreatedAt).toLocaleString()}</small>
                <br />
                <small>関連メモID: {memo.RelatedMemoIDs && memo.RelatedMemoIDs.length > 0 ? memo.RelatedMemoIDs.join(', ') : 'なし'}</small>
                {/* TODO: 更新・削除ボタン */}
              </li>
            ))}
          </ul>
        )}
      </main>
    </div>
  );
}

export default App;
