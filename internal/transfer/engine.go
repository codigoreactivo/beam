package transfer

import (
	"context"
	"fmt"

	"github.com/jesusjhoel/beam/internal/config"
)

type Job struct {
	LocalPath  string
	RemotePath string
	Project    *config.Project
}

type Result struct {
	Job     Job
	Bytes   int64
	Err     error
}

type Engine struct {
	cfg *config.GlobalConfig
}

func NewEngine(cfg *config.GlobalConfig) *Engine {
	return &Engine{cfg: cfg}
}

func (e *Engine) Upload(ctx context.Context, job Job) Result {
	_ = ctx
	return Result{Job: job, Err: fmt.Errorf("not implemented")}
}

func (e *Engine) Download(ctx context.Context, job Job) Result {
	_ = ctx
	return Result{Job: job, Err: fmt.Errorf("not implemented")}
}
