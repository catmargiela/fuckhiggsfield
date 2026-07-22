#!/usr/bin/env node
"use strict";

const { spawn } = require("child_process");
const path = require("path");
const fs = require("fs");

const binName = process.platform === "win32" ? "bara.exe" : "bara";
const bin = path.join(__dirname, "..", "vendor", binName);

if (!fs.existsSync(bin)) {
  console.error(
    `@catmargiela/bara: binary not found at ${bin}\n` +
      "Reinstall with: npm install -g @catmargiela/bara"
  );
  process.exit(1);
}

const child = spawn(bin, process.argv.slice(2), {
  stdio: "inherit",
  env: { ...process.env, BARA_INSTALL_METHOD: "npm" },
});

child.on("exit", (code, signal) => {
  if (signal) process.kill(process.pid, signal);
  else process.exit(code ?? 0);
});

child.on("error", (err) => {
  console.error("@catmargiela/bara: failed to exec —", err.message);
  process.exit(1);
});
