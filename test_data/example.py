def hello_world():
    """A simple hello world function."""
    print("Hello, World!")

class Calculator:
    """A simple calculator class."""
    
    def add(self, a, b):
        return a + b
    
    def subtract(self, a, b):
        return a - b

if __name__ == "__main__":
    hello_world()
    calc = Calculator()
    print(calc.add(5, 3))
