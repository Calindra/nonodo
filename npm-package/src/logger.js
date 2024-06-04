import colorize from "chalk"

/**
 * Enum for log levels
 * @readonly
 * @enum {number}
 */
export const Levels = {
    ERROR: 1,
    WARN: 2,
    INFO: 3,
    DEBUG: 4,
}


export class Logger {
    /** @type {string} */
    #prefix;
    #level;

    constructor(prefix, level = Levels.INFO) {
        this.#prefix = prefix;
        this.#level = level;
    }

    /**
     * @param {string} message
     * @param {string} level
     */
    #log = (message, level) => {
        if (Levels[level] > this.#level) {
            return;
        }

        const prefixName = this.#prefix || "Brunodo";
        const prefix = `[${prefixName} ${level}] `;
        console.log(`${prefix}${message}`);
    }

    error = (message) => {
        this.#log(message, "ERROR");
    }

    warn = (message) => {
        this.#log(message, "WARN");
    }

    info = (message) => {
        this.#log(message, "INFO");
    }

    debug = (message) => {
        this.#log(message, "DEBUG");
    }
}