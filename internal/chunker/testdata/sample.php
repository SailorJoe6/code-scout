<?php
/**
 * Sample PHP file for testing semantic chunking.
 * This file contains various PHP constructs to test extraction.
 */

namespace App\Sample;

use DateTime;
use Exception;

// Constants
const MAX_RETRIES = 3;
define('APP_VERSION', '1.0.0');

/**
 * Simple function with documentation
 * @param string $name The name to greet
 * @return string The greeting message
 */
function greet(string $name): string {
    return "Hello, {$name}!";
}

/**
 * Function with default parameters
 */
function add(int $x, int $y, int $z = 10): int {
    return $x + $y + $z;
}

/**
 * Function with reference parameter
 */
function increment(int &$value): void {
    $value++;
}

/**
 * Variadic function
 */
function sum_all(int ...$numbers): int {
    return array_sum($numbers);
}

/**
 * Anonymous function (closure)
 */
$multiply = function(int $x, int $y): int {
    return $x * $y;
};

/**
 * Arrow function (PHP 7.4+)
 */
$square = fn($x) => $x ** 2;

/**
 * Base class
 */
abstract class Animal {
    protected string $name;

    public function __construct(string $name) {
        $this->name = $name;
    }

    abstract public function makeSound(): string;

    public function move(int $distance): void {
        echo "{$this->name} moved {$distance}m.\n";
    }

    public function getName(): string {
        return $this->name;
    }

    public static function create(string $name): static {
        return new static($name);
    }
}

/**
 * Derived class
 */
class Dog extends Animal {
    private string $breed;

    public function __construct(string $name, string $breed) {
        parent::__construct($name);
        $this->breed = $breed;
    }

    public function makeSound(): string {
        return 'Woof!';
    }

    public function getBreed(): string {
        return $this->breed;
    }

    public function setBreed(string $breed): void {
        $this->breed = $breed;
    }

    public function __toString(): string {
        return "{$this->name} ({$this->breed})";
    }
}

/**
 * Interface definition
 */
interface Repository {
    public function find(string $id): ?object;
    public function save(string $id, object $entity): void;
    public function delete(string $id): bool;
    public function findAll(): array;
}

/**
 * User entity class
 */
class User {
    private string $id;
    private string $name;
    private string $email;

    public function __construct(string $id, string $name, string $email) {
        $this->id = $id;
        $this->name = $name;
        $this->email = $email;
    }

    // Getters
    public function getId(): string {
        return $this->id;
    }

    public function getName(): string {
        return $this->name;
    }

    public function getEmail(): string {
        return $this->email;
    }

    // Setters
    public function setName(string $name): void {
        $this->name = $name;
    }

    public function setEmail(string $email): void {
        $this->email = $email;
    }

    // Magic methods
    public function __sleep(): array {
        return ['id', 'name', 'email'];
    }

    public function __wakeup(): void {
        // Restore logic
    }
}

/**
 * Repository implementation
 */
class UserRepository implements Repository {
    private array $users = [];

    public function find(string $id): ?object {
        return $this->users[$id] ?? null;
    }

    public function save(string $id, object $entity): void {
        if (!$entity instanceof User) {
            throw new Exception('Entity must be a User');
        }
        $this->users[$id] = $entity;
    }

    public function delete(string $id): bool {
        if (isset($this->users[$id])) {
            unset($this->users[$id]);
            return true;
        }
        return false;
    }

    public function findAll(): array {
        return array_values($this->users);
    }

    public function findByName(string $name): array {
        return array_filter($this->users, function($user) use ($name) {
            return $user->getName() === $name;
        });
    }
}

/**
 * Trait definition
 */
trait Timestampable {
    private ?DateTime $createdAt = null;
    private ?DateTime $updatedAt = null;

    public function getCreatedAt(): ?DateTime {
        return $this->createdAt;
    }

    public function getUpdatedAt(): ?DateTime {
        return $this->updatedAt;
    }

    public function touch(): void {
        $now = new DateTime();
        if ($this->createdAt === null) {
            $this->createdAt = $now;
        }
        $this->updatedAt = $now;
    }
}

/**
 * Class using trait
 */
class Post {
    use Timestampable;

    private string $title;
    private string $content;

    public function __construct(string $title, string $content) {
        $this->title = $title;
        $this->content = $content;
        $this->touch();
    }

    public function getTitle(): string {
        return $this->title;
    }

    public function getContent(): string {
        return $this->content;
    }
}

/**
 * Enum (PHP 8.1+)
 */
enum Status: string {
    case Pending = 'pending';
    case Active = 'active';
    case Completed = 'completed';
    case Failed = 'failed';

    public function label(): string {
        return match($this) {
            self::Pending => 'Pending',
            self::Active => 'Active',
            self::Completed => 'Completed',
            self::Failed => 'Failed',
        };
    }

    public function isTerminal(): bool {
        return in_array($this, [self::Completed, self::Failed]);
    }
}

/**
 * Generic service class
 */
class UserService {
    private UserRepository $repository;

    public function __construct(UserRepository $repository) {
        $this->repository = $repository;
    }

    public function createUser(string $name, string $email): User {
        $id = uniqid('user_', true);
        $user = new User($id, $name, $email);
        $this->repository->save($id, $user);
        return $user;
    }

    public function getUser(string $id): ?User {
        $entity = $this->repository->find($id);
        return $entity instanceof User ? $entity : null;
    }

    public function deleteUser(string $id): bool {
        return $this->repository->delete($id);
    }

    public function getAllUsers(): array {
        return $this->repository->findAll();
    }
}

// Main execution
if (basename(__FILE__) === basename($_SERVER['PHP_SELF'])) {
    echo greet('World') . PHP_EOL;
    echo add(1, 2) . PHP_EOL;

    $dog = new Dog('Buddy', 'Golden Retriever');
    echo $dog->makeSound() . PHP_EOL;
    echo $dog . PHP_EOL;

    $service = new UserService(new UserRepository());
    $user = $service->createUser('John Doe', 'john@example.com');
    echo $user->getName() . PHP_EOL;
}
