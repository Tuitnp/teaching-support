package queue

import (
	// "crypto/tls"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

const AutoGradingTimeout = 30 * time.Minute
const AutoGradingMaxRetry = 3

var Client interface {
	EnqueueContext(context.Context, *asynq.Task, ...asynq.Option) (*asynq.TaskInfo, error)
}
var Inspector interface {
	GetTaskInfo(string, string) (*asynq.TaskInfo, error)
}

func Init(redisAddr, redisPassword string) {
	Client = asynq.NewClient(asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
		// TLSConfig: &tls.Config{
		// 	MinVersion: tls.VersionTLS12,
		// },
	})
	Inspector = asynq.NewInspector(asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword})
}

func EnqueueAutoGrading(payload AutoGradingPayload) error {
	if Client == nil {
		return fmt.Errorf("grading queue is not initialized")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypeAutoGrading, data)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = Client.EnqueueContext(ctx, task,
		asynq.TaskID(payload.JobID),
		asynq.MaxRetry(AutoGradingMaxRetry),
		asynq.Timeout(AutoGradingTimeout),
		asynq.Retention(24*time.Hour),
	)

	return err
}

func InspectAutoGrading(jobID string) (*asynq.TaskInfo, error) {
	if Inspector == nil {
		return nil, fmt.Errorf("grading queue inspector is not initialized")
	}
	return Inspector.GetTaskInfo("default", jobID)
}
