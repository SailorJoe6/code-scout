; Scala Tree-sitter Query
; Extracts functions, classes, objects, traits, and case classes for semantic chunking

; Function definitions (def)
(function_definition
  name: (identifier) @function.name
  parameters: (parameters) @function.parameters
  body: (_)? @function.body) @function.definition

; Class definitions
(class_definition
  name: (identifier) @class.name
  body: (template_body) @class.body) @class.definition

; Object definitions
(object_definition
  name: (identifier) @object.name
  body: (template_body) @object.body) @object.definition

; Trait definitions
(trait_definition
  name: (identifier) @trait.name
  body: (template_body) @trait.body) @trait.definition

; Case class definitions
(class_definition
  (case_modifier)
  name: (identifier) @case_class.name
  body: (template_body)? @case_class.body) @case_class.definition

; Case object definitions
(object_definition
  (case_modifier)
  name: (identifier) @case_object.name
  body: (template_body)? @case_object.body) @case_object.definition

; Type alias definitions
(type_definition
  name: (type_identifier) @type_alias.name
  type: (_) @type_alias.type) @type_alias.definition

; Value definitions (val)
(val_definition
  pattern: (identifier) @val.name
  value: (_)? @val.value) @val.definition

; Variable definitions (var)
(var_definition
  pattern: (identifier) @var.name
  value: (_)? @var.value) @var.definition

; Lambda expressions (anonymous functions)
(lambda_expression
  parameters: (_)? @lambda.parameters
  body: (_) @lambda.body) @lambda.definition

; Pattern matching case clauses (significant for understanding control flow)
(case_clause
  pattern: (_) @case.pattern
  body: (_) @case.body) @case.definition

; Package definitions
(package_clause
  name: (_) @package.name) @package.definition

; Extension methods (Scala 3)
(extension_definition
  type_parameters: (_)? @extension.type_params
  parameters: (parameters) @extension.parameters
  body: (template_body) @extension.body) @extension.definition

; Given instances (Scala 3)
(given_definition
  name: (identifier)? @given.name
  type: (_) @given.type
  body: (_)? @given.body) @given.definition
