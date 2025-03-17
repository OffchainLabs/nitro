package callstack

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"reflect"
	"strconv"
	"time"
)

// FillMapWithStructFields extracts struct fields into a flat map
// now with support for calling whitelisted methods
func FillMapWithStructFields(result map[string]string, input interface{}, prefix string, ignoredFields []string, whitelistedMethods []string) {
	ignored := make(map[string]bool)
	for _, field := range ignoredFields {
		ignored[field] = true
	}

	// Create a set of whitelisted methods for efficient lookup
	whitelist := make(map[string]bool)
	for _, method := range whitelistedMethods {
		whitelist[method] = true
	}

	extractFields(result, input, prefix, ignored, whitelist)
}

func PrintPrettyJson(data map[string]string) {
	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(prettyJSON))
}

func PrintKeyValue(data map[string]string) {
	for k, v := range data {
		fmt.Println(k, v)
	}
}

func extractFields(result map[string]string, input interface{}, prefix string, ignored map[string]bool, whitelist map[string]bool) {
	if input == nil {
		return
	}

	val := reflect.ValueOf(input)
	methodVal := val
	fieldVal := val

	// Handle pointers by dereferencing
	if fieldVal.Kind() == reflect.Ptr {
		if fieldVal.IsNil() {
			return
		}
		fieldVal = fieldVal.Elem()
	}

	// Handle atomic.Pointer
	if fieldVal.Type().String() == "atomic.Pointer" && fieldVal.IsValid() {
		if method := fieldVal.MethodByName("Load"); method.IsValid() {
			loadResult := method.Call(nil)[0].Interface()
			if loadResult != nil {
				extractFields(result, loadResult, prefix, ignored, whitelist)
			}
		}
		return
	}

	// If it's an interface, get the concrete value
	if fieldVal.Kind() == reflect.Interface && !fieldVal.IsNil() {
		fieldVal = fieldVal.Elem()
		methodVal = fieldVal
	}

	// Only handle struct types
	if fieldVal.Kind() != reflect.Struct {
		return
	}

	typ := fieldVal.Type()

	result[buildPath(prefix, "$type")] = typ.String()

	// Call whitelisted methods on the struct if any
	callWhitelistedMethods(result, methodVal, prefix, ignored, whitelist)

	// Iterate through all fields
	for i := 0; i < fieldVal.NumField(); i++ {
		field := fieldVal.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !fieldType.IsExported() {
			continue
		}

		fieldName := fieldType.Name
		fullPath := buildPath(prefix, fieldName)

		// Check if this field should be ignored
		if ignored[fullPath] || ignored[fieldName] {
			continue
		}

		captureValue(result, field, fullPath, ignored, whitelist)
	}
}

func callWhitelistedMethods(result map[string]string, val reflect.Value, prefix string, ignored map[string]bool, whitelist map[string]bool) {
	// Skip if not valid or not addressable
	if !val.IsValid() {
		return
	}

	valueType := val.Type()

	// Check for methods on the value itself
	for i := 0; i < valueType.NumMethod(); i++ {
		method := valueType.Method(i)
		methodName := method.Name

		// Check if method is whitelisted
		if !whitelist[methodName] {
			continue
		}

		// Get the method from value
		methodVal := val.Method(i)

		// Only invoke methods with no arguments
		if method.Type.NumIn() == 1 { // The receiver is the first argument
			// Invoke the method
			returnVals := methodVal.Call(nil)

			// Process return values
			for j, ret := range returnVals {
				methodSuffix := fmt.Sprintf("%s()[%d]", methodName, j)
				captureValue(result, ret, buildPath(prefix, methodSuffix), ignored, whitelist)
			}
		}
	}

	// If we have a pointer, also check if the pointer itself has methods
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		ptrType := reflect.PointerTo(valueType)
		for i := 0; i < ptrType.NumMethod(); i++ {
			method := ptrType.Method(i)
			methodName := method.Name

			// Check if method is whitelisted
			if !whitelist[methodName] {
				continue
			}

			// Create a pointer to the value
			ptrVal := reflect.New(valueType)
			ptrVal.Elem().Set(val)

			// Get the method from the pointer
			methodVal := ptrVal.Method(i)

			// Only invoke methods with no arguments
			if method.Type.NumIn() == 1 { // The receiver is the first argument
				// Invoke the method
				returnVals := methodVal.Call(nil)

				// Process return values
				for j, ret := range returnVals {
					methodSuffix := fmt.Sprintf("%s()[%d]", methodName, j)
					captureValue(result, ret, buildPath(prefix, methodSuffix), ignored, whitelist)
				}
			}
		}
	}
}

func captureValue(result map[string]string, val reflect.Value, prefix string, ignored map[string]bool, whitelist map[string]bool) {
	if !val.IsValid() {
		return
	}

	// Handle pointer field
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	// Handle interface return value - get concrete type
	if val.Kind() == reflect.Interface && !val.IsNil() {
		val = val.Elem()
		if val.Kind() == reflect.Ptr && !val.IsNil() {
			val = val.Elem()
		}
	}

	switch {
	// Handle primitive types
	case isPrimitive(val):
		result[prefix] = formatValue(val)

	// Skip arrays and maps
	case val.Kind() == reflect.Array || val.Kind() == reflect.Slice || val.Kind() == reflect.Map:
		// Handle []byte as primitive
		if val.Kind() == reflect.Slice && val.Type().Elem().Kind() == reflect.Uint8 {
			result[prefix] = formatValue(val)
		}
		// Otherwise ignored
		log.Print("DUMP val ignored:", prefix)

	// Recursively process struct rets
	case val.Kind() == reflect.Struct:
		extractFields(result, val.Interface(), prefix, ignored, whitelist)
	}
}

func isPrimitive(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}

	t := v.Type()
	switch t.Kind() {
	case reflect.Bool, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.String:
		return true
	case reflect.Slice:
		// []byte is primitive
		return t.Elem().Kind() == reflect.Uint8
	}

	// Check specific types by name, not by direct type dependency
	typeName := t.String()
	return typeName == "big.Int" ||
		typeName == "common.Hash" ||
		typeName == "common.Address" ||
		typeName == "time.Time"
}

func formatValue(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}

	switch v.Kind() {
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.String:
		return v.String()
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return "0x" + hex.EncodeToString(v.Bytes())
		}
	}

	// Handle special types
	typeName := v.Type().String()
	switch {
	case typeName == "big.Int":
		if bigInt, ok := v.Interface().(big.Int); ok {
			return bigInt.String()
		}
	case typeName == "time.Time":
		if t, ok := v.Interface().(time.Time); ok {
			return strconv.FormatInt(t.UnixNano(), 10)
		}
	// For types that might have a Hex() method (like common.Hash and common.Address)
	default:
		// Try to call Hex() method if it exists
		if method := v.MethodByName("Hex"); method.IsValid() {
			result := method.Call(nil)
			if len(result) == 1 && result[0].Kind() == reflect.String {
				return result[0].String()
			}
		}
	}

	// Fallback
	return fmt.Sprintf("%v", v.Interface())
}

func buildPath(prefix string, fieldName string) string {
	if prefix == "" {
		return fieldName
	}

	return prefix + "." + fieldName
}
