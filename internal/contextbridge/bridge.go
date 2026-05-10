package contextbridge

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidWorld       = errors.New("invalid isolated world")
	ErrInvalidAPIKey      = errors.New("invalid contextBridge API key")
	ErrInvalidValue       = errors.New("invalid contextBridge value")
	ErrAlreadyExposed     = errors.New("contextBridge API already exposed")
	ErrPreloadOrder       = errors.New("preload has already completed")
	ErrMutationNotAllowed = errors.New("exposed API is immutable")
)

type ValueKind string

const (
	ValueString   ValueKind = "string"
	ValueNumber   ValueKind = "number"
	ValueBool     ValueKind = "bool"
	ValueObject   ValueKind = "object"
	ValueArray    ValueKind = "array"
	ValueFunction ValueKind = "function"
	ValueNull     ValueKind = "null"
)

type Value struct {
	Kind     ValueKind
	String   string
	Number   float64
	Bool     bool
	Object   map[string]Value
	Array    []Value
	Function string
}

type Exposure struct {
	WorldID int
	Key     string
	Value   Value
}

type Bridge struct {
	preloadComplete bool
	exposures       map[int]map[string]Value
}

func New() *Bridge {
	return &Bridge{exposures: make(map[int]map[string]Value)}
}

func (b *Bridge) ExposeInMainWorld(key string, value Value) error {
	return b.ExposeInIsolatedWorld(0, key, value)
}

func (b *Bridge) ExposeInIsolatedWorld(worldID int, key string, value Value) error {
	if b.preloadComplete {
		return ErrPreloadOrder
	}
	if worldID < 0 {
		return fmt.Errorf("%w: %d", ErrInvalidWorld, worldID)
	}
	key = strings.TrimSpace(key)
	if key == "" || strings.ContainsAny(key, ".\x00\r\n") {
		return fmt.Errorf("%w: %q", ErrInvalidAPIKey, key)
	}
	cloned, err := cloneValue(value)
	if err != nil {
		return err
	}
	if b.exposures[worldID] == nil {
		b.exposures[worldID] = make(map[string]Value)
	}
	if _, exists := b.exposures[worldID][key]; exists {
		return fmt.Errorf("%w: %s", ErrAlreadyExposed, key)
	}
	b.exposures[worldID][key] = cloned
	return nil
}

func (b *Bridge) CompletePreload() {
	b.preloadComplete = true
}

func (b *Bridge) Exposure(worldID int, key string) (Value, bool) {
	values := b.exposures[worldID]
	if values == nil {
		return Value{}, false
	}
	value, ok := values[key]
	if !ok {
		return Value{}, false
	}
	cloned, err := cloneValue(value)
	if err != nil {
		return Value{}, false
	}
	return cloned, true
}

func (b *Bridge) MutateExposure(worldID int, key string, value Value) error {
	if _, ok := b.Exposure(worldID, key); !ok {
		return fmt.Errorf("%w: %s", ErrInvalidAPIKey, key)
	}
	return ErrMutationNotAllowed
}

func cloneValue(value Value) (Value, error) {
	switch value.Kind {
	case ValueString, ValueNumber, ValueBool, ValueNull:
		return value, nil
	case ValueFunction:
		value.Function = strings.TrimSpace(value.Function)
		if value.Function == "" {
			return Value{}, fmt.Errorf("%w: function name is required", ErrInvalidValue)
		}
		return value, nil
	case ValueObject:
		object := make(map[string]Value, len(value.Object))
		for key, child := range value.Object {
			key = strings.TrimSpace(key)
			if key == "" || strings.ContainsAny(key, "\x00\r\n") {
				return Value{}, fmt.Errorf("%w: object key", ErrInvalidValue)
			}
			cloned, err := cloneValue(child)
			if err != nil {
				return Value{}, err
			}
			object[key] = cloned
		}
		value.Object = object
		return value, nil
	case ValueArray:
		array := make([]Value, len(value.Array))
		for i, child := range value.Array {
			cloned, err := cloneValue(child)
			if err != nil {
				return Value{}, err
			}
			array[i] = cloned
		}
		value.Array = array
		return value, nil
	default:
		return Value{}, fmt.Errorf("%w: kind %q", ErrInvalidValue, value.Kind)
	}
}
