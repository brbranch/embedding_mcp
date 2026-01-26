package model

import (
	"encoding/json"
	"testing"
)

// TestRequest_JSONMarshal はRequestが正しくJSONシリアライズされることをテスト
func TestRequest_JSONMarshal(t *testing.T) {
	tests := []struct {
		name     string
		request  *Request
		expected string
	}{
		{
			name: "with params",
			request: &Request{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "memory/store",
				Params:  map[string]any{"text": "test"},
			},
			expected: `{"jsonrpc":"2.0","id":1,"method":"memory/store","params":{"text":"test"}}`,
		},
		{
			name: "without params",
			request: &Request{
				JSONRPC: "2.0",
				ID:      "req-123",
				Method:  "memory/list",
			},
			expected: `{"jsonrpc":"2.0","id":"req-123","method":"memory/list"}`,
		},
		{
			name: "null ID",
			request: &Request{
				JSONRPC: "2.0",
				ID:      nil,
				Method:  "memory/clear",
			},
			expected: `{"jsonrpc":"2.0","id":null,"method":"memory/clear"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("failed to marshal Request: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("expected JSON %q, got %q", tt.expected, string(data))
			}
		})
	}
}

// TestRequest_JSONUnmarshal はJSONからRequestが正しくデシリアライズされることをテスト
func TestRequest_JSONUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		validate func(*testing.T, *Request)
	}{
		{
			name:     "with params object",
			jsonData: `{"jsonrpc":"2.0","id":1,"method":"memory/store","params":{"text":"test"}}`,
			validate: func(t *testing.T, req *Request) {
				if req.JSONRPC != "2.0" {
					t.Errorf("expected JSONRPC %q, got %q", "2.0", req.JSONRPC)
				}
				if req.ID != float64(1) { // JSON numbersはfloat64にデコードされる
					t.Errorf("expected ID %v, got %v", 1, req.ID)
				}
				if req.Method != "memory/store" {
					t.Errorf("expected Method %q, got %q", "memory/store", req.Method)
				}
			},
		},
		{
			name:     "string ID",
			jsonData: `{"jsonrpc":"2.0","id":"req-123","method":"memory/list"}`,
			validate: func(t *testing.T, req *Request) {
				if req.ID != "req-123" {
					t.Errorf("expected ID %q, got %v", "req-123", req.ID)
				}
			},
		},
		{
			name:     "null ID",
			jsonData: `{"jsonrpc":"2.0","id":null,"method":"memory/clear"}`,
			validate: func(t *testing.T, req *Request) {
				if req.ID != nil {
					t.Errorf("expected ID nil, got %v", req.ID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req Request
			if err := json.Unmarshal([]byte(tt.jsonData), &req); err != nil {
				t.Fatalf("failed to unmarshal Request: %v", err)
			}

			tt.validate(t, &req)
		})
	}
}

// TestResponse_JSONMarshal はResponseが正しくJSONシリアライズされることをテスト
func TestResponse_JSONMarshal(t *testing.T) {
	response := &Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  map[string]any{"status": "ok"},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal Response: %v", err)
	}

	expected := `{"jsonrpc":"2.0","id":1,"result":{"status":"ok"}}`
	if string(data) != expected {
		t.Errorf("expected JSON %q, got %q", expected, string(data))
	}
}

// TestErrorResponse_JSONMarshal はErrorResponseが正しくJSONシリアライズされることをテスト
func TestErrorResponse_JSONMarshal(t *testing.T) {
	errorResponse := &ErrorResponse{
		JSONRPC: "2.0",
		ID:      1,
		Error: RPCError{
			Code:    ErrCodeInvalidParams,
			Message: "Invalid parameters",
			Data:    map[string]any{"field": "text"},
		},
	}

	data, err := json.Marshal(errorResponse)
	if err != nil {
		t.Fatalf("failed to marshal ErrorResponse: %v", err)
	}

	expected := `{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"Invalid parameters","data":{"field":"text"}}}`
	if string(data) != expected {
		t.Errorf("expected JSON %q, got %q", expected, string(data))
	}
}

// TestErrorResponse_ParseError_IDNull はパース失敗時のErrorResponseでIDがnullになることをテスト
func TestErrorResponse_ParseError_IDNull(t *testing.T) {
	errorResponse := &ErrorResponse{
		JSONRPC: "2.0",
		ID:      nil,
		Error: RPCError{
			Code:    ErrCodeParseError,
			Message: "Parse error",
		},
	}

	data, err := json.Marshal(errorResponse)
	if err != nil {
		t.Fatalf("failed to marshal ErrorResponse: %v", err)
	}

	expected := `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"Parse error"}}`
	if string(data) != expected {
		t.Errorf("expected JSON %q, got %q", expected, string(data))
	}
}

// TestNewResponse はNewResponseが正しいレスポンスを生成することをテスト
func TestNewResponse(t *testing.T) {
	result := map[string]any{"status": "ok"}
	response := NewResponse(1, result)

	if response.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC %q, got %q", "2.0", response.JSONRPC)
	}
	if response.ID != 1 {
		t.Errorf("expected ID %v, got %v", 1, response.ID)
	}
	if response.Result == nil {
		t.Error("expected Result to be non-nil")
	}
}

// TestNewErrorResponse はNewErrorResponseが正しいエラーレスポンスを生成することをテスト
func TestNewErrorResponse(t *testing.T) {
	data := map[string]any{"field": "text"}
	errorResponse := NewErrorResponse(1, ErrCodeInvalidParams, "Invalid parameters", data)

	if errorResponse.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC %q, got %q", "2.0", errorResponse.JSONRPC)
	}
	if errorResponse.ID != 1 {
		t.Errorf("expected ID %v, got %v", 1, errorResponse.ID)
	}
	if errorResponse.Error.Code != ErrCodeInvalidParams {
		t.Errorf("expected Error Code %d, got %d", ErrCodeInvalidParams, errorResponse.Error.Code)
	}
	if errorResponse.Error.Message != "Invalid parameters" {
		t.Errorf("expected Error Message %q, got %q", "Invalid parameters", errorResponse.Error.Message)
	}
	if errorResponse.Error.Data == nil {
		t.Error("expected Error Data to be non-nil")
	}
}

// TestNewParseError はParseErrorが-32700コードを持ち、IDがnullになることをテスト
func TestNewParseError(t *testing.T) {
	data := "invalid JSON"
	errorResponse := NewParseError(data)

	if errorResponse.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC %q, got %q", "2.0", errorResponse.JSONRPC)
	}
	if errorResponse.ID != nil {
		t.Errorf("expected ID nil, got %v", errorResponse.ID)
	}
	if errorResponse.Error.Code != ErrCodeParseError {
		t.Errorf("expected Error Code %d, got %d", ErrCodeParseError, errorResponse.Error.Code)
	}
	if errorResponse.Error.Data != data {
		t.Errorf("expected Error Data %v, got %v", data, errorResponse.Error.Data)
	}
}

// TestNewMethodNotFound はMethodNotFoundが-32601コードを持つことをテスト
func TestNewMethodNotFound(t *testing.T) {
	method := "unknown/method"
	errorResponse := NewMethodNotFound(1, method)

	if errorResponse.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC %q, got %q", "2.0", errorResponse.JSONRPC)
	}
	if errorResponse.ID != 1 {
		t.Errorf("expected ID %v, got %v", 1, errorResponse.ID)
	}
	if errorResponse.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("expected Error Code %d, got %d", ErrCodeMethodNotFound, errorResponse.Error.Code)
	}
	if errorResponse.Error.Data != method {
		t.Errorf("expected Error Data %v, got %v", method, errorResponse.Error.Data)
	}
}

// TestNewInvalidParams はInvalidParamsが-32602コードを持つことをテスト
func TestNewInvalidParams(t *testing.T) {
	message := "text field is required"
	errorResponse := NewInvalidParams(1, message)

	if errorResponse.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC %q, got %q", "2.0", errorResponse.JSONRPC)
	}
	if errorResponse.ID != 1 {
		t.Errorf("expected ID %v, got %v", 1, errorResponse.ID)
	}
	if errorResponse.Error.Code != ErrCodeInvalidParams {
		t.Errorf("expected Error Code %d, got %d", ErrCodeInvalidParams, errorResponse.Error.Code)
	}
	if errorResponse.Error.Message != message {
		t.Errorf("expected Error Message %q, got %q", message, errorResponse.Error.Message)
	}
}

// TestNewInternalError はInternalErrorが-32603コードを持つことをテスト
func TestNewInternalError(t *testing.T) {
	message := "database connection failed"
	errorResponse := NewInternalError(1, message)

	if errorResponse.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC %q, got %q", "2.0", errorResponse.JSONRPC)
	}
	if errorResponse.ID != 1 {
		t.Errorf("expected ID %v, got %v", 1, errorResponse.ID)
	}
	if errorResponse.Error.Code != ErrCodeInternalError {
		t.Errorf("expected Error Code %d, got %d", ErrCodeInternalError, errorResponse.Error.Code)
	}
	if errorResponse.Error.Message != message {
		t.Errorf("expected Error Message %q, got %q", message, errorResponse.Error.Message)
	}
}
