package prepare

import (
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

const logName = "log"

const (
	DirPermissions     = 0o755
	LogFilePermissions = 0o644
)

func OpenLogDir(dir string) (*os.File, error) {
	logDir := filepath.Clean(filepath.Dir(dir))

	if _, err := os.Stat(logDir); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrap(err, "error when try check log dir: ")
		}

		if err = os.MkdirAll(logDir, DirPermissions); err != nil {
			return nil, errors.Wrap(err, "error when try add log dir: ")
		}
	}

	t := time.Now().UTC()
	timeString := t.Format(time.RFC3339)
	fileName := timeString + "-" + logName + ".log"

	file, err := os.OpenFile(filepath.Join(logDir, fileName),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		LogFilePermissions,
	)
	if err != nil {
		return nil, errors.Wrap(err, "error when try open log file: ")
	}

	return file, nil
}
