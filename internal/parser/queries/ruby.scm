; Ruby Tree-sitter Query
; Extracts methods, classes, modules, and singleton methods for semantic chunking

; Method definitions
(method
  name: (_) @method.name
  parameters: (method_parameters)? @method.parameters
  (_)* @method.body) @method.definition

; Class definitions
(class
  name: (_) @class.name
  (_)* @class.body) @class.definition

; Module definitions
(module
  name: (_) @module.name
  (_)* @module.body) @module.definition

; Singleton class definitions
(singleton_class
  value: (_) @singleton_class.value
  (_)* @singleton_class.body) @singleton_class.definition

; Singleton method definitions
(singleton_method
  object: (_) @singleton_method.object
  name: (_) @singleton_method.name
  parameters: (method_parameters)? @singleton_method.parameters
  (_)* @singleton_method.body) @singleton_method.definition

; Lambda definitions
(lambda
  parameters: (lambda_parameters)? @lambda.parameters
  (_)* @lambda.body) @lambda.definition

; Block with parameters (procs)
(do_block
  parameters: (block_parameters)? @block.parameters
  (_)* @block.body) @block.definition

; Class method definitions (def self.method_name)
(singleton_method
  object: (self) @class_method.self
  name: (_) @class_method.name
  parameters: (method_parameters)? @class_method.parameters
  (_)* @class_method.body) @class_method.definition

; Alias definitions
(alias
  name: (_) @alias.new_name
  alias: (_) @alias.old_name) @alias.definition

; Attr accessors, readers, writers
(call
  method: (identifier) @attr.type
  arguments: (argument_list
    (simple_symbol) @attr.name)
  (#match? @attr.type "^attr_(accessor|reader|writer)$")) @attr.definition
