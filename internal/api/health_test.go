package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockPool struct {
	PingFunc func(ctx context.Context) error
}

func (m *mockPool) Ping(ctx context.Context) error {
	if m != nil && m.PingFunc != nil {
		return m.PingFunc(ctx)
	}
	return nil
}

func TestReadyHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mockPool   *mockPool
		wantStatus int
		wantErr    ErrorResponse
		wantResp   HealthResponse
	}{
		{
			name:       "Ready",
			wantStatus: http.StatusOK,
			mockPool: &mockPool{
				PingFunc: func(ctx context.Context) error {
					return nil
				},
			},
			wantResp: HealthResponse{Status: "Ready"},
		}, {
			name: "Service unavailable",
			mockPool: &mockPool{
				PingFunc: func(ctx context.Context) error {
					return errors.New("database timeout")
				},
			},
			wantStatus: http.StatusServiceUnavailable,
			wantErr:    ErrorResponse{Err: "Database is unavailable"},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := http.NewServeMux()
			mux.HandleFunc("GET /ready", ReadyHandler(tt.mockPool))

			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d\nBody: %s", w.Code, tt.wantStatus, w.Body.String())
				return
			}

			if tt.wantResp != (HealthResponse{}) {
				var gotResp HealthResponse
				if err := json.NewDecoder(w.Body).Decode(&gotResp); err != nil {
					t.Fatalf("failed to decode response JSON: %v", err)
				}
				if gotResp != tt.wantResp {
					t.Errorf("got response %+v, want %+v", gotResp, tt.wantErr)
				}
			}

			if tt.wantErr != (ErrorResponse{}) {
				var gotErr ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&gotErr); err != nil {
					t.Fatalf("failed to decode error JSON: %v", err)
				}
				if gotErr != tt.wantErr {
					t.Errorf("got error %+v, want %+v", gotErr, tt.wantErr)
				}
			}

		})
	}
}
