package tools

import (
	"context"
	"testing"
)

// MockPrinter captures output
type MockPrinter struct {
	LastOutput string
}

func (m *MockPrinter) Println(a ...any) {
	if len(a) > 0 {
		m.LastOutput = a[0].(string)
	}
}

func TestHandleTalk(t *testing.T) {
	// Setup Mock
	mock := &MockPrinter{}
	printer = mock
	defer func() { printer = DefaultPrinter{} }() // Reset after test

	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantMsg string
	}{
		{
			name:    "Valid Message",
			input:   `{"message": "Hello World"}`,
			wantErr: false,
			wantMsg: "Hello World",
		},
		{
			name:    "Empty Message",
			input:   `{"message": ""}`,
			wantErr: true,
		},
		{
			name:    "Invalid JSON",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := HandleTalk(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTalk() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && mock.LastOutput != tt.wantMsg {
				t.Errorf("HandleTalk() output = %v, want %v", mock.LastOutput, tt.wantMsg)
			}
		})
	}
}

func TestHandleTaskComplete(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Valid Summary",
			input:   `{"summary": "Done!"}`,
			wantErr: false,
		},
		{
			name:    "Valid Empty Summary (Maybe allowed?)",
			input:   `{"summary": ""}`,
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := HandleTaskComplete(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTaskComplete() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && out == "" {
				t.Error("HandleTaskComplete() returned empty string")
			}
		})
	}
}
