; C Tree-sitter Query
; Extracts functions, structs, unions, enums, and typedefs for semantic chunking

; Function definitions
(function_definition
  declarator: (function_declarator
    declarator: (identifier) @function.name
    parameters: (parameter_list) @function.parameters)
  body: (compound_statement) @function.body) @function.definition

; Function declarations (prototypes)
(declaration
  declarator: (function_declarator
    declarator: (identifier) @function_decl.name
    parameters: (parameter_list) @function_decl.parameters)) @function_decl.definition

; Struct definitions
(struct_specifier
  name: (type_identifier) @struct.name
  body: (field_declaration_list) @struct.body) @struct.definition

; Union definitions
(union_specifier
  name: (type_identifier) @union.name
  body: (field_declaration_list) @union.body) @union.definition

; Enum definitions
(enum_specifier
  name: (type_identifier) @enum.name
  body: (enumerator_list) @enum.body) @enum.definition

; Typedef declarations
(type_definition
  declarator: (type_identifier) @typedef.name) @typedef.definition

; Typedef'd structs
(type_definition
  type: (struct_specifier
    name: (type_identifier)? @typedef_struct.struct_name
    body: (field_declaration_list) @typedef_struct.body)
  declarator: (type_identifier) @typedef_struct.name) @typedef_struct.definition

; Typedef'd unions
(type_definition
  type: (union_specifier
    name: (type_identifier)? @typedef_union.union_name
    body: (field_declaration_list) @typedef_union.body)
  declarator: (type_identifier) @typedef_union.name) @typedef_union.definition

; Typedef'd enums
(type_definition
  type: (enum_specifier
    name: (type_identifier)? @typedef_enum.enum_name
    body: (enumerator_list) @typedef_enum.body)
  declarator: (type_identifier) @typedef_enum.name) @typedef_enum.definition
