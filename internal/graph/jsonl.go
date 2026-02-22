package graph

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AppendRecord appends a single JSON-encoded record as a line to a JSONL file.
// Creates the file and parent directories if they don't exist.
func AppendRecord(path string, record any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", path, err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshaling record: %w", err)
	}

	data = append(data, '\n')
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("writing to %s: %w", path, err)
	}

	return nil
}

// ReadRecords reads all JSONL records from a file and deserializes them into
// a slice of the given type T. Returns an empty slice if the file does not exist.
func ReadRecords[T any](path string) ([]T, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	var records []T
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB max line
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec T
		if err := json.Unmarshal(line, &rec); err != nil {
			return nil, fmt.Errorf("parsing line %d of %s: %w", lineNum, path, err)
		}
		records = append(records, rec)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %s: %w", path, err)
	}

	return records, nil
}

// WriteRecords writes a complete JSONL file, replacing any existing content.
// Each record is serialized as a single JSON line. Creates parent directories
// if they don't exist.
func WriteRecords(path string, records []any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", path, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for i, rec := range records {
		data, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshaling record %d: %w", i, err)
		}
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("writing record %d to %s: %w", i, path, err)
		}
		if err := w.WriteByte('\n'); err != nil {
			return fmt.Errorf("writing newline for record %d: %w", i, err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flushing %s: %w", path, err)
	}

	return nil
}
