/// Sample Rust file for testing semantic chunking.
/// This file contains various Rust constructs to test extraction.

use std::collections::HashMap;
use std::fmt;

// Module-level constant
const MAX_RETRIES: u32 = 3;

/// Simple function with documentation
pub fn greet(name: &str) -> String {
    format!("Hello, {}!", name)
}

/// Function with multiple parameters and return type
pub fn add(x: i32, y: i32, z: i32) -> i32 {
    x + y + z
}

/// Generic function
pub fn identity<T>(value: T) -> T {
    value
}

/// Function with Result return type
pub fn divide(x: f64, y: f64) -> Result<f64, String> {
    if y == 0.0 {
        Err("Division by zero".to_string())
    } else {
        Ok(x / y)
    }
}

/// Simple struct
#[derive(Debug, Clone)]
pub struct Point {
    pub x: f64,
    pub y: f64,
}

impl Point {
    /// Constructor method
    pub fn new(x: f64, y: f64) -> Self {
        Point { x, y }
    }

    /// Calculate distance from origin
    pub fn distance_from_origin(&self) -> f64 {
        (self.x * self.x + self.y * self.y).sqrt()
    }

    /// Associated function (static method)
    pub fn origin() -> Self {
        Point { x: 0.0, y: 0.0 }
    }
}

/// Struct with generic type parameter
#[derive(Debug)]
pub struct Container<T> {
    value: T,
}

impl<T> Container<T> {
    pub fn new(value: T) -> Self {
        Container { value }
    }

    pub fn get(&self) -> &T {
        &self.value
    }

    pub fn map<U, F>(self, f: F) -> Container<U>
    where
        F: FnOnce(T) -> U,
    {
        Container { value: f(self.value) }
    }
}

/// Enum definition
#[derive(Debug, PartialEq)]
pub enum Status {
    Pending,
    Active,
    Completed,
    Failed(String),
}

impl Status {
    pub fn is_terminal(&self) -> bool {
        matches!(self, Status::Completed | Status::Failed(_))
    }

    pub fn message(&self) -> &str {
        match self {
            Status::Pending => "Pending",
            Status::Active => "Active",
            Status::Completed => "Completed",
            Status::Failed(msg) => msg,
        }
    }
}

/// Trait definition
pub trait Repository<T> {
    fn find(&self, id: &str) -> Option<&T>;
    fn save(&mut self, id: String, entity: T);
    fn delete(&mut self, id: &str) -> bool;
}

/// User struct
#[derive(Debug, Clone)]
pub struct User {
    pub id: String,
    pub name: String,
    pub email: String,
}

impl User {
    pub fn new(id: String, name: String, email: String) -> Self {
        User { id, name, email }
    }
}

impl fmt::Display for User {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "User(id={}, name={}, email={})", self.id, self.name, self.email)
    }
}

/// Implementation of Repository trait
pub struct UserRepository {
    users: HashMap<String, User>,
}

impl UserRepository {
    pub fn new() -> Self {
        UserRepository {
            users: HashMap::new(),
        }
    }

    pub fn find_all(&self) -> Vec<&User> {
        self.users.values().collect()
    }
}

impl Repository<User> for UserRepository {
    fn find(&self, id: &str) -> Option<&User> {
        self.users.get(id)
    }

    fn save(&mut self, id: String, user: User) {
        self.users.insert(id, user);
    }

    fn delete(&mut self, id: &str) -> bool {
        self.users.remove(id).is_some()
    }
}

/// Async function
pub async fn fetch_data(url: &str) -> Result<String, Box<dyn std::error::Error>> {
    // Simulated async operation
    Ok(format!("Data from {}", url))
}

/// Module definition
pub mod utils {
    /// Capitalize a string
    pub fn capitalize(s: &str) -> String {
        let mut chars = s.chars();
        match chars.next() {
            None => String::new(),
            Some(first) => first.to_uppercase().collect::<String>() + chars.as_str(),
        }
    }

    /// Slugify a string
    pub fn slugify(s: &str) -> String {
        s.to_lowercase().replace(' ', "-")
    }
}

/// Type alias
pub type UserId = String;
pub type UserMap = HashMap<UserId, User>;

/// Const function
pub const fn const_add(x: i32, y: i32) -> i32 {
    x + y
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_greet() {
        assert_eq!(greet("World"), "Hello, World!");
    }

    #[test]
    fn test_point_distance() {
        let p = Point::new(3.0, 4.0);
        assert_eq!(p.distance_from_origin(), 5.0);
    }
}
