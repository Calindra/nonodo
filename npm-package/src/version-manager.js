#!/usr/bin/env node
"use strict";

import { execute } from "@oclif/core";


execute({
  dir: import.meta.url,
  development: process.env.NODE_ENV === "development",
});