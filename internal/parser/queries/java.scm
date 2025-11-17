; Java Tree-sitter Query
; Extracts classes, methods, constructors, and interfaces for semantic chunking

; Class declarations
(class_declaration
  name: (identifier) @class.name
  body: (class_body) @class.body) @class.definition

; Interface declarations
(interface_declaration
  name: (identifier) @interface.name
  body: (interface_body) @interface.body) @interface.definition

; Enum declarations
(enum_declaration
  name: (identifier) @enum.name
  body: (enum_body) @enum.body) @enum.definition

; Method declarations
(method_declaration
  name: (identifier) @method.name
  parameters: (formal_parameters) @method.parameters
  body: (block) @method.body) @method.definition

; Constructor declarations
(constructor_declaration
  name: (identifier) @constructor.name
  parameters: (formal_parameters) @constructor.parameters
  body: (constructor_body) @constructor.body) @constructor.definition

; Abstract method declarations (interface methods)
(method_declaration
  name: (identifier) @abstract_method.name
  parameters: (formal_parameters) @abstract_method.parameters) @abstract_method.definition

; Annotation type declarations
(annotation_type_declaration
  name: (identifier) @annotation.name
  body: (annotation_type_body) @annotation.body) @annotation.definition

; Record declarations (Java 14+)
(record_declaration
  name: (identifier) @record.name
  parameters: (formal_parameters) @record.parameters
  body: (class_body) @record.body) @record.definition
