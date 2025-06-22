// frontend/src/types.ts
export interface Memo {
  ID: string;
  CreatedAt: string; // APIレスポンスが文字列の場合。Date型としてパースも可能
  UpdatedAt: string; // APIレスポンスが文字列の場合。Date型としてパースも可能
  Title: string;
  Content: string;
  UserID: string; // バックエンドの Memo モデルに合わせる
  RelatedMemoIDs: string[];
}

// 必要に応じて他の型定義もここに追加できます
// 例: ユーザー認証関連の型など
export interface User {
  id: string; // バックエンドの User モデルに合わせる (例)
  username: string;
}

// APIレスポンスの型など
export interface AuthResponse {
  token: string;
}

export interface UserResponse {
    id: string;
    username: string;
}
