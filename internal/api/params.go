package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// parseChatID extracts chat_id from the request (supports both form/query params and JSON body).
func parseChatID(r *http.Request) (int64, bool) {
	return parseIntParam(r, "chat_id")
}

// parseIntParam extracts an int64 parameter from the request.
func parseIntParam(r *http.Request, name string) (int64, bool) {
	s := parseStringParam(r, name)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// parseStringParam extracts a string parameter from query, form data, or JSON body.
func parseStringParam(r *http.Request, name string) string {
	// Try query params first
	if v := r.URL.Query().Get(name); v != "" {
		return v
	}

	// Try form data
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/x-www-form-urlencoded") || strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseForm(); err == nil {
			if v := r.FormValue(name); v != "" {
				return v
			}
		}
	}

	// Try JSON body
	if strings.HasPrefix(ct, "application/json") {
		return parseJSONBodyParam(r, name)
	}

	// Fallback: try form value anyway
	if v := r.FormValue(name); v != "" {
		return v
	}

	return ""
}

// parseJSONBodyParam reads a JSON body and extracts a parameter.
// parseJSONBodyParam reads a JSON body and extracts a parameter by name.
func parseJSONBodyParam(r *http.Request, name string) string {
	if r.Body == nil {
		return ""
	}

	// Read body with size limit (10 MB)
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		return ""
	}
	r.Body.Close()

	// Re-create body for potential future reads
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	var data map[string]json.RawMessage
	if err := json.Unmarshal(body, &data); err != nil {
		return ""
	}

	raw, ok := data[name]
	if !ok {
		return ""
	}

	// Try as string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	// Return raw JSON for non-string values (numbers, objects, arrays)
	return strings.TrimSpace(string(raw))
}

// parseJSON unmarshals a JSON string into a target.
func parseJSON(s string, v any) error {
	return json.Unmarshal([]byte(s), v)
}
