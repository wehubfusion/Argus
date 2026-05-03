package emitter

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"unicode/utf8"
)

const (
	jsonActionProduce = "produce"
)

// BuildMonitoringPathWithSuffix returns the blob path for monitoring payloads.
// When pathCtx.IsInput is true, the path ends with "{node_id}-input.json" and dotExt is ignored.
// When pathCtx.IsInput is false, the path ends with "{node_id}" plus dotExt, which must be one of:
// ".json", ".csv", ".xlsx". Any other value falls back to ".json".
func BuildMonitoringPathWithSuffix(pathCtx PathContext, dotExt string) string {
	if pathCtx.ClientID == "" || pathCtx.ProjectID == "" || pathCtx.WorkflowID == "" || pathCtx.RunID == "" || pathCtx.NodeID == "" {
		return ""
	}

	var suffix string
	if pathCtx.IsInput {
		suffix = pathCtx.NodeID + "-input.json"
	} else {
		switch dotExt {
		case ".json", ".csv", ".xlsx":
			suffix = pathCtx.NodeID + dotExt
		default:
			suffix = pathCtx.NodeID + ".json"
		}
	}

	return "monitoring/" + pathCtx.ClientID + "/" + pathCtx.ProjectID + "/" + pathCtx.WorkflowID + "/" + pathCtx.RunID + "/" + suffix
}

// produceRawArtifactFromOutputJSON parses jsonBytes as a single JSON object. If it matches the
// Elysium CSV/XLSX produce envelope (action produce, encoded base64, fileExtension .csv/.xlsx),
// returns decoded bytes and the extension. Otherwise ok is false.
func produceRawArtifactFromOutputJSON(jsonBytes []byte) (raw []byte, dotExt string, ok bool) {
	jsonBytes = bytes.TrimSpace(jsonBytes)
	if len(jsonBytes) == 0 || jsonBytes[0] != '{' {
		return nil, "", false
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &obj); err != nil || obj == nil {
		return nil, "", false
	}

	act, _ := obj["action"].(string)
	if act != jsonActionProduce {
		return nil, "", false
	}

	ext, _ := obj["fileExtension"].(string)
	switch ext {
	case ".csv", ".xlsx":
		dotExt = ext
	default:
		return nil, "", false
	}

	enc, _ := obj["encoded"].(string)
	if enc == "" {
		return nil, "", false
	}

	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil || len(raw) == 0 {
		return nil, "", false
	}

	switch dotExt {
	case ".csv":
		if !utf8.Valid(raw) {
			return nil, "", false
		}
	case ".xlsx":
		if len(raw) < 2 || raw[0] != 'P' || raw[1] != 'K' {
			return nil, "", false
		}
	}

	return raw, dotExt, true
}
