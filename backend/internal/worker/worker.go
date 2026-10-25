package worker

import (
	"log"

	"github.com/hibiken/asynq"
)

func Start(redisAddr, redisPassword string) {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     redisAddr,
			Password: redisPassword,
			// For external redis
			// TLSConfig: &tls.Config{
			// 	MinVersion: tls.VersionTLS12,
			// },
		},
		asynq.Config{
			Concurrency:  3,
			ErrorHandler: asynq.ErrorHandlerFunc(HandleGradingError),
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc("assignment:auto_grading", HandleAutoGrading)

	if err := srv.Run(mux); err != nil {
		log.Fatal(err)
	}
}
