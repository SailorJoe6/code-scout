; PHP Tree-sitter Query
; Extracts functions, classes, methods, interfaces, traits, and namespaces for semantic chunking

; Function definitions
(function_definition
  name: (name) @function.name
  parameters: (formal_parameters) @function.parameters
  body: (compound_statement) @function.body) @function.definition

; Class declarations
(class_declaration
  name: (name) @class.name
  body: (declaration_list) @class.body) @class.definition

; Interface declarations
(interface_declaration
  name: (name) @interface.name
  body: (declaration_list) @interface.body) @interface.definition

; Trait declarations
(trait_declaration
  name: (name) @trait.name
  body: (declaration_list) @trait.body) @trait.definition

; Method declarations
(method_declaration
  name: (name) @method.name
  parameters: (formal_parameters) @method.parameters
  body: (compound_statement) @method.body) @method.definition

; Abstract method declarations
(method_declaration
  (abstract_modifier)
  name: (name) @abstract_method.name
  parameters: (formal_parameters) @abstract_method.parameters) @abstract_method.definition

; Namespace declarations
(namespace_definition
  name: (namespace_name) @namespace.name
  body: (_)? @namespace.body) @namespace.definition

; Anonymous functions (closures)
(anonymous_function_creation_expression
  parameters: (formal_parameters) @anonymous_function.parameters
  body: (compound_statement) @anonymous_function.body) @anonymous_function.definition

; Arrow functions (PHP 7.4+)
(arrow_function
  parameters: (formal_parameters) @arrow_function.parameters
  body: (_) @arrow_function.body) @arrow_function.definition

; Enum declarations (PHP 8.1+)
(enum_declaration
  name: (name) @enum.name
  body: (enum_declaration_list) @enum.body) @enum.definition

; Constructor method
(method_declaration
  name: (name) @constructor.name
  (#match? @constructor.name "^__construct$")
  parameters: (formal_parameters) @constructor.parameters
  body: (compound_statement) @constructor.body) @constructor.definition

; Destructor method
(method_declaration
  name: (name) @destructor.name
  (#match? @destructor.name "^__destruct$")
  parameters: (formal_parameters) @destructor.parameters
  body: (compound_statement) @destructor.body) @destructor.definition

; Magic methods
(method_declaration
  name: (name) @magic_method.name
  (#match? @magic_method.name "^__[a-zA-Z]+$")
  parameters: (formal_parameters) @magic_method.parameters
  body: (compound_statement) @magic_method.body) @magic_method.definition
