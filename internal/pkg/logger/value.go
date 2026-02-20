package logger

import (
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
)

// Any 自动展开任意类型，支持嵌套结构体
func Any(key string, v any) slog.Attr {
	return slog.Attr{
		Key:   key,
		Value: logValue(reflect.ValueOf(v)),
	}
}

func logValue(v reflect.Value) slog.Value {
	// 处理指针
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return slog.StringValue("<nil>")
		}
		v = v.Elem()
	}

	// 基础类型直接返回
	switch v.Kind() {
	case reflect.String:
		return slog.StringValue(v.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return slog.Int64Value(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return slog.Int64Value(int64(v.Uint()))
	case reflect.Float32, reflect.Float64:
		return slog.Float64Value(v.Float())
	case reflect.Bool:
		return slog.BoolValue(v.Bool())
	case reflect.Slice, reflect.Array:
		return sliceValue(v)
	case reflect.Map:
		return mapValue(v)
	case reflect.Struct:
		return structValue(v)
	default:
		return slog.AnyValue(v.Interface())
	}
}

func structValue(v reflect.Value) slog.Value {
	t := v.Type()
	attrs := make([]slog.Attr, 0, v.NumField())

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)

		// 跳过未导出字段
		if !fv.CanInterface() {
			continue
		}

		// 递归处理
		attrs = append(attrs, slog.Attr{
			Key:   field.Name,
			Value: logValue(fv),
		})
	}

	return slog.GroupValue(attrs...)
}

func sliceValue(v reflect.Value) slog.Value {
	if v.Len() > 10 {
		// 太长转 JSON
		return slog.AnyValue(v.Interface())
	}

	attrs := make([]slog.Attr, v.Len())
	for i := 0; i < v.Len(); i++ {
		attrs[i] = slog.Attr{
			Key:   strconv.Itoa(i),
			Value: logValue(v.Index(i)),
		}
	}
	return slog.GroupValue(attrs...)
}

func mapValue(v reflect.Value) slog.Value {
	attrs := make([]slog.Attr, 0, v.Len())
	for _, key := range v.MapKeys() {
		attrs = append(attrs, slog.Attr{
			Key:   fmt.Sprint(key.Interface()),
			Value: logValue(v.MapIndex(key)),
		})
	}
	return slog.GroupValue(attrs...)
}
