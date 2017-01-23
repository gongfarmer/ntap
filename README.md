# ntap

encoding/atom:
decode.go  // Decoder interface
encode.go  // Encoder interface
codec.co   // ADE type <=> go type conversions


## Notes on layout

# encoding/:
Both encoding/json and encoding/gob have encode.go and decode.go implementing Encoder and Decoder interfaces


# What to take from encoding/json:
has scanner.go for reading text

# What to take from encoding/gob:
This is the only stdlib format that implements both BinaryEncoder and TextEncoder interfaces
Has types.rb for containing type information

# Other goals:
* avoid exposing anything from reflect/
* 
