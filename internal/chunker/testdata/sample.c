/**
 * Sample C file for testing semantic chunking.
 * This file contains various C constructs to test extraction.
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* Constants */
#define MAX_RETRIES 3
#define BUFFER_SIZE 256

/**
 * Simple function with documentation
 */
int add(int x, int y) {
    return x + y;
}

/**
 * Function with multiple parameters
 */
int calculate(int x, int y, int z) {
    return x + y * z;
}

/**
 * Function returning pointer
 */
char* create_greeting(const char* name) {
    char* greeting = (char*)malloc(BUFFER_SIZE * sizeof(char));
    if (greeting != NULL) {
        snprintf(greeting, BUFFER_SIZE, "Hello, %s!", name);
    }
    return greeting;
}

/**
 * Void function (no return)
 */
void print_message(const char* message) {
    printf("%s\n", message);
}

/* Struct definition */
struct Point {
    double x;
    double y;
};

/**
 * Function operating on struct
 */
double point_distance(struct Point p1, struct Point p2) {
    double dx = p2.x - p1.x;
    double dy = p2.y - p1.y;
    return sqrt(dx * dx + dy * dy);
}

/**
 * Function returning struct
 */
struct Point point_create(double x, double y) {
    struct Point p;
    p.x = x;
    p.y = y;
    return p;
}

/* Typedef'd struct */
typedef struct {
    char* id;
    char* name;
    char* email;
} User;

/**
 * Function with typedef'd struct
 */
User* user_create(const char* id, const char* name, const char* email) {
    User* user = (User*)malloc(sizeof(User));
    if (user != NULL) {
        user->id = strdup(id);
        user->name = strdup(name);
        user->email = strdup(email);
    }
    return user;
}

/**
 * Function to free user memory
 */
void user_free(User* user) {
    if (user != NULL) {
        free(user->id);
        free(user->name);
        free(user->email);
        free(user);
    }
}

/* Union definition */
union Data {
    int i;
    float f;
    char str[BUFFER_SIZE];
};

/**
 * Function with union parameter
 */
void print_data(union Data data, char type) {
    switch(type) {
        case 'i':
            printf("Integer: %d\n", data.i);
            break;
        case 'f':
            printf("Float: %f\n", data.f);
            break;
        case 's':
            printf("String: %s\n", data.str);
            break;
    }
}

/* Enum definition */
enum Status {
    STATUS_PENDING,
    STATUS_ACTIVE,
    STATUS_COMPLETED,
    STATUS_FAILED
};

typedef enum Status Status;

/**
 * Function returning enum
 */
Status get_status(int code) {
    switch(code) {
        case 0: return STATUS_PENDING;
        case 1: return STATUS_ACTIVE;
        case 2: return STATUS_COMPLETED;
        default: return STATUS_FAILED;
    }
}

/**
 * Function pointer typedef
 */
typedef int (*BinaryOp)(int, int);

/**
 * Function accepting function pointer
 */
int apply_op(int x, int y, BinaryOp op) {
    return op(x, y);
}

/**
 * Static function (file scope)
 */
static int internal_calculation(int x) {
    return x * 2 + 1;
}

/**
 * Inline function
 */
inline int square(int x) {
    return x * x;
}

/**
 * Variadic function
 */
int sum_all(int count, ...) {
    va_list args;
    va_start(args, count);

    int sum = 0;
    for (int i = 0; i < count; i++) {
        sum += va_arg(args, int);
    }

    va_end(args);
    return sum;
}

/**
 * Array parameter function
 */
void process_array(int arr[], int size) {
    for (int i = 0; i < size; i++) {
        arr[i] *= 2;
    }
}

/**
 * Main function
 */
int main(int argc, char* argv[]) {
    struct Point p1 = point_create(0.0, 0.0);
    struct Point p2 = point_create(3.0, 4.0);

    printf("Distance: %f\n", point_distance(p1, p2));

    char* greeting = create_greeting("World");
    if (greeting != NULL) {
        print_message(greeting);
        free(greeting);
    }

    User* user = user_create("1", "John Doe", "john@example.com");
    if (user != NULL) {
        printf("User: %s <%s>\n", user->name, user->email);
        user_free(user);
    }

    return 0;
}
