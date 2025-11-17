/**
 * Sample Scala file for testing semantic chunking.
 * This file contains various Scala constructs to test extraction.
 */

package com.example.chunker.testdata

import scala.collection.mutable
import scala.concurrent.Future
import scala.util.{Try, Success, Failure}

// Package object
package object utils {
  def capitalize(str: String): String = {
    if (str.isEmpty) str
    else str.head.toUpper + str.tail
  }

  def slugify(str: String): String = {
    str.toLowerCase.replaceAll("\\s+", "-")
  }
}

/**
 * Simple function
 */
def greet(name: String): String = {
  s"Hello, $name!"
}

/**
 * Function with default parameters
 */
def add(x: Int, y: Int, z: Int = 10): Int = {
  x + y + z
}

/**
 * Generic function
 */
def identity[T](value: T): T = value

/**
 * Higher-order function
 */
def applyTwice[A](f: A => A, x: A): A = {
  f(f(x))
}

/**
 * Case class (immutable data structure)
 */
case class Point(x: Double, y: Double) {
  def distanceFromOrigin: Double = {
    math.sqrt(x * x + y * y)
  }

  def +(other: Point): Point = {
    Point(x + other.x, y + other.y)
  }
}

/**
 * Companion object for Point
 */
object Point {
  def origin: Point = Point(0.0, 0.0)

  def apply(x: Double): Point = Point(x, 0.0)
}

/**
 * Abstract class
 */
abstract class Animal(val name: String) {
  def makeSound(): String

  def move(distance: Int): Unit = {
    println(s"$name moved ${distance}m.")
  }
}

/**
 * Concrete class extending abstract class
 */
class Dog(name: String, val breed: String) extends Animal(name) {
  override def makeSound(): String = "Woof!"

  def fetch(item: String): String = {
    s"$name fetched the $item!"
  }
}

/**
 * Companion object with factory method
 */
object Dog {
  def apply(name: String, breed: String): Dog = {
    new Dog(name, breed)
  }

  def createPet(name: String): Dog = {
    new Dog(name, "Mixed")
  }
}

/**
 * Trait definition (like interface)
 */
trait Repository[T] {
  def find(id: String): Option[T]
  def save(id: String, entity: T): Unit
  def delete(id: String): Boolean
  def findAll(): List[T]
}

/**
 * Case class for User
 */
case class User(id: String, name: String, email: String) {
  def displayName: String = s"$name <$email>"
}

/**
 * Implementation of Repository trait
 */
class UserRepository extends Repository[User] {
  private val users = mutable.HashMap[String, User]()

  override def find(id: String): Option[User] = {
    users.get(id)
  }

  override def save(id: String, user: User): Unit = {
    users(id) = user
  }

  override def delete(id: String): Boolean = {
    users.remove(id).isDefined
  }

  override def findAll(): List[User] = {
    users.values.toList
  }

  def findByName(name: String): List[User] = {
    users.values.filter(_.name == name).toList
  }
}

/**
 * Generic container class
 */
class Container[T](private var value: T) {
  def get: T = value

  def set(newValue: T): Unit = {
    value = newValue
  }

  def map[U](f: T => U): Container[U] = {
    new Container(f(value))
  }
}

/**
 * Sealed trait for algebraic data types
 */
sealed trait Status
case object Pending extends Status
case object Active extends Status
case object Completed extends Status
case class Failed(reason: String) extends Status

/**
 * Pattern matching helper
 */
object Status {
  def isTerminal(status: Status): Boolean = status match {
    case Completed | Failed(_) => true
    case _ => false
  }

  def message(status: Status): String = status match {
    case Pending => "Pending"
    case Active => "Active"
    case Completed => "Completed"
    case Failed(reason) => s"Failed: $reason"
  }
}

/**
 * Trait with self-type
 */
trait Loggable {
  def log(message: String): Unit = {
    println(s"[${java.time.Instant.now}] $message")
  }

  def logError(error: String): Unit = {
    log(s"ERROR: $error")
  }
}

/**
 * Service class with trait mixin
 */
class UserService(repository: UserRepository) extends Loggable {
  def createUser(name: String, email: String): User = {
    val id = java.util.UUID.randomUUID().toString
    val user = User(id, name, email)
    repository.save(id, user)
    log(s"Created user: $id")
    user
  }

  def getUser(id: String): Option[User] = {
    repository.find(id)
  }

  def deleteUser(id: String): Boolean = {
    val deleted = repository.delete(id)
    if (deleted) log(s"Deleted user: $id")
    deleted
  }

  def getAllUsers: List[User] = {
    repository.findAll()
  }
}

/**
 * Implicit class for extension methods
 */
implicit class StringOps(s: String) {
  def shout: String = s.toUpperCase + "!"

  def isPalindrome: Boolean = {
    val normalized = s.toLowerCase.replaceAll("\\s", "")
    normalized == normalized.reverse
  }
}

/**
 * Enumeration
 */
object Color extends Enumeration {
  type Color = Value
  val Red, Green, Blue, Yellow = Value
}

/**
 * For comprehension example
 */
def cartesianProduct[A, B](as: List[A], bs: List[B]): List[(A, B)] = {
  for {
    a <- as
    b <- bs
  } yield (a, b)
}

/**
 * Partial function
 */
val divide: PartialFunction[(Int, Int), Int] = {
  case (x, y) if y != 0 => x / y
}

/**
 * Curried function
 */
def multiplier(factor: Int)(value: Int): Int = {
  factor * value
}

/**
 * By-name parameter
 */
def measure[T](block: => T): (T, Long) = {
  val start = System.nanoTime()
  val result = block
  val elapsed = System.nanoTime() - start
  (result, elapsed)
}

/**
 * Main object (entry point)
 */
object Main extends App {
  println(greet("World"))
  println(add(1, 2))

  val point = Point(3.0, 4.0)
  println(s"Distance from origin: ${point.distanceFromOrigin}")

  val dog = Dog("Buddy", "Golden Retriever")
  println(dog.makeSound())
  println(dog.fetch("ball"))

  val service = new UserService(new UserRepository)
  val user = service.createUser("John Doe", "john@example.com")
  println(user.displayName)
}
