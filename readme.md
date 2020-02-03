# protoc-gen-uwgo-entity

Given...

```protobuf
import "github.com/utilitywarehouse/protoc-gen-uwgo-entity/annotations.proto";

message SomeEvent {
    string some_id = 1 [(uw.entity.identifier) = true];
}
```

Will generate a handy helper like...

```go
// GetEntityIdentifier returns the value from the field marked as the identifier
func (m *SomeEvent) GetEntityIdentifier() string {
	if m != nil {
		return m.SomeId
	}
	return ""
}
```

This comes in handy, for example, if you want to get a Kafka partition key in a generic way.

## options

Pass these options along with `--uwgo-entity_out`

`enforce=true` - All messages must have the identifier set
`enforce-suffix=Event` - Only messages with this suffix will be enforced

Set these options within the proto message

`uw.entity.ignore` - Allows you when enforcing, to still skip a message

```protobuf
message SomeEvent {
    option (uw.entity.ignore) = true;

    string some_id = 1;
}
```
