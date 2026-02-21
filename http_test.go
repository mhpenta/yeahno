package yeahno

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterHTTPUsesTAPServer(t *testing.T) {
	var choice string

	menu := NewSelect[string]().
		Title("Notes").
		Description("Manage notes").
		ToolPrefix("note").
		Options(
			NewOption("Add note", "add").
				Description("Create a note").
				ToolName("add").
				WithField(NewInput().Key("title").Title("Title")).
				MCP(true),
		).
		Value(&choice).
		Handler(func(ctx context.Context, action string, fields map[string]string) (any, error) {
			if action == "add" {
				return map[string]any{"created": fields["title"]}, nil
			}
			return nil, fmt.Errorf("unknown action: %s", action)
		})

	mux := http.NewServeMux()
	if err := menu.RegisterHTTP(mux); err != nil {
		t.Fatalf("register http: %v", err)
	}

	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/tools")
	if err != nil {
		t.Fatalf("GET /tools: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /tools status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	docResp, err := http.Get(ts.URL + "/tools/note_add")
	if err != nil {
		t.Fatalf("GET /tools/note_add: %v", err)
	}
	defer docResp.Body.Close()
	if docResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /tools/note_add status = %d, want %d", docResp.StatusCode, http.StatusOK)
	}

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/tools/note_add/run", strings.NewReader(`{"title":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	runResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /tools/note_add/run: %v", err)
	}
	defer runResp.Body.Close()
	if runResp.StatusCode != http.StatusOK {
		t.Fatalf("POST /tools/note_add/run status = %d, want %d", runResp.StatusCode, http.StatusOK)
	}

	var envelope struct {
		Result map[string]string `json:"result"`
	}
	if err := json.NewDecoder(runResp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	if envelope.Result["created"] != "hello" {
		t.Fatalf("run result = %q, want %q", envelope.Result["created"], "hello")
	}

	badReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/tools/note_add/run", strings.NewReader("{"))
	badReq.Header.Set("Content-Type", "application/json")
	badResp, err := http.DefaultClient.Do(badReq)
	if err != nil {
		t.Fatalf("POST invalid JSON: %v", err)
	}
	defer badResp.Body.Close()
	if badResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST invalid JSON status = %d, want %d", badResp.StatusCode, http.StatusBadRequest)
	}

	var badBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(badResp.Body).Decode(&badBody); err != nil {
		t.Fatalf("decode invalid JSON error body: %v", err)
	}
	if badBody.Code != "invalid_request" {
		t.Fatalf("error code = %q, want %q", badBody.Code, "invalid_request")
	}

	notFoundResp, err := http.Get(ts.URL + "/tools/nope")
	if err != nil {
		t.Fatalf("GET /tools/nope: %v", err)
	}
	defer notFoundResp.Body.Close()
	if notFoundResp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET /tools/nope status = %d, want %d", notFoundResp.StatusCode, http.StatusNotFound)
	}

	var notFoundBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(notFoundResp.Body).Decode(&notFoundBody); err != nil {
		t.Fatalf("decode not found body: %v", err)
	}
	if notFoundBody.Code != "not_found" {
		t.Fatalf("not found code = %q, want %q", notFoundBody.Code, "not_found")
	}
}
