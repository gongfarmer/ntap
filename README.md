## Encoding library for ADE Atom data format

Go implementation of a proprietary data format.
Provides conversion tools useful for debugging.

### Tools
- **ccat**: converts binary format to text
- **ctac**: converts text format to binary

### Encoding library
- **atom.go**
- **text.go**
  * conversion of Atom to ADE Container Text format
  * implements TextMarshaler, TextUnmarshaler interfaces
- **binary.go**
  * conversion of Atom to binary format
  * implements BinaryMarshaler, BinaryUnmarshaler interfaces
- **xml.go**
  * conversion of Atom to XML format (work in progress)
- **path.go**
  * xpath for Atom
  * returns a set of atoms or atom data based on a path expression
  * strictly follows XPath documentation
- **codec/codec.go**
  * implements type system for all ADE data types
  * handles conversion of data between ADE type and equivalent Go type
