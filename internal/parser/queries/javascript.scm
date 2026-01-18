; JavaScript/TypeScript Tree-sitter Query
; Extracts functions, classes, methods, and arrow functions for semantic chunking

; Function declarations
(function_declaration
  name: (identifier) @function.name
  parameters: (formal_parameters) @function.parameters
  body: (statement_block) @function.body) @function.definition

; Function expressions
(function_expression
  name: (identifier)? @function_expr.name
  parameters: (formal_parameters) @function_expr.parameters
  body: (statement_block) @function_expr.body) @function_expr.definition

; Arrow functions
(arrow_function
  parameters: (_) @arrow.parameters
  body: (_) @arrow.body) @arrow.definition

; Method definitions in classes
(method_definition
  name: (property_identifier) @method.name
  parameters: (formal_parameters) @method.parameters
  body: (statement_block) @method.body) @method.definition

; Class declarations
(class_declaration
  name: (identifier) @class.name
  body: (class_body) @class.body) @class.definition

; Class expressions
(class
  name: (identifier)? @class_expr.name
  body: (class_body) @class_expr.body) @class_expr.definition

; Generator functions
(generator_function_declaration
  name: (identifier) @generator.name
  parameters: (formal_parameters) @generator.parameters
  body: (statement_block) @generator.body) @generator.definition

; Note: Async functions are captured by the regular function_declaration pattern above
; The 'async' keyword is part of the function definition content

; Note: TypeScript-specific patterns (interfaces, type aliases, enums) are in typescript.scm
