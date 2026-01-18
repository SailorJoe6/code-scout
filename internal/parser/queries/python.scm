; Python Tree-sitter Query
; Extracts functions, classes, and methods for semantic chunking

; Function definitions (includes async functions)
(function_definition
  name: (identifier) @function_name
  parameters: (parameters) @function_parameters) @function_definition

; Class definitions
(class_definition
  name: (identifier) @class_name) @class_definition
