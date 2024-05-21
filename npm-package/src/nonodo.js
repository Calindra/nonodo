#!/usr/bin/env node
const {
  existsSync,
  createReadStream,
  readFileSync,
  writeFileSync,
} = require("node:fs");
const { Buffer } = require("node:buffer");
const { URL } = require("node:url");
const { spawn } = require("node:child_process");
const path = require("node:path");
const { arch, platform, tmpdir } = require("node:os");
const { version } = require("../package.json");
const { createHash } = require("node:crypto");
const { get: request } = require("node:https");
const { unzipSync } = require("node:zlib");
const { SingleBar, Presets } = require("cli-progress");
const AdmZip = require("adm-zip");

const PACKAGE_NONODO_VERSION =
  process.env.PACKAGE_NONODO_VERSION ?? version;
const PACKAGE_NONODO_URL = new URL(
  process.env.PACKAGE_NONODO_URL ??
    `https://github.com/Calindra/nonodo/releases/download/v${PACKAGE_NONODO_VERSION}/`,
);
const PACKAGE_NONODO_DIR = process.env.PACKAGE_NONODO_DIR ?? tmpdir();

const HASH_ALGO = "md5";

const AVAILABLE_BINARY_NAME = new Set([
  "darwin-amd64",
  "darwin-arm64",
  "linux-amd64",
  "linux-arm64",
  "windows-amd64",
]);

function getPlatform() {
  const plat = platform();
  if (plat === "win32") return "windows";
  else return plat;
}

function getArch() {
  const arc = arch();
  if (arc === "x64") return "amd64";
  else return arc;
}

function getReleaseName() {
  const arcName = getArch();
  const platformName = getPlatform();
  const exe = platform() === "win32" ? ".zip" : ".tar.gz";
  return `nonodo-v${PACKAGE_NONODO_VERSION}-${platformName}-${arcName}${exe}`;
}

function getBinaryName() {
  const arcName = getArch();
  const platformName = getPlatform();
  const exe = platform() === "win32" ? ".exe" : "";
  return `nonodo-v${PACKAGE_NONODO_VERSION}-${platformName}-${arcName}${exe}`;
}

const releaseName = getReleaseName();
const binaryName = getBinaryName();
const asyncController = new AbortController();

/**
 *
 * @param {string} path
 * @param {string} algorithm
 * @returns {Promise<string>}
 */
function calculateHash(path, algorithm) {
  return new Promise((resolve, reject) => {
    const stream = createReadStream(path);
    const hash = createHash(algorithm);

    stream.on("data", (chunk) => {
      hash.update(chunk);
    });

    stream.on("error", (err) => {
      reject(err);
    });

    stream.on("end", () => {
      resolve(hash.digest("hex"));
    });
  });
}

function unpackZip(zipPath, destPath) {
  const zip = new AdmZip(zipPath);
  const entry = zip.getEntry("nonodo.exe");
  if (!entry) throw new Error("Dont find binary on zip");
  const buffer = entry.getData();
  writeFileSync(destPath, buffer, { mode: 0o755 });
}

function unpackTarball(tarballPath, destPath) {
  const tarballDownloadBuffer = readFileSync(tarballPath);
  const tarballBuffer = unzipSync(tarballDownloadBuffer);
  writeFileSync(destPath, extractFileFromTarball(tarballBuffer, "nonodo"), {
    mode: 0o755,
  });
}

/**
 *
 * @param {Buffer} tarballBuffer
 * @param {string} filepath
 * @returns
 */
function extractFileFromTarball(tarballBuffer, filepath) {
  // Tar archives are organized in 512 byte blocks.
  // Blocks can either be header blocks or data blocks.
  // Header blocks contain file names of the archive in the first 100 bytes, terminated by a null byte.
  // The size of a file is contained in bytes 124-135 of a header block and in octal format.
  // The following blocks will be data blocks containing the file.

  let offset = 0;
  while (offset < tarballBuffer.length) {
    const header = tarballBuffer.slice(offset, offset + 512);
    offset += 512;

    const fileName = header.toString("utf-8", 0, 100).replace(/\0.*/g, "");
    const fileSize = parseInt(
      header.toString("utf-8", 124, 136).replace(/\0.*/g, ""),
      8,
    );

    if (fileName === filepath) {
      return tarballBuffer.subarray(offset, offset + fileSize);
    }

    // Clamp offset to the uppoer multiple of 512
    offset = (offset + fileSize + 511) & ~511;
  }
}

async function downloadBinary() {
  const dir = PACKAGE_NONODO_DIR;
  const url = new URL(PACKAGE_NONODO_URL);
  if (!url.href.endsWith("/")) url.pathname += "/";
  url.pathname += releaseName;

  console.log(`Downloading: ${url.href}`);

  const dest = path.join(dir, releaseName);

  const binary = await makeRequest(url);

  writeFileSync(dest, binary, {
    signal: asyncController.signal,
  });
}

async function downloadHash() {
  const algo = HASH_ALGO;
  const filename = `${releaseName}.${algo}`;

  const dir = PACKAGE_NONODO_DIR;
  const url = new URL(PACKAGE_NONODO_URL);
  if (!url.href.endsWith("/")) url.pathname += "/";
  url.pathname += filename;

  console.log(`Downloading: ${url.href}`);

  const dest = path.join(dir, filename);

  const response = await makeRequest(url);
  const body = response.toString("utf-8");

  writeFileSync(dest, body, {
    signal: asyncController.signal,
  });

  console.log(`Downloaded hex: ${dest}`);

  return body.trim();
}

/**
 *
 * @param {URL} url
 * @returns {Promise<Buffer>}
 */
function makeRequest(url) {
  return new Promise((resolve, reject) => {
    /** @type {SingleBar=} */
    let bar;

    const req = request(url, (res) => {
      if (res.statusCode >= 200 && res.statusCode < 300) {
        const length = parseInt(res.headers["content-length"], 10);
        const chunks = [];
        let size = 0;
        if (!Number.isNaN(length)) {
          bar = new SingleBar({}, Presets.shades_classic);
          bar.start(length, 0);
        }

        res.on("data", (chunk) => {
          // const percent = Math.floor(100 * size / length);
          // console.log(`progress ${url.pathname}`, size, "/", length, "bytes");
          chunks.push(chunk);
          size += chunk.length;
          bar?.update(size);
        });

        res.on("end", () => {
          bar?.stop();
          resolve(Buffer.concat(chunks));
        });
      } else if (
        res.statusCode >= 300 &&
        res.statusCode < 400 &&
        res.headers.location
      ) {
        makeRequest(new URL(res.headers.location)).then(resolve).catch(reject);
      } else {
        bar?.stop();
        console.error(res.statusCode, res.statusMessage);
        reject(
          new Error(`Error ${res.statusCode} when downloading the package!`),
        );
      }
    });
    req.on("error", (e) => {
      bar?.stop();
      reject(e);
    });

    asyncController.signal.addEventListener("abort", () => {
      req.destroy();
      reject(new Error("Request aborted."));
    });
  });
}

async function runNonodo(location) {
  console.log(`Running brunodo binary: ${location}`);

  const args = process.argv.slice(2);
  const nonodoBin = spawn(location, args, { stdio: "inherit" });
  nonodoBin.on("exit", (code, signal) => {
    process.on("exit", () => {
      if (signal) {
        process.kill(process.pid, signal);
      } else {
        process.exit(code ?? 1);
      }
    });
  });

  process.on("SIGINT", function () {
    nonodoBin.kill("SIGINT");
    nonodoBin.kill("SIGTERM");
  });
}

async function getNonodoAvailable() {
  const nonodoPath = PACKAGE_NONODO_DIR;

  const myPlatform = getPlatform();
  const myArch = getArch();
  const support = `${myPlatform}-${myArch}`;

  if (AVAILABLE_BINARY_NAME.has(support)) {
    const binaryPath = path.join(nonodoPath, binaryName);

    if (existsSync(binaryPath)) return binaryPath;

    console.log(`Nonodo binary not found: ${binaryPath}`);
    console.log(`Downloading nonodo binary...`);
    const [hash] = await Promise.all([downloadHash(), downloadBinary()]);

    console.log(`Downloaded nonodo binary.`);
    console.log(`Verifying hash...`);

    const releasePath = path.join(nonodoPath, releaseName);
    const calculatedHash = await calculateHash(releasePath, HASH_ALGO);

    if (hash !== calculatedHash) {
      throw new Error(
        `Hash mismatch for nonodo binary. Expected ${hash}, got ${calculatedHash}`,
      );
    }

    console.log(`Hash verified.`);

    if (getPlatform() !== "windows") {
      unpackTarball(releasePath, binaryPath);
    } else {
      /** unzip this */
      unpackZip(releasePath, binaryPath);
    }

    if (!existsSync(binaryPath)) throw new Error("Problem on unpack");

    return binaryPath;
  }

  throw new Error(`Incompatible platform.`);
}

async function tryPackageNonodo() {
  console.log(`Running brunodo ${version} for ${arch()} ${platform()}`);

  try {
    process.once("SIGINT", () => asyncController.abort());
    const nonodoPath = await getNonodoAvailable();
    console.log("nonodo path:", nonodoPath);
    await runNonodo(nonodoPath);
    return true;
  } catch (e) {
    console.error(e);
  }

  return false;
}

tryPackageNonodo()
  .then((success) => {
    if (!success) {
      process.exit(1);
    }
  })
  .catch((e) => {
    console.error(e);
    process.exit(1);
  });
