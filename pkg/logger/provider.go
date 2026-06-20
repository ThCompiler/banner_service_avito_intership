package logger

import (
	"context"
	"io"
	"os"

	"bannersrv/internal/pkg/prepare"

	"github.com/ThCompiler/sdi"
	"github.com/pkg/errors"
)

type Resource struct {
	Logger  *Logger
	logFile *os.File
}

func (r *Resource) Debug(message any, args ...any) {
	r.Logger.Debug(message, args...)
}

func (r *Resource) Info(message any, args ...any) {
	r.Logger.Info(message, args...)
}

func (r *Resource) Warn(message any, args ...any) {
	r.Logger.Warn(message, args...)
}

func (r *Resource) Error(message any, args ...any) {
	r.Logger.Error(message, args...)
}

func (r *Resource) Panic(message any, args ...any) {
	r.Logger.Panic(message, args...)
}

func (r *Resource) Fatal(message any, args ...any) {
	r.Logger.Fatal(message, args...)
}

func (r *Resource) With(key Field, value any) Interface {
	return r.Logger.With(key, value)
}

func NewResource(params Params) (*Resource, error) {
	var out io.Writer = os.Stderr

	var logFile *os.File

	if params.LogDir != "" {
		file, err := prepare.OpenLogDir(params.LogDir)
		if err != nil {
			return nil, errors.Wrap(err, "create logger resource")
		}

		logFile = file
		out = file
	}

	return &Resource{
		Logger:  New(params, out),
		logFile: logFile,
	}, nil
}

func (r *Resource) Close() error {
	if r == nil {
		return nil
	}

	if r.Logger != nil {
		if err := r.Logger.Sync(); err != nil {
			return err
		}
	}

	if r.logFile != nil {
		_ = r.logFile.Close() //nolint:errcheck // best-effort cleanup
	}

	return nil
}

func NewProvider() sdi.Provider[*Resource, Params] {
	return sdi.ProviderFunc[*Resource, Params](func(_ context.Context, params Params) (*Resource, error) {
		return NewResource(params)
	})
}
