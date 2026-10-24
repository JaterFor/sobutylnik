package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExpoClient_Send_DetectsInvalidToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var messages []expoMessage
		_ = json.NewDecoder(r.Body).Decode(&messages)

		resp := expoResponse{}
		for _, m := range messages {
			if m.To == "valid-token" {
				resp.Data = append(resp.Data, expoReceipt{Status: "ok"})
			} else {
				r := expoReceipt{Status: "error"}
				r.Details.Error = "DeviceNotRegistered"
				resp.Data = append(resp.Data, r)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := &ExpoClient{httpClient: srv.Client()}
	results, err := client.sendTo(context.Background(), srv.URL, []string{"valid-token", "stale-token"}, "title", "body", nil)
	if err != nil {
		t.Fatalf("send error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].InvalidNow {
		t.Error("expected valid-token to not be marked invalid")
	}
	if !results[1].InvalidNow {
		t.Error("expected stale-token to be marked invalid (DeviceNotRegistered)")
	}
}

func TestExpoClient_Send_EmptyTokens(t *testing.T) {
	client := NewExpoClient("")
	results, err := client.Send(context.Background(), nil, "title", "body", nil)
	if err != nil || results != nil {
		t.Fatalf("expected no-op for empty token list, got results=%v err=%v", results, err)
	}
}
