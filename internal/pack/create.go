package pack

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/contextsubstrate/ctx/internal/store"
)

// CreatePack builds an immutable Context Pack from an execution log.
// It stores all content as blobs and produces a content-addressed manifest.
func CreatePack(storeRoot string, log *ExecutionLog) (*Pack, error) {
	// Store system prompt as blob
	sysPromptRef, err := store.WriteBlob(storeRoot, []byte(log.SystemPrompt))
	if err != nil {
		return nil, fmt.Errorf("storing system prompt: %w", err)
	}

	// Store prompts
	prompts := make([]Prompt, len(log.Prompts))
	for i, p := range log.Prompts {
		ref, err := store.WriteBlob(storeRoot, []byte(p.Content))
		if err != nil {
			return nil, fmt.Errorf("storing prompt %d: %w", i, err)
		}
		prompts[i] = Prompt{Role: p.Role, ContentRef: ref}
	}

	// Store inputs
	inputs := make([]Input, len(log.Inputs))
	for i, inp := range log.Inputs {
		data := []byte(inp.Content)
		ref, err := store.WriteBlob(storeRoot, data)
		if err != nil {
			return nil, fmt.Errorf("storing input %d: %w", i, err)
		}
		inputs[i] = Input{Name: inp.Name, ContentRef: ref, Size: int64(len(data))}
	}

	// Store step outputs
	steps := make([]Step, len(log.Steps))
	for i, s := range log.Steps {
		var outputRef string
		if s.Output != "" {
			ref, err := store.WriteBlob(storeRoot, []byte(s.Output))
			if err != nil {
				return nil, fmt.Errorf("storing step %d output: %w", i, err)
			}
			outputRef = ref
		}
		steps[i] = Step{
			Index:         s.Index,
			Type:          s.Type,
			Tool:          s.Tool,
			Parameters:    s.Parameters,
			OutputRef:     outputRef,
			Deterministic: s.Deterministic,
			Timestamp:     s.Timestamp,
		}
	}

	// Store outputs
	outputs := make([]Output, len(log.Outputs))
	for i, o := range log.Outputs {
		ref, err := store.WriteBlob(storeRoot, []byte(o.Content))
		if err != nil {
			return nil, fmt.Errorf("storing output %d: %w", i, err)
		}
		outputs[i] = Output{Name: o.Name, ContentRef: ref}
	}

	// Build manifest (without hash — computed next)
	p := &Pack{
		Version:      "0.1",
		Created:      time.Now().UTC(),
		Model:        Model{Identifier: log.Model.Identifier, Parameters: log.Model.Parameters},
		SystemPrompt: sysPromptRef,
		Prompts:      prompts,
		Inputs:       inputs,
		Steps:        steps,
		Outputs:      outputs,
		Environment: Environment{
			OS:           log.Environment.OS,
			Runtime:      log.Environment.Runtime,
			ToolVersions: log.Environment.ToolVersions,
		},
	}

	// Serialize manifest without hash field (hash IS the content hash of this JSON)
	manifestData, err := canonicalJSON(p)
	if err != nil {
		return nil, fmt.Errorf("serializing manifest: %w", err)
	}

	// Store manifest as blob — the blob hash becomes the pack hash
	hash, err := store.WriteBlob(storeRoot, manifestData)
	if err != nil {
		return nil, fmt.Errorf("storing manifest: %w", err)
	}
	p.Hash = hash

	// Set context_pack back-reference on outputs
	for i := range p.Outputs {
		p.Outputs[i].ContextPack = hash
	}

	return p, nil
}

// CanonicalHash computes the content hash of a pack manifest using canonical JSON.
func CanonicalHash(p *Pack) (string, error) {
	saved := p.Hash
	p.Hash = ""
	defer func() { p.Hash = saved }()

	data, err := canonicalJSON(p)
	if err != nil {
		return "", err
	}
	return store.HashContent(data), nil
}

// canonicalJSON produces deterministic JSON output with sorted keys.
func canonicalJSON(v interface{}) ([]byte, error) {
	// Marshal to JSON
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	// Re-parse and re-marshal with sorted keys to ensure canonical form
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	sorted := sortKeys(raw)
	return json.Marshal(sorted)
}

// sortKeys recursively sorts map keys for canonical JSON output.
func sortKeys(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		sorted := make(map[string]interface{})
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sorted[k] = sortKeys(val[k])
		}
		return sorted
	case []interface{}:
		for i, item := range val {
			val[i] = sortKeys(item)
		}
		return val
	default:
		return val
	}
}

// RegisterPack records a pack hash in the .ctx/packs/ index.
func RegisterPack(storeRoot string, hash string) error {
	_, hexStr, err := store.ParseHash(hash)
	if err != nil {
		return err
	}

	// Write a file named by the hash in packs/
	path := storeRoot + "/packs/" + hexStr
	return writeFileIfNotExists(path, []byte(hash))
}

func writeFileIfNotExists(path string, data []byte) error {
	f, err := openFileExclusive(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}
