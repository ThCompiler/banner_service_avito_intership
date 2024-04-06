package prepare

import (
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

const logName = "log"

func OpenLogDir(dir string) (*os.File, error) {
	dir = filepath.Clean(filepath.Dir(dir))

	if _, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrap(err, "error when try check log dir: ")
		}

		if err = os.MkdirAll(dir, 0o755); err != nil {
			return nil, errors.Wrap(err, "error when try add log dir: ")
		}
	}

	t := time.Now().UTC()
	timeString := t.Format(time.RFC3339)
	fileName := timeString + "-" + logName + ".log"

	file, err := os.OpenFile(filepath.Join(dir, fileName),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0o644,
	)
	if err != nil {
		return nil, errors.Wrap(err, "error when try open log file: ")
	}

	return file, nil
}
