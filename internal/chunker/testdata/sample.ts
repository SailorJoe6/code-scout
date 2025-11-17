/**
 * Sample TypeScript file for testing semantic chunking.
 * This file contains various TypeScript constructs to test extraction.
 */

// Type aliases
type UserId = string;
type UserRole = 'admin' | 'user' | 'guest';

// Interface definition
interface User {
    id: UserId;
    name: string;
    email: string;
    role: UserRole;
}

// Interface with optional and readonly properties
interface Config {
    readonly apiUrl: string;
    timeout?: number;
    retries?: number;
}

/**
 * Generic interface
 */
interface Repository<T> {
    find(id: string): Promise<T | null>;
    save(entity: T): Promise<void>;
    delete(id: string): Promise<boolean>;
}

/**
 * Enum definition
 */
enum Status {
    Pending = 'PENDING',
    Active = 'ACTIVE',
    Completed = 'COMPLETED',
    Failed = 'FAILED'
}

/**
 * Const enum
 */
const enum LogLevel {
    Debug,
    Info,
    Warning,
    Error
}

/**
 * Simple function with type annotations
 */
function greet(name: string): string {
    return `Hello, ${name}!`;
}

/**
 * Function with optional and default parameters
 */
function configure(url: string, timeout: number = 5000, retries?: number): Config {
    return { apiUrl: url, timeout, retries };
}

/**
 * Generic function
 */
function identity<T>(value: T): T {
    return value;
}

/**
 * Arrow function with type annotation
 */
const add = (x: number, y: number): number => x + y;

/**
 * Async arrow function
 */
const fetchUser = async (id: UserId): Promise<User | null> => {
    const response = await fetch(`/api/users/${id}`);
    return response.json();
};

/**
 * Abstract base class
 */
abstract class Animal {
    constructor(protected name: string) {}

    abstract makeSound(): string;

    move(distance: number = 0): void {
        console.log(`${this.name} moved ${distance}m.`);
    }
}

/**
 * Concrete class extending abstract class
 */
class Dog extends Animal {
    constructor(name: string, private breed: string) {
        super(name);
    }

    makeSound(): string {
        return 'Woof!';
    }

    getBreed(): string {
        return this.breed;
    }
}

/**
 * Class implementing interface
 */
class UserRepository implements Repository<User> {
    private users: Map<string, User> = new Map();

    async find(id: string): Promise<User | null> {
        return this.users.get(id) || null;
    }

    async save(user: User): Promise<void> {
        this.users.set(user.id, user);
    }

    async delete(id: string): Promise<boolean> {
        return this.users.delete(id);
    }

    // Additional method not in interface
    async findAll(): Promise<User[]> {
        return Array.from(this.users.values());
    }
}

/**
 * Generic class
 */
class Container<T> {
    private value: T;

    constructor(value: T) {
        this.value = value;
    }

    getValue(): T {
        return this.value;
    }

    setValue(value: T): void {
        this.value = value;
    }

    map<U>(fn: (value: T) => U): Container<U> {
        return new Container(fn(this.value));
    }
}

/**
 * Namespace definition
 */
namespace Utils {
    export function capitalize(str: string): string {
        return str.charAt(0).toUpperCase() + str.slice(1);
    }

    export function slugify(str: string): string {
        return str.toLowerCase().replace(/\s+/g, '-');
    }
}

/**
 * Decorator function
 */
function log(target: any, propertyKey: string, descriptor: PropertyDescriptor) {
    const originalMethod = descriptor.value;
    descriptor.value = function(...args: any[]) {
        console.log(`Calling ${propertyKey} with`, args);
        return originalMethod.apply(this, args);
    };
}

/**
 * Class using decorator
 */
class Service {
    @log
    process(data: string): string {
        return data.toUpperCase();
    }
}

// Export statements
export { User, UserRepository, Status, Container };
export type { Config, Repository };
