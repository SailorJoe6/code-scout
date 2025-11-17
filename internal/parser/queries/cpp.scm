; C++ Tree-sitter Query
; Extracts functions, classes, methods, namespaces, templates, and more for semantic chunking

; Function definitions
(function_definition
  declarator: (function_declarator
    declarator: (_) @function.name
    parameters: (parameter_list) @function.parameters)
  body: (compound_statement) @function.body) @function.definition

; Class definitions
(class_specifier
  name: (type_identifier) @class.name
  body: (field_declaration_list) @class.body) @class.definition

; Struct definitions (treated as classes in C++)
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

; Namespace definitions
(namespace_definition
  name: (identifier) @namespace.name
  body: (declaration_list) @namespace.body) @namespace.definition

; Template declarations (function templates)
(template_declaration
  (function_definition
    declarator: (function_declarator
      declarator: (_) @template_function.name
      parameters: (parameter_list) @template_function.parameters)
    body: (compound_statement) @template_function.body)) @template_function.definition

; Template declarations (class templates)
(template_declaration
  (class_specifier
    name: (type_identifier) @template_class.name
    body: (field_declaration_list) @template_class.body)) @template_class.definition

; Method definitions (functions inside classes)
(class_specifier
  body: (field_declaration_list
    (function_definition
      declarator: (function_declarator
        declarator: (_) @method.name
        parameters: (parameter_list) @method.parameters)
      body: (compound_statement) @method.body) @method.definition))

; Constructor definitions
(function_definition
  declarator: (function_declarator
    declarator: (qualified_identifier
      name: (identifier) @constructor.name)
    parameters: (parameter_list) @constructor.parameters)
  body: (compound_statement) @constructor.body) @constructor.definition

; Destructor definitions
(function_definition
  declarator: (function_declarator
    declarator: (destructor_name) @destructor.name
    parameters: (parameter_list) @destructor.parameters)
  body: (compound_statement) @destructor.body) @destructor.definition

; Operator overload definitions
(function_definition
  declarator: (function_declarator
    declarator: (operator_name) @operator.name
    parameters: (parameter_list) @operator.parameters)
  body: (compound_statement) @operator.body) @operator.definition

; Type alias (using declarations)
(alias_declaration
  name: (type_identifier) @type_alias.name
  type: (_) @type_alias.type) @type_alias.definition

; Typedef declarations
(type_definition
  declarator: (type_identifier) @typedef.name) @typedef.definition
