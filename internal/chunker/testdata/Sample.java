package com.example.chunker.testdata;

import java.util.*;
import java.util.stream.Collectors;

/**
 * Sample Java file for testing semantic chunking.
 * This file contains various Java constructs to test extraction.
 */
public class Sample {

    // Constants
    private static final int MAX_RETRIES = 3;
    private static final String DEFAULT_NAME = "Unknown";

    /**
     * Simple static method
     * @param name The name to greet
     * @return The greeting message
     */
    public static String greet(String name) {
        return "Hello, " + name + "!";
    }

    /**
     * Method with multiple parameters
     */
    public static int add(int x, int y, int z) {
        return x + y + z;
    }
}

/**
 * Base interface for repositories
 */
interface Repository<T> {
    Optional<T> findById(String id);
    void save(T entity);
    void delete(String id);
    List<T> findAll();
}

/**
 * User entity class
 */
class User {
    private String id;
    private String name;
    private String email;

    /**
     * Constructor
     */
    public User(String id, String name, String email) {
        this.id = id;
        this.name = name;
        this.email = email;
    }

    // Getters
    public String getId() {
        return id;
    }

    public String getName() {
        return name;
    }

    public String getEmail() {
        return email;
    }

    // Setters
    public void setName(String name) {
        this.name = name;
    }

    public void setEmail(String email) {
        this.email = email;
    }

    @Override
    public String toString() {
        return String.format("User{id='%s', name='%s', email='%s'}", id, name, email);
    }
}

/**
 * Abstract base class
 */
abstract class Animal {
    protected String name;

    public Animal(String name) {
        this.name = name;
    }

    public abstract String makeSound();

    public void move(int distance) {
        System.out.println(name + " moved " + distance + "m.");
    }
}

/**
 * Concrete implementation
 */
class Dog extends Animal {
    private String breed;

    public Dog(String name, String breed) {
        super(name);
        this.breed = breed;
    }

    @Override
    public String makeSound() {
        return "Woof!";
    }

    public String getBreed() {
        return breed;
    }
}

/**
 * Implementation of Repository interface
 */
class UserRepository implements Repository<User> {
    private Map<String, User> users = new HashMap<>();

    @Override
    public Optional<User> findById(String id) {
        return Optional.ofNullable(users.get(id));
    }

    @Override
    public void save(User user) {
        users.put(user.getId(), user);
    }

    @Override
    public void delete(String id) {
        users.remove(id);
    }

    @Override
    public List<User> findAll() {
        return new ArrayList<>(users.values());
    }

    public List<User> findByName(String name) {
        return users.values().stream()
            .filter(u -> u.getName().equals(name))
            .collect(Collectors.toList());
    }
}

/**
 * Generic class example
 */
class Container<T> {
    private T value;

    public Container(T value) {
        this.value = value;
    }

    public T getValue() {
        return value;
    }

    public void setValue(T value) {
        this.value = value;
    }

    public <U> Container<U> map(Function<T, U> mapper) {
        return new Container<>(mapper.apply(value));
    }
}

/**
 * Enum with methods
 */
enum Status {
    PENDING("Pending"),
    ACTIVE("Active"),
    COMPLETED("Completed"),
    FAILED("Failed");

    private final String displayName;

    Status(String displayName) {
        this.displayName = displayName;
    }

    public String getDisplayName() {
        return displayName;
    }

    public boolean isTerminal() {
        return this == COMPLETED || this == FAILED;
    }
}

/**
 * Record class (Java 14+)
 */
record Point(int x, int y) {
    public double distanceFromOrigin() {
        return Math.sqrt(x * x + y * y);
    }
}

/**
 * Annotation definition
 */
@interface ValidatedBy {
    Class<?> validator();
    String message() default "Validation failed";
}

/**
 * Service class with various methods
 */
class UserService {
    private final UserRepository repository;

    public UserService(UserRepository repository) {
        this.repository = repository;
    }

    public User createUser(String name, String email) {
        String id = UUID.randomUUID().toString();
        User user = new User(id, name, email);
        repository.save(user);
        return user;
    }

    public Optional<User> getUser(String id) {
        return repository.findById(id);
    }

    public void deleteUser(String id) {
        repository.delete(id);
    }

    public List<User> getAllUsers() {
        return repository.findAll();
    }
}
