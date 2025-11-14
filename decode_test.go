package conflata

import (
	"reflect"
	"testing"
	"time"
)

func TestDecodePrimitiveCoversCommonTypes(t *testing.T) {
	check := func(expected any, raw string, targetType reflect.Type) {
		t.Helper()
		got, err := decodePrimitive(raw, targetType)
		if err != nil {
			t.Fatalf("decodePrimitive error: %v", err)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expected %v (%T), got %v (%T)", expected, expected, got, got)
		}
	}
	check(true, "true", reflect.TypeOf(true))
	check(int64(42), "42", reflect.TypeOf(int64(0)))
	check(uint32(7), "7", reflect.TypeOf(uint32(0)))
	check(float32(3.14), "3.14", reflect.TypeOf(float32(0)))
	check(time.Second*5, "5s", reflect.TypeOf(time.Duration(0)))
	check([]byte("abc"), "abc", reflect.TypeOf([]byte(nil)))
}

func TestDecodeJSONStruct(t *testing.T) {
	type payload struct {
		Value string `json:"value"`
	}
	target := reflect.TypeOf(payload{})
	got, err := decodeJSON(`{"value":"hello"}`, target)
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if got.(payload).Value != "hello" {
		t.Fatalf("expected hello, got %+v", got)
	}
}

func TestDecodeXML(t *testing.T) {
	type XMLConfig struct {
		Value string `xml:"value"`
	}
	target := reflect.TypeOf(XMLConfig{})
	got, err := decodeXML(`<XMLConfig><value>hi</value></XMLConfig>`, target)
	if err != nil {
		t.Fatalf("decodeXML error: %v", err)
	}
	if got.(XMLConfig).Value != "hi" {
		t.Fatalf("expected hi, got %+v", got)
	}
}
