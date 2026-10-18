"use strict";
const { chromium } = require("playwright");
const { spawn, execFileSync } = require("node:child_process");
const { watch } = require("node:fs");
const { mkdtemp, readFile, rm, mkdir, writeFile } = require("node:fs/promises");
const { tmpdir } = require("node:os");
const { resolve, join } = require("node:path");
const assert = require("node:assert/strict");

(async () => {
  const directory = await mkdtemp(join(tmpdir(), "akari-development-"));
  const artifacts = join(__dirname, "artifacts");
  await mkdir(artifacts, { recursive: true });
  let host, exited, address, token, browser;
  async function start(extra = {}) {
    host = spawn(resolve(process.argv[2]), [], {
      env: { ...process.env, AKARI_DATA_DIR: directory, AKARI_ADDR: "127.0.0.1:0", AKARI_ADMIN_ADDR: "127.0.0.1:0", AKARI_ADMIN_AUTH_DISABLED: "false", AKARI_DOCS_REPO: "", ...extra },
      stdio: ["ignore", "pipe", "pipe"],
    });
    exited = new Promise((done) => host.once("exit", (code) => done(code)));
    let logs = "";
    await new Promise((done, fail) => {
      const timeout = setTimeout(() => fail(new Error("Development host startup timed out")), 10000);
      const capture = (chunk) => {
        logs += chunk.toString();
        if (logs.includes("foundation host listening")) { clearTimeout(timeout); done(); }
      };
      host.stderr.on("data", capture);
      host.stdout.on("data", capture);
      host.once("error", (err) => { clearTimeout(timeout); fail(err); });
      host.once("exit", () => { clearTimeout(timeout); fail(new Error("Development host exited before startup")); });
    });
    address = `http://${logs.match(/management console listening.*address=(127\.0\.0\.1:\d+)/)[1]}`;
    token = (await readFile(join(directory, "admin.token"), "utf8")).trim();
  }
  async function stop(signal = "SIGTERM") {
    if (!host) return;
    host.kill(signal);
    const deadline = setTimeout(() => host.kill("SIGKILL"), 6000);
    const code = await exited;
    clearTimeout(deadline);
    host = null;
    if (signal === "SIGTERM") assert.equal(code, 0, "Host did not stop cleanly");
  }
  async function login(page, hash = "tasks") {
    await page.goto(`${address}/#${hash}`);
    await page.locator("#code").fill(token);
    await page.locator("#login-form button").click();
    await page.locator("#app:not([hidden])").waitFor();
  }
  async function reauthenticate(page) {
    await page.locator("#login:not([hidden])").waitFor();
    await page.locator("#code").fill(token);
    await page.locator("#login-form button").click();
    await page.locator("#app:not([hidden])").waitFor();
  }
  async function mcp(name, args) {
    const response = await fetch(`${address}/mcp`, {
      method: "POST", headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json", Accept: "application/json, text/event-stream", "MCP-Protocol-Version": "2025-11-25" },
      body: JSON.stringify({ jsonrpc: "2.0", id: 1, method: "tools/call", params: { name, arguments: args } }),
    });
    assert.equal(response.status, 200);
    const result = await response.json();
    assert.equal(result.error, undefined, `MCP ${name} protocol error`);
    assert.equal(result.result.isError, undefined, `MCP ${name} tool error`);
    return result.result.structuredContent;
  }
  try {
    await start();
    browser = await chromium.launch({ executablePath: process.env.AKARI_CHROMIUM || undefined, headless: true, args: ["--no-sandbox"] });
    const context = await browser.newContext({ viewport: { width: 1500, height: 1050 }, reducedMotion: "reduce" });
    const page = await context.newPage();
    const errors = [], external = [];
    page.on("pageerror", (err) => errors.push(err.message));
    page.on("request", (request) => { if (!request.url().startsWith(`${address}/`) && !request.url().startsWith("data:")) external.push(request.url()); });
    await login(page);
    await page.locator(".task-card").first().waitFor();
    assert.ok(await page.locator(".task-card").count() >= 20);
    await page.locator("#task-filter-status").selectOption("decision");
    assert.ok(await page.locator(".task-column-decision .task-card").count() >= 5);
    await page.locator("#task-search").fill("not-found-task");
    await page.locator("#task-empty:not([hidden])").waitFor();
    await page.locator("#task-search").fill("");
    await page.locator("#task-filter-status").selectOption("active");
    await page.screenshot({ path: join(artifacts, "development-tasks.png"), fullPage: true });

    await page.locator("#decision-new").click();
    assert.equal(await page.locator("#task-status").inputValue(), "decision");
    assert.equal(await page.locator("#task-category").inputValue(), "decision");
    await page.locator("#task-title").fill("Console judgment pending");
    await page.locator("#task-description").fill("A decision can be recorded directly from the console.");
    await page.locator("#task-save").click();
    await page.locator("#task-dialog").waitFor({ state: "hidden" });
    const decisionBoard = await mcp("tasks_list", { status: "decision" });
    assert.ok(decisionBoard.tasks.some((task) => task.title === "Console judgment pending" && task.category === "decision"));
    await page.locator("#task-filter-status").selectOption("decision");
    assert.ok(await page.locator(".task-column-decision .task-card").filter({ hasText: "Console judgment pending" }).count());
    await page.locator("#task-filter-status").selectOption("active");

    await page.locator("#task-new").click();
    const title = '<img src="https://invalid.example/task"> API task';
    await page.locator("#task-title").fill(title);
    await page.locator("#task-description").fill("UI and MCP use the same database.");
    await page.locator("#task-notes").fill("created in console");
    await page.locator("#task-priority").selectOption("high");
    await page.locator("#task-references").selectOption("design/05-delivery.md");
    let releaseTask, taskStarted;
    const savingTask = new Promise((done) => { taskStarted = done; });
    const resumeTask = new Promise((done) => { releaseTask = done; });
    const delayedTask = async (route) => {
      if (route.request().method() !== "POST") return route.continue();
      taskStarted();
      await resumeTask;
      await route.continue();
    };
    await page.route("**/api/development/tasks", delayedTask);
    await page.locator("#task-save").click();
    await savingTask;
    assert.equal(await page.locator("#task-title").isDisabled(), true);
    assert.equal(await page.locator("#task-notes").isDisabled(), true);
    assert.equal(await page.locator("#task-status").isDisabled(), true);
    await page.keyboard.press("Escape");
    assert.equal(await page.locator("#task-dialog").isVisible(), true);
    releaseTask();
    await page.locator("#task-dialog").waitFor({ state: "hidden" });
    await page.unroute("**/api/development/tasks", delayedTask);
    const board = await mcp("tasks_list", {});
    const task = board.tasks.find((value) => value.title === title);
    assert.ok(task);
    assert.equal(task.notes, "created in console");
    assert.equal(await page.locator('img[src="https://invalid.example/task"]').count(), 0);

    await page.locator(`.task-card[data-task-id="${task.id}"] .task-title`).click();
    await page.locator("#task-notes").fill("unsaved conflict draft");
    const edited = { ...task, status: "in_progress", notes: "updated by MCP" };
    await mcp("tasks_update", { task: edited });
    await page.locator("#task-save").click();
    await page.locator("#task-form-error:not([hidden])").waitFor();
    assert.equal(await page.locator("#task-notes").inputValue(), "unsaved conflict draft");
    assert.ok((await page.locator("#task-form-error").textContent()).includes("更新"));
    await page.locator("#task-cancel").click();
    await page.locator("#tasks-refresh").click();
    await page.waitForFunction((id) => document.querySelector(`.task-card[data-task-id="${id}"] select`)?.value === "in_progress", task.id);
    assert.equal((await mcp("tasks_get", { id: task.id })).notes, "updated by MCP");

    await page.locator(`.task-card[data-task-id="${task.id}"] .task-title`).click();
    await page.locator("#task-notes").fill("reauthentication draft");
    await context.clearCookies();
    await page.locator("#task-save").click();
    await reauthenticate(page);
    await page.locator("#task-dialog[open]").waitFor();
    assert.equal(await page.locator("#task-notes").inputValue(), "reauthentication draft");
    await page.locator("#task-cancel").click();

    const download = page.waitForEvent("download");
    await page.locator("#tasks-export").click();
    const exported = JSON.parse(await readFile(await (await download).path(), "utf8"));
    assert.ok(exported.tasks.some((value) => value.id === task.id));
    await page.setViewportSize({ width: 390, height: 844 });
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), "Mobile task board overflow");
    await page.screenshot({ path: join(artifacts, "development-mobile.png"), fullPage: true });
    await page.setViewportSize({ width: 1500, height: 1050 });

    await page.locator("[data-nav=docs]").click();
    await page.locator("#doc-new").click();
    await page.locator("#doc-edit-path").fill("design/browser-note.md");
    await page.locator("#doc-content").fill("# Browser note\n\nCommitted from the console.\n");
    await page.locator("#doc-save").click();
    await page.locator("#doc-dialog").waitFor({ state: "hidden" });
    await page.locator("#article h1").filter({ hasText: "Browser note" }).waitFor();
    const document = await mcp("documents_read", { path: "design/browser-note.md" });
    assert.ok(document.content.includes("Committed from the console."));
    assert.equal(document.revision.length, 40);
    const previous = document.revision;
    await page.locator("#doc-edit").click();
    await page.locator("#doc-content").fill("# Browser note\n\nEdited twice.\n");
    let releaseWrite, writeStarted;
    const waitingWrite = new Promise((done) => { writeStarted = done; });
    const resumeWrite = new Promise((done) => { releaseWrite = done; });
    const delayedWrite = async (route) => {
      if (route.request().method() !== "PUT") return route.continue();
      writeStarted();
      await resumeWrite;
      await route.continue();
    };
    await page.route("**/api/docs", delayedWrite);
    await page.locator("#doc-save").click();
    await waitingWrite;
    assert.equal(await page.locator("#doc-cancel").isDisabled(), true);
    assert.equal(await page.locator("#doc-content").isDisabled(), true);
    assert.equal(await page.locator("#doc-edit-path").isDisabled(), true);
    await page.keyboard.press("Escape");
    assert.equal(await page.locator("#doc-dialog").isVisible(), true);
    releaseWrite();
    await page.locator("#doc-dialog").waitFor({ state: "hidden" });
    await page.unroute("**/api/docs", delayedWrite);
    await page.locator("#article").filter({ hasText: "Edited twice." }).waitFor();
    assert.notEqual((await mcp("documents_read", { path: document.path })).revision, previous);

    await page.locator("#doc-edit").click();
    const capturedRevision = await page.locator("#doc-edit-revision").textContent();
    await page.locator("#doc-content").fill("# Reauthentication draft\n");
    await context.clearCookies();
    await page.locator("#doc-save").click();
    const current = await mcp("documents_read", { path: document.path });
    await mcp("documents_write", { document: { ...current, content: "# Browser note\n\nEdited twice.\n\nMCP changed.\n" } });
    await reauthenticate(page);
    await page.locator("#doc-dialog[open]").waitFor();
    assert.equal(await page.locator("#doc-content").inputValue(), "# Reauthentication draft\n");
    assert.equal(await page.locator("#doc-edit-revision").textContent(), capturedRevision);
    await page.locator("#doc-save").click();
    await page.locator("#doc-form-error:not([hidden])").waitFor();
    assert.ok((await page.locator("#doc-form-error").textContent()).includes("更新"));
    await page.locator("#doc-cancel").click();
    await page.screenshot({ path: join(artifacts, "git-document.png"), fullPage: true });
    await page.setViewportSize({ width: 390, height: 844 });
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), "Mobile document actions overflow");
    await page.locator("#doc-edit").click();
    assert.ok(await page.locator("#doc-dialog").evaluate((node) => node.getBoundingClientRect().width <= innerWidth), "Mobile document editor overflow");
    await page.locator("#doc-cancel").click();
    await page.setViewportSize({ width: 1500, height: 1050 });

    await stop("SIGKILL");
    await start();
    assert.equal((await mcp("tasks_get", { id: task.id })).notes, "updated by MCP");
    assert.ok((await mcp("documents_read", { path: document.path })).content.includes("Edited twice."));
    await login(page);
    await page.locator(`.task-card[data-task-id="${task.id}"] .task-title`).click();
    page.once("dialog", (dialog) => dialog.accept());
    await page.locator("#task-delete").click();
    await page.locator("#task-dialog").waitFor({ state: "hidden" });
    assert.equal((await mcp("tasks_list", {})).tasks.some((value) => value.id === task.id), false);

    await stop();
    const git = (...args) => execFileSync("git", ["-C", join(directory, "documentation"), "-c", "user.name=Browser test", "-c", "user.email=browser@localhost", "-c", "core.hooksPath=/dev/null", ...args], { stdio: "pipe" });
    git("rm", "-r", "docs", "design");
    git("commit", "--no-gpg-sign", "-m", "Empty document library");
    await start();
    await login(page, "docs");
    const empty = await mcp("documents_list", {});
    assert.equal(empty.documents.length, 0);
    assert.equal(empty.revision.length, 40);
    await page.locator("#doc-new").click();
    await page.locator("#doc-edit-path").fill("docs/first.md");
    await page.locator("#doc-content").fill("# First document\n");
    await page.locator("#doc-save").click();
    await page.locator("#doc-dialog").waitFor({ state: "hidden" });
    await page.locator("#article h1").filter({ hasText: "First document" }).waitFor();
    assert.equal((await mcp("documents_list", {})).documents.length, 1);
    await page.locator("#logout").click();
    await page.locator("#login:not([hidden])").waitFor();
    assert.equal((await context.request.get(`${address}/api/development/tasks`)).status(), 401);
    assert.deepEqual(errors, [], "Development UI JavaScript errors");
    assert.deepEqual(external, [], "External requests from development UI");

    await stop();
    const wrapper = join(directory, "git-wrapper");
    await mkdir(wrapper);
    const armed = join(wrapper, "armed"), marker = join(wrapper, "started");
    const realGit = execFileSync("which", ["git"], { encoding: "utf8" }).trim();
    await writeFile(join(wrapper, "git"), `#!/usr/bin/python3
import os, sys, time
if os.path.exists(${JSON.stringify(armed)}) and ("status" in sys.argv or "add" in sys.argv):
    with open(${JSON.stringify(marker)}, "w") as signal:
        signal.write(str(os.getpid()))
    time.sleep(3.5)
os.execv(${JSON.stringify(realGit)}, [${JSON.stringify(realGit)}, *sys.argv[1:]])
`, { mode: 0o700 });
    await start({ PATH: `${wrapper}:${process.env.PATH}` });
    const initial = await mcp("documents_list", {});
    await writeFile(armed, "delay status and staging past the host shutdown deadline");
    const started = new Promise((done, fail) => {
      const timeout = setTimeout(() => { watcher.close(); fail(new Error("Git write did not start")); }, 10000);
      const watcher = watch(wrapper, (_, name) => {
        if (name === "started") { clearTimeout(timeout); watcher.close(); done(); }
      });
    });
    const pendingWrite = mcp("documents_write", { document: { path: "docs/shutdown.md", content: "# Drained before exit\n", revision: initial.revision } }).catch(() => null);
    await started;
    host.kill("SIGTERM");
    const deadline = setTimeout(() => host.kill("SIGKILL"), 15000);
    const exitCode = await exited;
    clearTimeout(deadline);
    host = null;
    await pendingWrite;
    assert.equal(exitCode, 1, "Shutdown deadline should be reported after draining");
    const committed = execFileSync(realGit, ["-C", join(directory, "documentation"), "show", "HEAD:docs/shutdown.md"], { encoding: "utf8" });
    assert.equal(committed, "# Drained before exit\n", "Host exited before its Git subprocess completed");
    await rm(armed);
    await start();
    assert.equal((await mcp("documents_read", { path: "docs/shutdown.md" })).content, committed);
    console.log("Development browser passed: task CRUD, MCP/API/console sharing, stale and reauthenticated drafts, save-in-flight guard, JSON export, mobile, Git commits, empty document library, SIGKILL persistence, logout, no external requests");
  } finally {
    if (browser) await browser.close();
    await stop();
    await rm(directory, { recursive: true, force: true });
  }
})().catch((error) => { console.error(error.stack); process.exitCode = 1; });
