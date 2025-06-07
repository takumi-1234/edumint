package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"github.com/your-username/edumint/api-gateway/internal/api"
	"github.com/your-username/edumint/api-gateway/internal/queue"
	"github.com/your-username/edumint/api-gateway/internal/storage"
)

func main() {
	// Initialize database and message queue connections
	db := storage.ConnectDB()
	queueClient := queue.MustConnect()
	defer db.Close()
	defer queueClient.Close()

	// Create the main handler which holds the DB and Queue clients
	handler := &api.Handler{DB: db, QueueClient: queueClient}

	// Configure the main router using gorilla/mux
	router := mux.NewRouter()

	// Group all API routes under the /api/v1 path prefix
	apiV1 := router.PathPrefix("/api/v1").Subrouter()

	// ============================================================================
	// !! 修正箇所 !!
	// 各エンドポイントで、POSTやGETに加えて、CORSプリフライトリクエスト用の
	// http.MethodOptionsを許可するように設定します。
	// ============================================================================
	apiV1.HandleFunc("/generate", handler.GenerateProblemHandler).Methods(http.MethodPost, http.MethodOptions)
	apiV1.HandleFunc("/problems/{id:[0-9]+}/status", handler.GetProblemStatusHandler).Methods(http.MethodGet, http.MethodOptions)
	apiV1.HandleFunc("/admin/history", handler.GetHistoryHandler).Methods(http.MethodGet, http.MethodOptions)

	// Configure CORS middleware using rs/cors
	c := cors.New(cors.Options{
		// 許可するオリジン（フロントエンドと管理画面のURL）
		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:3001"},

		// 許可するHTTPメソッド
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},

		// 許可するHTTPヘッダー
		AllowedHeaders: []string{"Content-Type", "Authorization"},

		// Cookieなどの資格情報を含むリクエストを許可するかどうか
		AllowCredentials: true,
	})

	// Wrap the router with the CORS middleware
	// これにより、すべてのリクエストはまずCORSミドルウェアを通過します。
	handlerWithCORS := c.Handler(router)

	// Start the server
	log.Println("API Gateway is running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", handlerWithCORS); err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}
