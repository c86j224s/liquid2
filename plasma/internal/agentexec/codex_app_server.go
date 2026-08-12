package agentexec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

type codexRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type codexRPCMessage struct {
	ID     *int            `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *codexRPCError  `json:"error,omitempty"`
	Params struct {
		ThreadID string `json:"threadId"`
		TurnID   string `json:"turnId"`
		Item     struct {
			Type string `json:"type"`
		} `json:"item"`
		Turn struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"turn"`
	} `json:"params,omitempty"`
}

func decodeCodexRPC(reader io.Reader, messages chan<- codexRPCMessage, readErrors chan<- error) {
	defer close(messages)
	decoder := json.NewDecoder(reader)
	for {
		var message codexRPCMessage
		if err := decoder.Decode(&message); err != nil {
			if err != io.EOF {
				readErrors <- err
			}
			return
		}
		messages <- message
	}
}

func waitCodexRPCResponse(ctx context.Context, messages <-chan codexRPCMessage, readErrors <-chan error, id int) error {
	for {
		message, err := nextCodexRPC(ctx, messages, readErrors)
		if err != nil {
			return err
		}
		if message.ID == nil || *message.ID != id {
			continue
		}
		if message.Error != nil {
			return fmt.Errorf("Codex app-server request failed (%d): %s", message.Error.Code, message.Error.Message)
		}
		return nil
	}
}

func waitCodexCompaction(ctx context.Context, messages <-chan codexRPCMessage, readErrors <-chan error, sessionID string) error {
	acknowledged := false
	itemCompleted := false
	turnCompleted := false
	compactionTurnID := ""
	for !acknowledged || !itemCompleted || !turnCompleted {
		message, err := nextCodexRPC(ctx, messages, readErrors)
		if err != nil {
			return err
		}
		if message.ID != nil && *message.ID == 2 {
			if message.Error != nil {
				return fmt.Errorf("Codex compaction request failed (%d): %s", message.Error.Code, message.Error.Message)
			}
			acknowledged = true
		}
		if message.Params.ThreadID != sessionID {
			continue
		}
		switch message.Method {
		case "item/completed":
			if message.Params.Item.Type == "contextCompaction" {
				itemCompleted = true
				compactionTurnID = message.Params.TurnID
			}
		case "turn/completed":
			if compactionTurnID == "" || message.Params.Turn.ID != compactionTurnID {
				continue
			}
			if message.Params.Turn.Status != "completed" {
				return fmt.Errorf("Codex compaction turn ended with status %q", message.Params.Turn.Status)
			}
			turnCompleted = true
		}
	}
	return nil
}

func nextCodexRPC(ctx context.Context, messages <-chan codexRPCMessage, readErrors <-chan error) (codexRPCMessage, error) {
	select {
	case <-ctx.Done():
		return codexRPCMessage{}, ctx.Err()
	case err := <-readErrors:
		return codexRPCMessage{}, err
	case message, ok := <-messages:
		if !ok {
			return codexRPCMessage{}, fmt.Errorf("Codex app-server closed before compaction completed")
		}
		return message, nil
	}
}

type boundedLogBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
	limit  int
}

func (buffer *boundedLogBuffer) Write(value []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	original := len(value)
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining > 0 {
		if len(value) > remaining {
			value = value[:remaining]
		}
		_, _ = buffer.buffer.Write(value)
	}
	return original, nil
}

func (buffer *boundedLogBuffer) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.String()
}
