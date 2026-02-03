package model

// Request はJSON-RPC 2.0リクエスト
type Request struct {
	JSONRPC string `json:"jsonrpc"`          // 常に "2.0"
	ID      any    `json:"id"`               // string | number | null
	Method  string `json:"method"`           // メソッド名
	Params  any    `json:"params,omitempty"` // 任意のオブジェクト、省略可
}

// Response はJSON-RPC 2.0レスポンス（成功時）
type Response struct {
	JSONRPC string `json:"jsonrpc"` // 常に "2.0"
	ID      any    `json:"id"`      // リクエストのIDと同一
	Result  any    `json:"result"`  // 結果オブジェクト
}

// ErrorResponse はJSON-RPC 2.0エラーレスポンス
type ErrorResponse struct {
	JSONRPC string   `json:"jsonrpc"` // 常に "2.0"
	ID      any      `json:"id"`      // リクエストのIDと同一（パース失敗時はnull）
	Error   RPCError `json:"error"`   // エラーオブジェクト
}

// RPCError はJSON-RPC 2.0エラーオブジェクト
type RPCError struct {
	Code    int    `json:"code"`           // エラーコード
	Message string `json:"message"`        // エラーメッセージ
	Data    any    `json:"data,omitempty"` // 追加情報、省略可
}

// JSON-RPC 2.0 標準エラーコード
const (
	ErrCodeParseError     = -32700 // Invalid JSON
	ErrCodeInvalidRequest = -32600 // Invalid Request
	ErrCodeMethodNotFound = -32601 // Method not found
	ErrCodeInvalidParams  = -32602 // Invalid params
	ErrCodeInternalError  = -32603 // Internal error
)

// カスタムエラーコード（-32000 〜 -32099 はサーバー予約）
const (
	ErrCodeAPIKeyMissing    = -32001 // API key not configured
	ErrCodeInvalidKeyPrefix = -32002 // Invalid key prefix (global.* required)
	ErrCodeNotFound         = -32003 // Resource not found
	ErrCodeProviderError    = -32004 // Embedding provider error
	ErrCodeConflict         = -32005 // Resource conflict (e.g., duplicate key)
)

// NewResponse は成功レスポンスを生成
func NewResponse(id any, result any) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

// NewErrorResponse はエラーレスポンスを生成
func NewErrorResponse(id any, code int, message string, data any) *ErrorResponse {
	return &ErrorResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// NewParseError はパースエラーレスポンスを生成（IDはnull）
func NewParseError(data any) *ErrorResponse {
	return &ErrorResponse{
		JSONRPC: "2.0",
		ID:      nil,
		Error: RPCError{
			Code:    ErrCodeParseError,
			Message: "Parse error",
			Data:    data,
		},
	}
}

// NewInvalidRequest は無効リクエストエラーレスポンスを生成
func NewInvalidRequest(id any, data any) *ErrorResponse {
	return &ErrorResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: RPCError{
			Code:    ErrCodeInvalidRequest,
			Message: "Invalid Request",
			Data:    data,
		},
	}
}

// NewMethodNotFound はメソッド未検出エラーレスポンスを生成
func NewMethodNotFound(id any, method string) *ErrorResponse {
	return &ErrorResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: RPCError{
			Code:    ErrCodeMethodNotFound,
			Message: "Method not found",
			Data:    method,
		},
	}
}

// NewInvalidParams は無効パラメータエラーレスポンスを生成
func NewInvalidParams(id any, message string) *ErrorResponse {
	return &ErrorResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: RPCError{
			Code:    ErrCodeInvalidParams,
			Message: message,
			Data:    nil,
		},
	}
}

// NewInternalError は内部エラーレスポンスを生成
func NewInternalError(id any, message string) *ErrorResponse {
	return &ErrorResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: RPCError{
			Code:    ErrCodeInternalError,
			Message: message,
			Data:    nil,
		},
	}
}
