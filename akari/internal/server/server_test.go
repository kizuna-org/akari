package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestNewMux(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want int
	}{
		{name: "health check", path: healthProcedure, want: http.StatusOK},
		{name: "unknown route", path: "/unknown", want: http.StatusNotFound},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mux := NewMux()
			server := httptest.NewServer(mux)
			t.Cleanup(server.Close)

			client := connect.NewClient[emptypb.Empty, emptypb.Empty](
				server.Client(),
				server.URL+testCase.path,
			)

			_, err := client.CallUnary(t.Context(), connect.NewRequest(&emptypb.Empty{}))
			if testCase.want == http.StatusOK && err != nil {
				t.Fatalf("CallUnary() error = %v", err)
			}

			if testCase.want == http.StatusNotFound && err == nil {
				t.Fatal("CallUnary() error = nil, want error")
			}
		})
	}
}
