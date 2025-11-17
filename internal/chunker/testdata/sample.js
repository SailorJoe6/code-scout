/**
 * Sample JavaScript file for testing semantic chunking.
 * This file contains various JavaScript constructs to test extraction.
 */

// Module-level constant
const MAX_RETRIES = 3;

/**
 * A simple function with JSDoc.
 * @param {string} name - The name to greet
 * @returns {string} The greeting message
 */
function simpleFunction(name) {
    return `Hello, ${name}!`;
}

/**
 * Function with multiple parameters
 * @param {number} x
 * @param {number} y
 * @param {number} z
 */
function functionWithParams(x, y, z = 10) {
    return x + y + z;
}

/**
 * Async function example
 * @param {string} url
 */
async function asyncFunction(url) {
    const response = await fetch(url);
    return response.json();
}

/**
 * Arrow function stored in variable
 */
const arrowFunction = (x, y) => {
    return x + y;
};

/**
 * Single expression arrow function
 */
const shortArrow = x => x * 2;

/**
 * Base class for testing inheritance
 */
class BaseClass {
    /**
     * Constructor
     * @param {string} name
     */
    constructor(name) {
        this.name = name;
    }

    /**
     * Get the name
     * @returns {string}
     */
    getName() {
        return this.name;
    }

    /**
     * Static method
     */
    static create(name) {
        return new BaseClass(name);
    }
}

/**
 * Derived class extending BaseClass
 */
class DerivedClass extends BaseClass {
    constructor(name, value) {
        super(name);
        this.value = value;
    }

    getValue() {
        return this.value;
    }

    /**
     * Getter property
     */
    get displayName() {
        return `${this.name}: ${this.value}`;
    }

    /**
     * Setter property
     */
    set displayName(value) {
        const parts = value.split(':');
        this.name = parts[0].trim();
        this.value = parseInt(parts[1].trim());
    }

    /**
     * Async method
     */
    async fetchData() {
        return await asyncFunction('https://example.com/api');
    }
}

/**
 * Generator function
 */
function* generatorFunction(n) {
    for (let i = 0; i < n; i++) {
        yield i * 2;
    }
}

/**
 * Function with destructuring parameters
 */
function destructuringParams({ name, age, email = 'none' }) {
    return { name, age, email };
}

/**
 * Higher-order function
 */
function higherOrder(callback) {
    return function(x) {
        return callback(x * 2);
    };
}

// IIFE (Immediately Invoked Function Expression)
(function() {
    console.log('IIFE executed');
})();

// Module pattern
const Module = (function() {
    let privateVar = 'private';

    return {
        getPrivate() {
            return privateVar;
        },
        setPrivate(val) {
            privateVar = val;
        }
    };
})();

// Export for modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        simpleFunction,
        BaseClass,
        DerivedClass
    };
}
