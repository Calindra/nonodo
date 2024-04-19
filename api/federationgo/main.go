package federationgo

import (
	"log/slog"
	"os"

	log "github.com/jensneuse/abstractlogger"
)

func fallback(sc *ServiceConfig) (string, error) {
	dat, err := os.ReadFile(sc.Name + "/graph/schema.graphqls")
	if err != nil {
		return "", err
	}

	return string(dat), nil
}

// It's just a simple example of graphql federation gateway server, it's NOT a production ready code.
func logger() log.Logger {
	// return slog.NewLogger(slog.Config{
	// 	Level: slog.DebugLevel,
	// })

	return nil
}

func StartServer() {
	slog.Info("Starting federation gateway")

	// datasourceWatcher := NewDatasourcePoller(httpClient, DatasourcePollerConfig{
	// 	Services: []ServiceConfig{
	// 		{Name: "accounts", URL: "http://localhost:4001/query", Fallback: fallback},
	// 		{Name: "products", URL: "http://localhost:4002/query", WS: "ws://localhost:4002/query"},
	// 		{Name: "reviews", URL: "http://localhost:4003/query"},
	// 	},
	// 	PollingInterval: 30 * time.Second,
	// })

}
