package storage

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/nikmy/meowbot/internal/logger"
	"io/fs"
	"os"
	"time"
)

func newFileStorage[T Indexed](
	fileName string,
	interval time.Duration,
	model Model[T],
	logger logger.Logger,
) *fileStorage[T] {
	return &fileStorage[T]{
		fileName: fileName,
		model:    model,
		interval: interval,
		logger:   logger,
	}
}

type fileStorage[T Indexed] struct {
	fileName string
	model    Model[T]
	interval time.Duration
	logger   logger.Logger
}

func (s *fileStorage[T]) Run(ctx context.Context) error {
	data := s.getData()
	if data != nil {
		s.model.SetData(data)
	}
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.saveData()
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *fileStorage[T]) Close() {

}

func (s *fileStorage[T]) saveData() {
	s.logger.Infof("saving data to %s", s.fileName)
	data := s.model.GetData()
	if data == nil {
		return
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		s.logger.Warnf("json.Marshal error: %s", err)
		return
	}
	err = os.WriteFile(s.fileName, bytes, fs.ModePerm)
	if err != nil {
		s.logger.Warnf("os.WriteFile(%s, ...) error: %s", s.fileName, err)
		return
	}
}

func (s *fileStorage[T]) getData() map[string]T {
	s.logger.Infof("reading data from %s", s.fileName)
	bytes, err := os.ReadFile(s.fileName)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		s.logger.Warnf("os.ReadFile error: %s", err)
		return nil
	}
	var data map[string]T
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		s.logger.Warnf("json.Unmarshal error: %s", err)
		return nil
	}
	return data
}
