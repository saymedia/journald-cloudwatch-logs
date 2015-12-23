package main

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/coreos/go-systemd/sdjournal"
)

func UnmarshalRecord(journal *sdjournal.Journal, to *Record) error {
	return unmarshalRecord(journal, reflect.ValueOf(to).Elem())
}

func unmarshalRecord(journal *sdjournal.Journal, toVal reflect.Value) error {
	toType := toVal.Type()

	numField := toVal.NumField()

	// This intentionally supports only the few types we actually
	// use on the Record struct. It's not intended to be generic.

	for i := 0; i < numField; i++ {
		fieldVal := toVal.Field(i)
		fieldDef := toType.Field(i)
		fieldType := fieldDef.Type
		fieldTag := fieldDef.Tag
		fieldTypeKind := fieldType.Kind()

		if fieldTypeKind == reflect.Struct {
			// Recursively unmarshal from the same journal
			unmarshalRecord(journal, fieldVal)
		}

		jdKey := fieldTag.Get("journald")
		if jdKey == "" {
			continue
		}

		value, err := journal.GetData(jdKey)
		if err != nil || value == "" {
			fieldVal.Set(reflect.Zero(fieldType))
			continue
		}

		// The value is returned with the key and an equals sign on
		// the front, so we'll trim those off.
		value = value[len(jdKey)+1:]

		switch fieldTypeKind {
		case reflect.Int:
			intVal, err := strconv.Atoi(value)
			if err != nil {
				// Should never happen, but not much we can do here.
				fieldVal.Set(reflect.Zero(fieldType))
				continue
			}
			fieldVal.SetInt(int64(intVal))
			break
		case reflect.String:
			fieldVal.SetString(value)
			break
		default:
			// Should never happen
			panic(fmt.Errorf("Can't unmarshal to %s", fieldType))
		}
	}

	return nil
}

