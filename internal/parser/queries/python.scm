; Python Tree-sitter Query
; Extracts functions, classes, and methods for semantic chunking

; Function definitions
(function_definition
  name: (identifier) @function.name
  parameters: (parameters) @function.parameters
  body: (block) @function.body) @function.definition

; Class definitions
(class_definition
  name: (identifier) @class.name
  body: (block) @class.body) @class.definition

; Method definitions (functions inside classes)
(class_definition
  body: (block
    (function_definition
      name: (identifier) @method.name
      parameters: (parameters) @method.parameters
      body: (block) @method.body) @method.definition))

; Async function definitions
(function_definition
  (async) @function.async
  name: (identifier) @async_function.name
  parameters: (parameters) @async_function.parameters
  body: (block) @async_function.body) @async_function.definition

; Decorated functions (with @decorator)
(decorated_definition
  (decorator) @decorator
  definition: (function_definition
    name: (identifier) @decorated_function.name
    parameters: (parameters) @decorated_function.parameters
    body: (block) @decorated_function.body) @decorated_function.definition)

; Decorated classes
(decorated_definition
  (decorator) @decorator
  definition: (class_definition
    name: (identifier) @decorated_class.name
    body: (block) @decorated_class.body) @decorated_class.definition)
