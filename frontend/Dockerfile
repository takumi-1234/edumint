# Stage 1: ビルドステージ
# ここでは、開発用の依存関係も含めて全てをインストールし、Next.jsアプリをビルドします。
FROM node:18-alpine AS builder

# 作業ディレクトリを設定
WORKDIR /app

# 依存関係のキャッシュを有効にするため、package.jsonを先にコピー
COPY package*.json ./

# 開発用も含め、全ての依存関係をインストール
RUN npm install

# アプリケーションのソースコードを全てコピー
COPY . .

# 本番用にアプリケーションをビルド
RUN npm run build


# Stage 2: 本番ステージ
# ここでは、ビルドステージから必要な成果物だけをコピーし、最小限のイメージを作成します。
FROM node:18-alpine

# 作業ディレクトリを設定
WORKDIR /app

# セキュリティ向上のため、非rootユーザーを作成
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# ビルドステージから、ビルドされた.nextフォルダをコピー
COPY --from=builder /app/.next ./.next

# ビルドステージから、publicフォルダが存在する場合にのみコピー
RUN if [ -d /app/public ]; then cp -r /app/public ./public; fi

# 本番用の依存関係をインストールするためにpackage.jsonをコピー
COPY --from=builder /app/package.json ./package.json


# ============================================================================
# !! 最終的なエラー解決のための修正箇所 !!
# 'npm start' (内部的に 'next start') を実行するために、'next'コマンドが必要です。
# '--omit=dev' を付けることで、本番環境に不要な開発用パッケージ(devDependencies)を除外し、
# 必要な本番用の依存関係(dependencies)のみをインストールします。
# ============================================================================
RUN npm install --omit=dev


# 作成した非rootユーザーに切り替え
USER appuser

# アプリケーションがリッスンするポートを公開
EXPOSE 3000

# Next.jsが使用するポートの環境変数を設定
ENV PORT 3000

# コンテナ起動時に実行するコマンド
CMD ["npm", "start"]