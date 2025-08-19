package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	"github.com/fxamacker/cbor/v2"
)

const USAGE = `cbor.

Turns CBOR into JSON or JSON into CBOR.

cat file.json | cbor > file.cbor
cat file.cbor | cbor > file.json
cbor file.json > file.cbor
cbor file.cbor > file.json
`

func main() {
	var inputValue []byte

	if m, _ := os.Stdin.Stat(); m.Mode()&os.ModeCharDevice != os.ModeCharDevice {
		b, err := ioutil.ReadAll(os.Stdin)
		if err == nil {
			inputValue = b
		} else {
			fmt.Fprint(os.Stderr, "ERROR: stdin error.\n\n"+USAGE)
			return
		}
	} else if len(os.Args) == 2 {
		b, err := ioutil.ReadFile(os.Args[1])
		if err != nil {
			fmt.Fprint(os.Stderr, "ERROR: failed to read file "+os.Args[2]+".\n\n"+USAGE)
		}
		inputValue = b
	} else {
		fmt.Fprint(os.Stderr, USAGE)
		return
	}

	var value interface{}
	err := cbor.Unmarshal(inputValue, &value)
	if err == nil {
		// is CBOR, encode to JSON
		err = json.NewEncoder(os.Stdout).Encode(turnKeysIntoStrings(value))
		if err != nil {
			log.Print(err)
		}
	} else {
		// is JSON, encode to CBOR
		decoder := json.NewDecoder(bytes.NewReader(inputValue))
		decoder.UseNumber()
		err := decoder.Decode(&value)
		if err == nil {
			// Convert json.Number values to appropriate types
			value = convertNumbers(value)
			err = cbor.NewEncoder(os.Stdout).Encode(value)
			if err != nil {
				log.Print(err)
			}
		} else {
			fmt.Fprint(os.Stderr, "ERROR: invalid CBOR or JSON value.\n\n"+USAGE)
			return
		}
	}
}

func convertNumbers(anything interface{}) interface{} {
	switch v := anything.(type) {
	case json.Number:
		// Try to convert to int64 first
		if i, err := strconv.ParseInt(string(v), 10, 64); err == nil {
			return i
		}
		// If that fails, convert to float64
		if f, err := strconv.ParseFloat(string(v), 64); err == nil {
			return f
		}
		// If both fail, return the original string
		return string(v)
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for k, val := range v {
			out[k] = convertNumbers(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, val := range v {
			out[i] = convertNumbers(val)
		}
		return out
	default:
		return anything
	}
}

func turnKeysIntoStrings(anything interface{}) interface{} {
	switch m := anything.(type) {
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(m))
		for k, v := range m {
			if ks, ok := k.(string); ok {
				out[ks] = turnKeysIntoStrings(v)
			} else {
				kj, _ := json.Marshal(k)
				out[string(kj)] = turnKeysIntoStrings(v)
			}
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(m))
		for i, v := range m {
			out[i] = turnKeysIntoStrings(v)
		}
		return out
	default:
		return anything
	}
}
