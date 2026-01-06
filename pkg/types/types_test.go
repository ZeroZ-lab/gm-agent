package types

import (
	"regexp"
	"testing"
)

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		fn     func() string
	}{
		{"event", "evt_", GenerateEventID},
		{"command", "cmd_", GenerateCommandID},
		{"task", "tsk_", GenerateTaskID},
		{"goal", "gol_", GenerateGoalID},
		{"patch", "pch_", GeneratePatchID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := tt.fn()
			if ok, _ := regexp.MatchString("^"+regexp.QuoteMeta(tt.prefix)+"[0-9A-HJKMNP-TV-Z]{26}$", id); !ok {
				t.Fatalf("generated id %s does not have expected prefix %s", id, tt.prefix)
			}
		})
	}
}

func TestBaseCommand(t *testing.T) {
	bc := NewBaseCommand("call")
	if bc.ID == "" {
		t.Fatalf("expected id to be set")
	}
	if bc.CommandType() != "call" {
		t.Fatalf("unexpected command type %s", bc.CommandType())
	}
}

func TestBaseEvent(t *testing.T) {
	evt := NewBaseEvent("test", "actor", "subject")
	if evt.ID == "" {
		t.Fatalf("expected id to be set")
	}
	if evt.EventType() != "test" || evt.EventActor() != "actor" || evt.EventSubject() != "subject" {
		t.Fatalf("unexpected event fields: %+v", evt)
	}
	if evt.EventTimestamp().IsZero() {
		t.Fatalf("expected timestamp to be set")
	}
}

func TestNewStateAndClone(t *testing.T) {
	st := NewState()
	if st.Version != 0 || len(st.Goals) != 0 {
		t.Fatalf("unexpected initial state: %+v", st)
	}
	if st.Tasks == nil || st.Artifacts == nil || st.Locks == nil || st.Context == nil {
		t.Fatalf("expected collections to be initialized")
	}

	st.Goals = append(st.Goals, Goal{ID: "g1"})
	clone := st.Clone()
	if clone == st {
		t.Fatalf("expected clone to be a different instance")
	}
	if len(clone.Goals) != len(st.Goals) {
		t.Fatalf("expected goals to be copied")
	}
}

func TestToolStructures(t *testing.T) {
	tool := Tool{Name: "sample", Description: "desc", Parameters: JSONSchema{"type": "object"}}
	if tool.Parameters["type"] != "object" {
		t.Fatalf("unexpected parameters: %+v", tool.Parameters)
	}

	tc := ToolCall{ID: "id1", Name: "t", Arguments: "{}"}
	if tc.Arguments != "{}" || tc.Name != "t" {
		t.Fatalf("unexpected tool call: %+v", tc)
	}

	cp := Checkpoint{ID: "ck", StateVersion: 1}
	if cp.ID != "ck" || cp.StateVersion != 1 {
		t.Fatalf("unexpected checkpoint: %+v", cp)
	}
}
