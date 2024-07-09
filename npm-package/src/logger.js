import { format, inspect } from "node:util";

/**
 * @callback LogFn
 * @param {string} message
 * @param {any} [args]
 * @returns {void}
 */

/**
 * @typedef {Object} HaveLogFn
 * @property {LogFn} log
 */

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
};

export class Logger {
  #prefix;
  #level;
  #baseLogger;

  /**
   * @param {string} prefix
   * @param {number=} level
   * @param {HaveLogFn=} baseLogger
   */
  constructor(prefix, level = Levels.INFO, baseLogger) {
    this.#prefix = prefix;
    this.#level = level;
    this.#baseLogger = baseLogger;
  }

  /**
   * @param {string} message
   * @param {string} level
   */
  #log = (message, level) => {
    if (Levels[level] && Levels[level] > this.#level) {
      return;
    }

    const writer = this.#baseLogger ? this.#baseLogger.log :
      {
        ERROR: console.error,
        WARN: console.warn,
        INFO: console.info,
        DEBUG: console.debug,
      }[level] ?? console.log;

    const prefixName = this.#prefix || "Brunodo";
    const prefix = `[${prefixName} ${level}]`;
    const msg = inspect(message, { colors: true, depth: 4 });

    writer(prefix, msg);
  };

  error = (message) => {
    this.#log(message, "ERROR");
  };

  warn = (message) => {
    this.#log(message, "WARN");
  };

  info = (message) => {
    this.#log(message, "INFO");
  };

  debug = (message) => {
    this.#log(message, "DEBUG");
  };
}


/**
 *
 * @param {string} namespace
 * @returns {import("@oclif/core/interfaces").Logger}
 */
export function customLogger(namespace) {
  const myLogger = new Logger(namespace, Levels.DEBUG);

  return {
    child: (ns, delimiter) => customLogger(`${namespace}${delimiter ?? ':'}${ns}`),
    debug: (formatter, ...args) => myLogger.debug(format(formatter, ...args)),
    error: (formatter, ...args) => myLogger.error(format(formatter, ...args)),
    info: (formatter, ...args) => myLogger.info(format(formatter, ...args)),
    trace: (formatter, ...args) => myLogger.info(format(formatter, ...args)),
    warn: (formatter, ...args) => myLogger.warn(format(formatter, ...args)),
    namespace,
  };
}