/**
 * Sample C++ file for testing semantic chunking.
 * This file contains various C++ constructs to test extraction.
 */

#include <iostream>
#include <string>
#include <vector>
#include <memory>
#include <algorithm>

// Namespace definition
namespace math {
    /**
     * Simple function in namespace
     */
    int add(int x, int y) {
        return x + y;
    }

    /**
     * Template function
     */
    template<typename T>
    T max(T a, T b) {
        return (a > b) ? a : b;
    }
}

/**
 * Simple struct
 */
struct Point {
    double x;
    double y;

    Point(double x = 0.0, double y = 0.0) : x(x), y(y) {}

    double distanceFromOrigin() const {
        return std::sqrt(x * x + y * y);
    }
};

/**
 * Base class
 */
class Animal {
protected:
    std::string name;

public:
    Animal(const std::string& name) : name(name) {}
    virtual ~Animal() = default;

    virtual std::string makeSound() const = 0;

    void move(int distance) const {
        std::cout << name << " moved " << distance << "m." << std::endl;
    }

    std::string getName() const {
        return name;
    }
};

/**
 * Derived class
 */
class Dog : public Animal {
private:
    std::string breed;

public:
    Dog(const std::string& name, const std::string& breed)
        : Animal(name), breed(breed) {}

    std::string makeSound() const override {
        return "Woof!";
    }

    std::string getBreed() const {
        return breed;
    }

    // Static method
    static Dog* createPet(const std::string& name) {
        return new Dog(name, "Mixed");
    }
};

/**
 * Template class
 */
template<typename T>
class Container {
private:
    T value;

public:
    Container(const T& value) : value(value) {}

    T getValue() const {
        return value;
    }

    void setValue(const T& value) {
        this->value = value;
    }

    template<typename U>
    Container<U> map(std::function<U(T)> fn) const {
        return Container<U>(fn(value));
    }
};

/**
 * Enum class (C++11)
 */
enum class Status {
    Pending,
    Active,
    Completed,
    Failed
};

/**
 * Interface (abstract class)
 */
template<typename T>
class Repository {
public:
    virtual ~Repository() = default;
    virtual std::unique_ptr<T> find(const std::string& id) const = 0;
    virtual void save(const std::string& id, const T& entity) = 0;
    virtual bool remove(const std::string& id) = 0;
};

/**
 * User class
 */
class User {
private:
    std::string id;
    std::string name;
    std::string email;

public:
    User(const std::string& id, const std::string& name, const std::string& email)
        : id(id), name(name), email(email) {}

    // Copy constructor
    User(const User& other) = default;

    // Move constructor
    User(User&& other) noexcept = default;

    // Copy assignment
    User& operator=(const User& other) = default;

    // Move assignment
    User& operator=(User&& other) noexcept = default;

    ~User() = default;

    // Getters
    std::string getId() const { return id; }
    std::string getName() const { return name; }
    std::string getEmail() const { return email; }

    // Setters
    void setName(const std::string& name) { this->name = name; }
    void setEmail(const std::string& email) { this->email = email; }

    // Operator overload
    friend std::ostream& operator<<(std::ostream& os, const User& user) {
        os << "User(" << user.id << ", " << user.name << ", " << user.email << ")";
        return os;
    }
};

/**
 * Repository implementation
 */
class UserRepository : public Repository<User> {
private:
    std::map<std::string, User> users;

public:
    std::unique_ptr<User> find(const std::string& id) const override {
        auto it = users.find(id);
        if (it != users.end()) {
            return std::make_unique<User>(it->second);
        }
        return nullptr;
    }

    void save(const std::string& id, const User& user) override {
        users[id] = user;
    }

    bool remove(const std::string& id) override {
        return users.erase(id) > 0;
    }

    std::vector<User> findAll() const {
        std::vector<User> result;
        for (const auto& pair : users) {
            result.push_back(pair.second);
        }
        return result;
    }
};

/**
 * Lambda and functional programming example
 */
auto createMultiplier(int factor) {
    return [factor](int x) { return x * factor; };
}

/**
 * Const expression function (C++11)
 */
constexpr int factorial(int n) {
    return (n <= 1) ? 1 : n * factorial(n - 1);
}

/**
 * Namespace with nested namespace (C++17)
 */
namespace utils::string {
    std::string capitalize(const std::string& str) {
        if (str.empty()) return str;
        std::string result = str;
        result[0] = std::toupper(result[0]);
        return result;
    }

    std::string slugify(const std::string& str) {
        std::string result = str;
        std::transform(result.begin(), result.end(), result.begin(), ::tolower);
        std::replace(result.begin(), result.end(), ' ', '-');
        return result;
    }
}

/**
 * Template specialization
 */
template<>
class Container<std::string> {
private:
    std::string value;

public:
    Container(const std::string& value) : value(value) {}

    std::string getValue() const {
        return value;
    }

    size_t length() const {
        return value.length();
    }
};

/**
 * Main function
 */
int main(int argc, char* argv[]) {
    std::cout << math::add(5, 3) << std::endl;
    std::cout << math::max(10, 20) << std::endl;

    Dog dog("Buddy", "Golden Retriever");
    std::cout << dog.makeSound() << std::endl;

    Container<int> intContainer(42);
    std::cout << intContainer.getValue() << std::endl;

    auto multiply = createMultiplier(5);
    std::cout << multiply(10) << std::endl;

    return 0;
}
