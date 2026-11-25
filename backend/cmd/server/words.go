package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"drawguess/internal/game"
)

const maxWordBytes = 512 << 10

func readWords(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxWordBytes+1))
	if len(data) > maxWordBytes {
		return nil, errors.New("word bank exceeds size limit")
	}
	return data, err
}

func startWordSync(ctx context.Context, source, database string, bank *game.WordBank) {
	cache := filepath.Join(filepath.Dir(database), "word-bank.txt")
	if f, err := os.Open(cache); err == nil {
		data, err := readWords(f)
		f.Close()
		if err == nil {
			_, err = bank.Replace(data)
		}
		if err != nil {
			slog.Warn("invalid cached word bank", "error", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		slog.Warn("read word bank cache", "error", err)
	}
	go func() {
		client := &http.Client{Timeout: 20 * time.Second}
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		etag := ""
		for {
			next, err := refreshWords(ctx, client, source, cache, etag, bank)
			if err != nil && ctx.Err() == nil {
				slog.Warn("refresh word bank", "error", err)
			} else if err == nil {
				etag = next
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func refreshWords(ctx context.Context, client *http.Client, source, cache, etag string, bank *game.WordBank) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.raw+json")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified && etag != "" {
		return etag, nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}
	data, err := readWords(resp.Body)
	if err != nil {
		return "", err
	}
	count, err := bank.Replace(data)
	if err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(filepath.Dir(cache), ".word-bank-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.Write(data)
	closeErr := tmp.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	if err := os.Rename(tmp.Name(), cache); err != nil {
		return "", err
	}
	slog.Info("word bank refreshed", "words", count)
	return resp.Header.Get("ETag"), nil
}
