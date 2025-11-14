package conflata

import (
	"encoding"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

type DecodeFunc func(raw string, targetType reflect.Type) (any, error)

var builtinDecoders = map[string]DecodeFunc{
	"json": decodeJSON,
	"xml":  decodeXML,
	"text": decodeTextFormat,
}

var textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
var jsonUnmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
var timeDurationType = reflect.TypeOf(time.Duration(0))

func decodeJSON(raw string, targetType reflect.Type) (any, error) {
	holder := reflect.New(targetType)
	if err := json.Unmarshal([]byte(raw), holder.Interface()); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}
	return holder.Elem().Interface(), nil
}

func decodeXML(raw string, targetType reflect.Type) (any, error) {
	holder := reflect.New(targetType)
	if err := xml.Unmarshal([]byte(raw), holder.Interface()); err != nil {
		return nil, fmt.Errorf("xml decode: %w", err)
	}
	return holder.Elem().Interface(), nil
}

func decodeTextFormat(raw string, targetType reflect.Type) (any, error) {
	ptrType := reflect.PointerTo(targetType)
	if ptrType.Implements(textUnmarshalerType) {
		dest := reflect.New(targetType)
		if err := dest.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(raw)); err != nil {
			return nil, fmt.Errorf("text decode: %w", err)
		}
		return dest.Elem().Interface(), nil
	}
	return decodePrimitive(raw, targetType)
}

func decodePrimitive(raw string, targetType reflect.Type) (any, error) {
	switch targetType.Kind() {
	case reflect.String:
		return raw, nil
	case reflect.Bool:
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("parse bool: %w", err)
		}
		return v, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if targetType == timeDurationType {
			d, err := time.ParseDuration(raw)
			if err != nil {
				return nil, fmt.Errorf("parse duration: %w", err)
			}
			return d, nil
		}
		v, err := strconv.ParseInt(raw, 10, targetType.Bits())
		if err != nil {
			return nil, fmt.Errorf("parse int: %w", err)
		}
		return reflect.ValueOf(v).Convert(targetType).Interface(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v, err := strconv.ParseUint(raw, 10, targetType.Bits())
		if err != nil {
			return nil, fmt.Errorf("parse uint: %w", err)
		}
		return reflect.ValueOf(v).Convert(targetType).Interface(), nil
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(raw, targetType.Bits())
		if err != nil {
			return nil, fmt.Errorf("parse float: %w", err)
		}
		return reflect.ValueOf(v).Convert(targetType).Interface(), nil
	case reflect.Slice:
		if targetType.Elem().Kind() == reflect.Uint8 {
			return []byte(raw), nil
		}
		fallthrough
	case reflect.Struct, reflect.Array, reflect.Map, reflect.Interface:
		return decodeJSON(raw, targetType)
	default:
		ptrType := reflect.PointerTo(targetType)
		switch {
		case ptrType.Implements(textUnmarshalerType):
			return decodeTextFormat(raw, targetType)
		case ptrType.Implements(jsonUnmarshalerType):
			holder := reflect.New(targetType)
			if err := holder.Interface().(json.Unmarshaler).UnmarshalJSON([]byte(raw)); err != nil {
				return nil, fmt.Errorf("json decode: %w", err)
			}
			return holder.Elem().Interface(), nil
		default:
			return nil, fmt.Errorf("unsupported target type %s", targetType)
		}
	}
}
