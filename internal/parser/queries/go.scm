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

; Type declarations - other type aliases (exclude structs and interfaces)
; These patterns match type aliases but NOT struct_type or interface_type
(type_declaration
  (type_spec
    name: (type_identifier) @type_alias.name
    type: (type_identifier) @type_alias.type)) @type_alias.definition

(type_declaration
  (type_spec
    name: (type_identifier) @type_alias.name
    type: (qualified_type) @type_alias.type)) @type_alias.definition

(type_declaration
  (type_spec
    name: (type_identifier) @type_alias.name
    type: (pointer_type) @type_alias.type)) @type_alias.definition

(type_declaration
  (type_spec
    name: (type_identifier) @type_alias.name
    type: (slice_type) @type_alias.type)) @type_alias.definition

(type_declaration
  (type_spec
    name: (type_identifier) @type_alias.name
    type: (array_type) @type_alias.type)) @type_alias.definition

(type_declaration
  (type_spec
    name: (type_identifier) @type_alias.name
    type: (map_type) @type_alias.type)) @type_alias.definition

(type_declaration
  (type_spec
    name: (type_identifier) @type_alias.name
    type: (channel_type) @type_alias.type)) @type_alias.definition

(type_declaration
  (type_spec
    name: (type_identifier) @type_alias.name
    type: (function_type) @type_alias.type)) @type_alias.definition

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
