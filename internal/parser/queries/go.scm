; Go Tree-sitter Query
; Extracts functions, methods, structs, interfaces, and types for semantic chunking
; Note: The current Go implementation uses hardcoded node traversal in extractor.go
; This query file is provided for future query-based implementation

; Function declarations
(function_declaration
  name: (identifier) @function.name
  parameters: (parameter_list) @function.parameters
  result: (_)? @function.result
  body: (block) @function.body) @function.definition

; Method declarations
(method_declaration
  receiver: (parameter_list) @method.receiver
  name: (field_identifier) @method.name
  parameters: (parameter_list) @method.parameters
  result: (_)? @method.result
  body: (block) @method.body) @method.definition

; Type declarations - struct
(type_declaration
  (type_spec
    name: (type_identifier) @struct.name
    type: (struct_type
      (field_declaration_list) @struct.fields))) @struct.definition

; Type declarations - interface
(type_declaration
  (type_spec
    name: (type_identifier) @interface.name
    type: (interface_type) @interface.body)) @interface.definition

; Type declarations - other type aliases
(type_declaration
  (type_spec
    name: (type_identifier) @type_alias.name
    type: (_) @type_alias.type)) @type_alias.definition

; Package clause
(package_clause
  (package_identifier) @package.name) @package.definition

; Import declarations
(import_declaration
  (import_spec
    path: (interpreted_string_literal) @import.path)) @import.definition

; Import declaration lists
(import_declaration
  (import_spec_list
    (import_spec
      path: (interpreted_string_literal) @import.path))) @import.definition

; Const declarations
(const_declaration
  (const_spec
    name: (identifier) @const.name
    value: (_)? @const.value)) @const.definition

; Var declarations
(var_declaration
  (var_spec
    name: (identifier) @var.name
    value: (_)? @var.value)) @var.definition
