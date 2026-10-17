"use strict";
(() => {
  const $ = (id) => document.getElementById(id);
  const endpoint = "/api/development/tasks";
  const statuses = { todo: "未着手", in_progress: "進行中", blocked: "保留・ブロック", decision: "判断待ち", done: "完了" };
  const priorities = { high: "高", medium: "中", low: "低" };
  const categories = { runtime: "思考・対話", model: "LLM", tools: "道具・行為境界", inner: "内面・記憶", console: "管理画面", operations: "運用", decision: "決定事項" };
  const text = (tag, value, className = "") => {
    const node = document.createElement(tag);
    node.textContent = value;
    node.className = className;
    return node;
  };
  let board, api, go, documents = [], editing = null, formRevision, busy = false;
  function choices(select, values) {
    for (const [value, label] of Object.entries(values)) select.add(new Option(label, value));
  }
  choices($("task-filter-category"), categories);
  choices($("task-category"), categories);
  choices($("task-status"), statuses);
  function announce(message, failure = false, form = false) {
    const node = $(failure ? (form ? "task-form-error" : "task-page-error") : "task-notice");
    node.textContent = message;
    node.hidden = !message;
  }
  async function refresh() {
    const latest = await api(endpoint);
    board = latest;
    announce("", true);
    render();
  }
  function render() {
    if (!board) return;
    const done = board.tasks.filter((task) => task.status === "done").length;
    const decisions = board.tasks.filter((task) => task.status === "decision").length;
    const progress = text("div", "", "task-progress-card");
    const meter = document.createElement("progress");
    meter.max = Math.max(1, board.tasks.length);
    meter.value = done;
    meter.setAttribute("aria-label", "開発タスクの完了数");
    progress.append(text("span", "開発の進み具合"), text("strong", `${done} / ${board.tasks.length} 完了`), meter);
    $("task-summary").replaceChildren(progress);
    for (const [label, count] of [["残りのタスク", board.tasks.length - done], ["判断待ち", decisions], ["進行中", board.tasks.filter((task) => task.status === "in_progress").length]]) {
      const card = text("div", "", "task-summary-card");
      card.append(text("span", label), text("strong", count));
      $("task-summary").append(card);
    }
    const query = $("task-search").value.trim().toLocaleLowerCase();
    const status = $("task-filter-status").value;
    const priority = $("task-filter-priority").value;
    const category = $("task-filter-category").value;
    const matching = board.tasks.filter((task) =>
      (status === "all" || (status === "active" ? task.status !== "done" : task.status === status)) &&
      (priority === "all" || task.priority === priority) &&
      (category === "all" || task.category === category) &&
      [task.id, task.title, task.description, task.notes].join("\n").toLocaleLowerCase().includes(query));
    const order = { high: 0, medium: 1, low: 2 };
    matching.sort((left, right) => order[left.priority] - order[right.priority] || left.id.localeCompare(right.id));
    const columns = Object.entries(statuses).filter(([value]) => status === "all" || (status === "active" ? value !== "done" : value === status));
    $("task-board").replaceChildren();
    $("task-board").dataset.columns = columns.length;
    for (const [value, label] of columns) {
      const column = text("section", "", `task-column task-column-${value}`);
      column.setAttribute("aria-label", label);
      const tasks = matching.filter((task) => task.status === value);
      const heading = text("h2", label);
      heading.append(text("span", tasks.length, "task-count"));
      column.append(heading);
      if (!tasks.length) column.append(text("p", "タスクなし", "task-column-empty"));
      for (const task of tasks) column.append(card(task));
      $("task-board").append(column);
    }
    $("task-empty").hidden = matching.length > 0;
    $("task-updated").textContent = `保存済み ${new Date(board.updatedAt).toLocaleString("ja-JP")} · ${matching.length}件を表示`;
    document.querySelectorAll("#tasks-view button, #task-form button").forEach((button) => { button.disabled = busy; });
  }
  function card(task) {
    const node = text("article", "", "task-card");
    node.dataset.taskId = task.id;
    const meta = text("div", "", "task-card-meta");
    meta.append(text("span", task.id), text("span", `優先度 ${priorities[task.priority]}`, `task-priority task-priority-${task.priority}`));
    const title = text("button", task.title, "task-title");
    title.type = "button";
    title.addEventListener("click", () => edit(task));
    node.append(meta, title, text("p", task.description, "task-description"), text("span", categories[task.category], "pill"));
    if (task.notes) node.append(text("p", task.notes, "task-card-notes"));
    const refs = text("div", "", "task-links");
    for (const reference of task.references || []) {
      const [path, section] = reference.split("#");
      const link = text("a", documents.find((doc) => doc.path === path)?.title || path);
      link.href = go(path, section || "");
      refs.append(link);
    }
    node.append(refs);
    const label = text("label", "状態を変更", "task-card-status");
    const select = document.createElement("select");
    choices(select, statuses);
    select.value = task.status;
    select.disabled = busy;
    select.setAttribute("aria-label", `${task.title}の状態`);
    select.addEventListener("change", () => mutate("PUT", { ...task, status: select.value }, task.version).catch((err) => { announce(err.message, true); render(); }));
    label.append(select);
    node.append(label);
    return node;
  }
  function edit(task) {
    editing = task?.id || null;
    formRevision = task?.version || board.revision;
    $("task-form").reset();
    $("task-dialog-title").textContent = editing ? "タスクを編集" : "タスクを追加";
    $("task-id").textContent = editing || "保存時にIDを発行します";
    for (const field of ["title", "description", "notes"]) $(`task-${field}`).value = task?.[field] || "";
    $("task-status").value = task?.status || "todo";
    $("task-priority").value = task?.priority || "medium";
    $("task-category").value = task?.category || "runtime";
    $("task-delete").hidden = !editing;
    $("task-references").replaceChildren();
    const references = new Set(task?.references || []);
    // Preserve section-specific references when editing an existing task.
    const values = new Set([...references, ...documents.map((doc) => doc.path)]);
    for (const path of values) $("task-references").add(new Option(path, path, false, references.has(path)));
    announce("", true, true);
    $("task-dialog").showModal();
  }
  async function mutate(method, task, revision, identity = task?.id) {
    if (busy) throw new Error("保存中です。完了後に操作してください。");
    busy = true;
    render();
    try {
      const latest = await api(`${endpoint}${method === "POST" ? "" : `/${encodeURIComponent(identity)}`}`, {
        method,
        headers: { "Content-Type": "application/json", "If-Match": `"${revision}"` },
        ...(method === "DELETE" ? {} : { body: JSON.stringify(task) }),
      });
      board = latest;
      announce("保存しました。");
      announce("", true);
    } finally {
      busy = false;
      render();
    }
  }
  $("task-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    const task = {
      id: editing || "", title: $("task-title").value.trim(), description: $("task-description").value,
      status: $("task-status").value, priority: $("task-priority").value, category: $("task-category").value,
      notes: $("task-notes").value, references: [...$("task-references").selectedOptions].map((option) => option.value),
    };
    try {
      await mutate(editing ? "PUT" : "POST", task, formRevision);
      $("task-dialog").close();
    } catch (err) { announce(err.message, true, true); }
  });
  $("task-delete").addEventListener("click", async () => {
    if (!confirm("この開発タスクを削除しますか？")) return;
    try {
      await mutate("DELETE", null, formRevision, editing);
      $("task-dialog").close();
    } catch (err) { announce(err.message, true, true); }
  });
  $("task-new").addEventListener("click", () => edit(null));
  $("task-cancel").addEventListener("click", () => $("task-dialog").close());
  $("task-dialog").addEventListener("cancel", (event) => { if (busy) event.preventDefault(); });
  for (const id of ["task-search", "task-filter-status", "task-filter-priority", "task-filter-category"]) $(id).addEventListener("input", render);
  $("tasks-refresh").addEventListener("click", () => refresh().then(() => announce("一覧を更新しました。")).catch((err) => announce(err.message, true)));
  $("tasks-export").addEventListener("click", async () => {
    try {
      const saved = await api(endpoint);
      const url = URL.createObjectURL(new Blob([JSON.stringify(saved, null, 2)], { type: "application/json" }));
      const link = document.createElement("a");
      link.href = url;
      link.download = "akari-development-tasks.json";
      link.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (err) { announce(err.message, true); }
  });
  window.akariTasks = {
    async open(library, request, documentationLink) {
      documents = library;
      api = request;
      go = documentationLink;
      await refresh();
    },
  };
})();
