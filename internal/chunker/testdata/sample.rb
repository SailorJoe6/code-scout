# Sample Ruby file for testing semantic chunking.
# This file contains various Ruby constructs to test extraction.

# Module-level constant
MAX_RETRIES = 3

# Simple method
def greet(name)
  "Hello, #{name}!"
end

# Method with default parameters
def add(x, y, z = 10)
  x + y + z
end

# Method with keyword arguments
def configure(url:, timeout: 5000, retries: nil)
  { api_url: url, timeout: timeout, retries: retries }
end

# Method with splat operator
def sum_all(*numbers)
  numbers.sum
end

# Module definition
module Comparable
  # Module method
  def compare(a, b)
    a <=> b
  end

  # Module constant
  VERSION = '1.0.0'
end

# Class definition
class Animal
  attr_reader :name

  # Constructor
  def initialize(name)
    @name = name
  end

  # Instance method
  def speak
    raise NotImplementedError, 'Subclass must implement abstract method'
  end

  # Instance method with parameters
  def move(distance)
    puts "#{@name} moved #{distance}m."
  end

  # Class method
  def self.create(name)
    new(name)
  end

  # Private methods
  private

  def internal_calculation
    @name.length * 2
  end
end

# Derived class
class Dog < Animal
  attr_accessor :breed

  def initialize(name, breed)
    super(name)
    @breed = breed
  end

  # Override parent method
  def speak
    'Woof!'
  end

  def fetch(item)
    "#{@name} fetched the #{item}!"
  end

  # Class variable
  @@count = 0

  # Class method accessing class variable
  def self.count
    @@count
  end

  def self.increment_count
    @@count += 1
  end
end

# Singleton class methods
class User
  attr_reader :id, :name, :email

  def initialize(id, name, email)
    @id = id
    @name = name
    @email = email
  end

  # Instance method
  def to_s
    "User(id=#{@id}, name=#{@name}, email=#{@email})"
  end

  # Singleton method on instance
  def self.admin(name, email)
    new('admin-' + rand(1000).to_s, name, email)
  end
end

# Mixin module
module Loggable
  def log(message)
    puts "[#{Time.now}] #{message}"
  end

  def log_error(error)
    log("ERROR: #{error}")
  end
end

# Class using mixin
class Service
  include Loggable

  def initialize(name)
    @name = name
  end

  def process(data)
    log("Processing #{data}")
    data.upcase
  end
end

# Module with class methods
module StringUtils
  module_function

  def capitalize(str)
    str.capitalize
  end

  def slugify(str)
    str.downcase.gsub(/\s+/, '-')
  end
end

# Lambda and Proc
multiply = lambda { |x, y| x * y }
square = ->(x) { x ** 2 }

add_proc = Proc.new { |x, y| x + y }

# Block methods
def with_logging
  puts 'Starting...'
  result = yield
  puts 'Finished.'
  result
end

# Method accepting block
def map_values(hash, &block)
  hash.transform_values(&block)
end

# Struct definition
Person = Struct.new(:name, :age, :email) do
  def adult?
    age >= 18
  end

  def to_s
    "#{name} (#{age})"
  end
end

# Singleton methods on objects
obj = Object.new

def obj.custom_method
  'I am a singleton method'
end

# Monkey patching (reopening class)
class String
  def shout
    upcase + '!'
  end
end

# Eigenclass (singleton class)
class Calculator
  class << self
    def add(x, y)
      x + y
    end

    def subtract(x, y)
      x - y
    end
  end
end

# Metaprogramming - define_method
class DynamicMethods
  ['get', 'set', 'delete'].each do |action|
    define_method("#{action}_value") do |key|
      puts "#{action.capitalize}ting #{key}"
    end
  end
end

# Main execution
if __FILE__ == $0
  puts greet('World')
  puts add(1, 2)

  dog = Dog.new('Buddy', 'Golden Retriever')
  puts dog.speak
  puts dog.fetch('ball')

  user = User.new('1', 'John Doe', 'john@example.com')
  puts user.to_s

  service = Service.new('DataProcessor')
  service.process('hello')
end
