; Rust Tree-sitter Query
; Extracts functions, methods, structs, enums, traits, and impls for semantic chunking

; Function definitions
(function_item
  name: (identifier) @function.name
  parameters: (parameters) @function.parameters
  body: (block) @function.body) @function.definition

; Struct definitions
(struct_item
  name: (type_identifier) @struct.name
  body: (_)? @struct.body) @struct.definition

; Enum definitions
(enum_item
  name: (type_identifier) @enum.name
  body: (enum_variant_list) @enum.body) @enum.definition

; Trait definitions
(trait_item
  name: (type_identifier) @trait.name
  body: (declaration_list) @trait.body) @trait.definition

; Implementation blocks
(impl_item
  type: (type_identifier) @impl.type
  body: (declaration_list) @impl.body) @impl.definition

; Trait implementation blocks
(impl_item
  trait: (type_identifier) @trait_impl.trait
  type: (type_identifier) @trait_impl.type
  body: (declaration_list) @trait_impl.body) @trait_impl.definition

; Associated functions (methods) within impl blocks
(impl_item
  body: (declaration_list
    (function_item
      name: (identifier) @method.name
      parameters: (parameters) @method.parameters
      body: (block) @method.body) @method.definition))

; Type alias definitions
(type_item
  name: (type_identifier) @type_alias.name
  type: (_) @type_alias.type) @type_alias.definition

; Module definitions
(mod_item
  name: (identifier) @module.name
  body: (declaration_list)? @module.body) @module.definition

; Const definitions
(const_item
  name: (identifier) @const.name
  type: (_) @const.type
  value: (_) @const.value) @const.definition

; Static definitions
(static_item
  name: (identifier) @static.name
  type: (_) @static.type
  value: (_) @static.value) @static.definition
