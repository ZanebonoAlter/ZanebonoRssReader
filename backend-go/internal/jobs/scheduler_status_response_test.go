package jobs

import (
	"reflect"
	"testing"
)

func TestSchedulerStatusResponseDefinition(t *testing.T) {
	type fieldExpectation struct {
		name    string
		typeOf  reflect.Type
		jsonTag string
	}

	expected := []fieldExpectation{
		{name: "Name", typeOf: reflect.TypeOf(""), jsonTag: "name"},
		{name: "Status", typeOf: reflect.TypeOf(""), jsonTag: "status"},
		{name: "CheckInterval", typeOf: reflect.TypeOf(int64(0)), jsonTag: "check_interval"},
		{name: "NextRun", typeOf: reflect.TypeOf(int64(0)), jsonTag: "next_run"},
		{name: "IsExecuting", typeOf: reflect.TypeOf(false), jsonTag: "is_executing"},
	}

	typ := reflect.TypeOf(SchedulerStatusResponse{})
	if typ.NumField() != len(expected) {
		t.Fatalf("field count = %d, want %d", typ.NumField(), len(expected))
	}

	for _, want := range expected {
		field, ok := typ.FieldByName(want.name)
		if !ok {
			t.Fatalf("missing field %s", want.name)
		}
		if field.Type != want.typeOf {
			t.Fatalf("field %s type = %v, want %v", want.name, field.Type, want.typeOf)
		}
		if field.Tag.Get("json") != want.jsonTag {
			t.Fatalf("field %s json tag = %q, want %q", want.name, field.Tag.Get("json"), want.jsonTag)
		}
	}
}
