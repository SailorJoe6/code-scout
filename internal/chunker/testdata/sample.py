"""
Sample Python file for testing semantic chunking.
This file contains various Python constructs to test extraction.
"""

import os
import sys
from typing import List, Dict, Optional


# Module-level constant
MAX_RETRIES = 3


def simple_function(name: str) -> str:
    """A simple function with a docstring."""
    return f"Hello, {name}!"


def function_with_params(x: int, y: int, z: int = 10) -> int:
    """Function with multiple parameters and default value."""
    return x + y + z


async def async_function(url: str) -> Dict:
    """An async function for testing."""
    import asyncio
    await asyncio.sleep(1)
    return {"url": url, "status": "ok"}


class BaseClass:
    """A base class for testing inheritance."""

    def __init__(self, name: str):
        """Initialize the base class."""
        self.name = name

    def get_name(self) -> str:
        """Get the name."""
        return self.name


class DerivedClass(BaseClass):
    """A derived class that extends BaseClass."""

    def __init__(self, name: str, value: int):
        """Initialize with name and value."""
        super().__init__(name)
        self.value = value

    def get_value(self) -> int:
        """Get the value."""
        return self.value

    @property
    def display_name(self) -> str:
        """Property decorator example."""
        return f"{self.name}: {self.value}"

    @staticmethod
    def static_method() -> str:
        """Static method example."""
        return "I'm static"

    @classmethod
    def class_method(cls) -> str:
        """Class method example."""
        return f"I'm a class method of {cls.__name__}"


@dataclass
class DataClassExample:
    """Example using dataclass decorator."""
    name: str
    age: int
    email: Optional[str] = None


def generator_function(n: int):
    """A generator function."""
    for i in range(n):
        yield i * 2


lambda_example = lambda x: x ** 2


# Nested function
def outer_function(x: int):
    """Function with nested inner function."""

    def inner_function(y: int):
        """Inner function."""
        return x + y

    return inner_function(10)


if __name__ == "__main__":
    print(simple_function("World"))
    print(function_with_params(1, 2))

    obj = DerivedClass("Test", 42)
    print(obj.display_name)
