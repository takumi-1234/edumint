# EduMint - AI駆動の教育問題生成プラットフォーム

[![Go Version](https://img.shields.io/badge/Go-1.22.2-blue.svg)](https://go.dev/dl/)
[![Node.js Version](https://img.shields.io/badge/Node.js-18.x-green.svg)](https://nodejs.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**EduMint**は、大学の講義ノートや既存の試験問題（テキスト/PDF形式）をAIで分析し、高品質な演習問題と解答のセットを自動生成する、スケーラブルな教育支援プラットフォームです。

単に問題を再生成するだけでなく、非構造化された講義ノートから試験全体の構成を設計し、それに基づいた問題作成を行うインテリジェントな機能を備えています。

## ✨ 主要機能

-   **多様な入力形式**: プレーンテキストとPDFファイルの両方を入力としてサポート。
-   **インテリジェントな入力分析**: AIが入力内容を「試験問題」か「講義ノート」かを自動で判断し、最適な処理を実行。
    -   **試験問題の場合**: 既存の構造を正確に**抽出**。
    -   **講義ノートの場合**: 内容を理解し、試験構成案を**提案・生成**。
-   **非同期処理アーキテクチャ**: 時間のかかるAI処理はバックグラウンドで実行。ユーザーは待ち時間なく快適にUIを操作可能。
-   **高品質な表示**: 生成された問題と解答は、MarkdownとLaTeXの数式を美しくレンダリング。
-   **管理者向けダッシュボード**: ジョブの実行履歴、ステータス、AIのトークン消費量をリアルタイムで監視。
-   **スケーラビリティ**: マイクロサービスアーキテクチャにより、負荷に応じてAI処理ワーカーを独立してスケール可能。

## 🏛️ アーキテクチャ概要

本プロジェクトは、100万ユーザー規模のトラフィックを想定し、パフォーマンスと耐障害性を重視した**非同期マイクロサービスアーキテクチャ**を採用しています。

```
[User Browser] (Frontend/Admin)
      |
      | (HTTP Request: /generate)
      v
[API Gateway (Go)] ---> [Database (PostgreSQL)]
      | (1. Job登録、Status: 'pending')
      |
      | (2. Job IDをキューにPublish)
      v
[Message Queue (RabbitMQ)]
      |
      | (3. JobをConsume)
      v
[Problem Generator Worker (Go)] <---> [Google Gemini API]
      | (4. AI処理実行)
      |
      | (5. 結果をDBに保存、Status: 'completed'/'failed')
      v
[Database (PostgreSQL)]
      ^
      | (6. フロントエンドがステータスをポーリング)
      |
[Frontend/Admin] <--- [API Gateway (Go)]
      (API Call: /problems/{id}/status)
```

-   **API Gateway**: リクエストの受付とジョブのキューイングに特化した軽量なサービス。
-   **Problem Generator Worker**: 実際にAIとの通信を行う重い処理を担当。負荷に応じてコンテナ数を増減できます。
-   **RabbitMQ**: サービス間の通信を疎結合にし、システム全体の信頼性を担保するメッセージブローカー。

## 🛠️ 技術スタック

-   **バックエンド (API Gateway)**: Go, `gorilla/mux`, `rs/cors`
-   **バックエンド (Worker)**: Go, `google/generative-ai-go`
-   **フロントエンド**: Next.js, React, `react-markdown`
-   **データベース**: PostgreSQL (Image: `postgres:16-alpine`)
-   **メッセージキュー**: RabbitMQ (Image: `rabbitmq:3.13-management-alpine`)
-   **コンテナ化**: Docker, Docker Compose

## 📁 ディレクトリ構造

```
edumint/
├── .env                  # <-- 【重要】環境変数を設定するファイル（手動で作成）
├── docker-compose.yml    # 全サービスのオーケストレーションを定義
│
├── api-gateway/          # HTTPリクエストを受け付けるゲートウェイサービス
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── api/handlers.go
│   │   ├── queue/rabbitmq.go
│   │   └── storage/db.go
│   └── Dockerfile
│
├── problem-generator-worker/ # AIとの通信を行う非同期ワーカー
│   ├── cmd/worker/main.go
│   ├── internal/
│   │   ├── models/models.go
│   │   ├── processor/processor.go
│   │   ├── queue/rabbitmq.go
│   │   ├── services/gemini/gemini_service.go
│   │   └── storage/storage_service.go
│   └── Dockerfile
│
├── db/                     # データベース初期化用SQL
│   └── init.sql
│
├── frontend/               # ユーザー向けUI (Next.js)
│   ├── components/
│   ├── pages/
│   └── Dockerfile
│
└── admin-dashboard/        # 管理者向けUI (Next.js)
    ├── pages/
    └── Dockerfile
```

## 🚀 環境構築と起動方法

以下の手順に従って、ローカルマシンで開発環境をセットアップします。

### 1. 前提条件
-   **Git**
-   **Docker** および **Docker Compose** (Docker Desktopのインストールを強く推奨)

### 2. プロジェクトのセットアップ

```bash
# 1. このリポジトリをクローンします
git clone <repository-url>
cd edumint

# 2. .envファイルを作成します
# プロジェクトのルートに `.env` という名前のファイルを作成し、
# 以下の内容をコピー＆ペーストしてください。
```

### 3. 環境変数 (`.env`) の設定

以下の内容で、プロジェクトのルートディレクトリに`.env`ファイルを作成してください。

```dotenv
# .env - 必ず自身の情報に書き換えてください

# =================================================
# !!【必須】自身のGoogle Gemini APIキーを設定 !!
# =================================================
GEMINI_API_KEY=YOUR_GEMINI_API_KEY_HERE

# タスクに応じたGeminiモデルの指定
GEMINI_EXTRACTION_MODEL=gemini-1.5-flash-001
GEMINI_GENERATION_MODEL=gemini-1.5-pro-latest

# PostgreSQL データベース設定 (通常は変更不要)
POSTGRES_USER=edumint_user
POSTGRES_PASSWORD=edumint_password
POSTGRES_DB=edumint_db
DATABASE_URL=postgres://edumint_user:edumint_password@db:5432/edumint_db?sslmode=disable

# RabbitMQ 接続設定 (通常は変更不要)
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
```

**【最重要】** `YOUR_GEMINI_API_KEY_HERE`の部分を、あなたが取得した実際のAPIキーに置き換えてください。

### 4. アプリケーションのビルドと起動

ターミナルでプロジェクトのルートディレクトリ(`edumint/`)にいることを確認し、以下のコマンドを実行します。

```bash
# 全てのサービスをビルドして、バックグラウンドで起動します
docker compose up --build -d
```
-   初回起動時は、Dockerイメージのダウンロードとビルドに数分かかります。
-   `-d` (detachedモード) オプションを外して `docker compose up --build` を実行すると、全サービスのログをリアルタイムで確認できます。

### 5. 動作確認

全サービスが起動したら、以下のエンドポイントにアクセスできます。

-   **メインアプリケーション**: `http://localhost:3000`
-   **管理者ダッシュボード**: `http://localhost:3001`
-   **RabbitMQ 管理画面**: `http://localhost:15672` (ユーザー名: `guest`, パスワード: `guest`)

## 💡 アプリケーションの使い方

1.  ブラウザで `http://localhost:3000` を開きます。
2.  「テキスト入力」または「PDFアップロード」を選択し、情報を入力またはファイルを選択します。
3.  「問題を生成」ボタンをクリックします。
4.  UIが「処理中」となり、バックグラウンドで問題生成が開始されます。
5.  処理が完了すると、画面が自動的に更新され、生成された問題が表示されます。
6.  `http://localhost:3001` を開くと、実行したジョブの履歴とステータスを確認できます。

## 🛣️ 今後のロードマップ (Future Work)

-   [ ] **認証・認可**: JWTを用いたユーザー認証とAPI保護の実装。
-   [ ] **検索機能**: Elasticsearch等を導入し、生成された問題の高度な検索機能を実装。
-   [ ] **テスト**: 各サービスのユニットテストと結合テストを記述し、CI/CDパイプラインを構築。
-   [ ] **デプロイ**: KubernetesとHelmチャートを用いて、本番環境へのデプロイ戦略を確立。
-   [ ] **UI/UXの改善**: より洗練されたユーザーインターフェースとエラーハンドリング。

## 🤝 コントリビューション

プルリクエストはいつでも歓迎します。大きな変更については、まずIssueを立てて議論してください。

## 📜 ライセンス

このプロジェクトは [MIT License](LICENSE) の下で公開されています。
