package cache

// KeyValue ...
type KeyValue struct {
	Key string
	FieldValue
}

// KeyValues ...
type KeyValues []KeyValue

// FieldValue ...
type FieldValue struct {
	Field string
	Value string
}

// FieldValues ...
type FieldValues []FieldValue

// KeyMapValues ...
type KeyMapValues []KeyMapValue

// KeyMapValue ...
type KeyMapValue struct {
	Key    string
	Values map[string]interface{}
}
