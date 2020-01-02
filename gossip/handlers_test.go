package gossip

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorsOptionsResponse(t *testing.T) {
	methods := "GET, POST"
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080", nil)
	w := httptest.NewRecorder()
	corsOptionsResponse(w, req, DefaultConfig, methods)
	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Errorf("res.StatusCode == %d; want %d", res.StatusCode, http.StatusOK)
	}

	if res.Header.Get("Access-Control-Allow-Origin") != DefaultConfig.Cors.AllowOrigin {
		t.Errorf("\"Access-Control-Allow-Origin\" == %s; want %s", res.Header.Get("Access-Control-Allow-Origin"), DefaultConfig.Cors.AllowOrigin)
	}
	if res.Header.Get("Access-Control-Allow-Headers") != DefaultConfig.Cors.AllowHeaders {
		t.Errorf("\"Access-Control-Allow-Headers\" == %s; want %s", res.Header.Get("Access-Control-Allow-Headers"), DefaultConfig.Cors.AllowHeaders)
	}
	if res.Header.Get("Access-Control-Allow-Methods") != methods {
		t.Errorf("\"Access-Control-Allow-Methods\" == %s; want %s", res.Header.Get("Access-Control-Allow-Methods"), methods)
	}
}

func TestMethodNotAllowedHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080", nil)
	w := httptest.NewRecorder()
	methodNotAllowedHandler(w, req)
	res := w.Result()

	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("res.StatusCode == %d; want %d", res.StatusCode, http.StatusMethodNotAllowed)
	}
}

func TestResponse(t *testing.T) {
	testCases := []struct {
		Message    string
		StatusCode int
	}{
		{"Network Authentication Required", http.StatusNetworkAuthenticationRequired},
	}

	for i, tc := range testCases {
		req := httptest.NewRequest("GET", "http://127.0.0.1:8080", nil)
		w := httptest.NewRecorder()
		response(w, req, tc.StatusCode, tc.Message)
		res := w.Result()

		var msg Response
		json.NewDecoder(res.Body).Decode(&msg)

		if res.StatusCode != tc.StatusCode {
			t.Errorf("res.StatusCode == %d in test case %d; want %d", res.StatusCode, i, tc.StatusCode)
		}

		if msg.Message != tc.Message {
			t.Errorf("msg.Message == %s in test case %d; want %s", msg.Message, i, tc.Message)
		}
	}
}
