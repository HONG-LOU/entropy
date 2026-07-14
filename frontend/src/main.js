import {
  Activity,
  ArrowDownLeft,
  ArrowUpRight,
  CircleAlert,
  CircleCheck,
  Clock3,
  Copy,
  Cpu,
  Database,
  Download,
  Eye,
  EyeOff,
  FileKey,
  History,
  KeyRound,
  LayoutDashboard,
  LoaderCircle,
  LockKeyhole,
  Network,
  Pickaxe,
  Play,
  Plus,
  RefreshCw,
  RotateCcw,
  Save,
  Send,
  ShieldCheck,
  Square,
  SquareDot,
  Trash2,
  TriangleAlert,
  Upload,
  Wifi,
  WifiOff,
  X,
  createIcons,
} from "lucide";
import "./style.css";

const appIcons = {
  Activity,
  ArrowDownLeft,
  ArrowUpRight,
  CircleAlert,
  CircleCheck,
  Clock3,
  Copy,
  Cpu,
  Database,
  Download,
  Eye,
  EyeOff,
  FileKey,
  History,
  KeyRound,
  LayoutDashboard,
  LoaderCircle,
  LockKeyhole,
  Network,
  Pickaxe,
  Play,
  Plus,
  RefreshCw,
  RotateCcw,
  Save,
  Send,
  ShieldCheck,
  Square,
  SquareDot,
  Trash2,
  TriangleAlert,
  Upload,
  Wifi,
  WifiOff,
  X,
};

const state = {
  dashboard: null,
  history: [],
  ready: false,
  startupCode: "starting",
  startupChecking: false,
  dashboardRefreshing: false,
  historyRefreshing: false,
  recoveryPhrase: "",
  pendingPruneRetain: 0,
  lastDashboardError: "",
  lastHistoryRefresh: 0,
};

const $ = (id) => document.getElementById(id);
const encoder = new TextEncoder();
let toastTimer;

function backend() {
  return window.go?.main?.App;
}

async function invoke(method, ...args) {
  const api = backend();
  if (!api?.[method]) throw new Error(`Entropy desktop backend does not expose ${method}`);
  return api[method](...args);
}

function icon(name) {
  const element = document.createElement("i");
  element.dataset.lucide = name;
  return element;
}

function setText(id, value) {
  const element = $(id);
  if (element) element.textContent = value ?? "--";
}

function asArray(value) {
  return Array.isArray(value) ? value : [];
}

function asNumber(value, fallback = 0) {
  const number = Number(value);
  return Number.isFinite(number) ? number : fallback;
}

function shortHash(value, length = 12) {
  const text = String(value || "");
  if (!text) return "--";
  return text.length <= length ? text : `${text.slice(0, length)}...`;
}

function formatAmount(value) {
  const text = String(value ?? "0");
  const [whole, fraction] = text.split(".");
  try {
    const grouped = BigInt(whole || "0").toLocaleString("en-US");
    return fraction === undefined ? grouped : `${grouped}.${fraction}`;
  } catch {
    return text;
  }
}

function formatTimestamp(timestamp) {
  const seconds = asNumber(timestamp, 0);
  if (seconds <= 0) return "--";
  const date = new Date(seconds * 1000);
  if (Number.isNaN(date.getTime())) return "--";
  return new Intl.DateTimeFormat("en-GB", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(date);
}

function formatBytes(value) {
  let bytes = asNumber(value, -1);
  if (bytes < 0) return "--";
  if (bytes < 1024) return `${bytes.toLocaleString()} B`;
  const units = ["KiB", "MiB", "GiB", "TiB"];
  let unit = -1;
  do {
    bytes /= 1024;
    unit += 1;
  } while (bytes >= 1024 && unit < units.length - 1);
  return `${bytes.toFixed(bytes >= 100 ? 0 : bytes >= 10 ? 1 : 2)} ${units[unit]}`;
}

function errorMessage(error) {
  if (typeof error === "string") return error;
  if (error?.message) return error.message;
  return "The operation failed";
}

function showToast(message, type = "success") {
  clearTimeout(toastTimer);
  const toast = $("toast");
  toast.textContent = String(message || "Done");
  toast.className = `toast visible ${type}`;
  toastTimer = setTimeout(() => {
    toast.className = "toast";
  }, 3600);
}

function setButtonBusy(button, busy, busyLabel = "Working") {
  if (!button) return;
  const label = button.querySelector("span");
  if (busy) {
    button.dataset.originalLabel = label?.textContent || "";
    button.dataset.busyLabel = busyLabel;
    if (label) label.textContent = busyLabel;
    button.classList.add("button-busy");
    button.disabled = true;
  } else {
    if (label && button.dataset.originalLabel && label.textContent === button.dataset.busyLabel) {
      label.textContent = button.dataset.originalLabel;
    }
    delete button.dataset.originalLabel;
    delete button.dataset.busyLabel;
    button.classList.remove("button-busy");
    button.disabled = false;
  }
}

function passwordBytes(password) {
  return encoder.encode(password).byteLength;
}

function validatePassword(password) {
  const bytes = passwordBytes(password);
  if (bytes < 12) throw new Error("Password must contain at least 12 UTF-8 bytes");
  if (bytes > 1024) throw new Error("Password must not exceed 1024 UTF-8 bytes");
}

function updatePasswordCount(inputID, outputID) {
  const bytes = passwordBytes($(inputID).value);
  const output = $(outputID);
  output.textContent = `${bytes.toLocaleString()} bytes`;
  output.classList.toggle("invalid", bytes > 0 && (bytes < 12 || bytes > 1024));
}

function isAmount(value, allowZero) {
  if (!/^(?:0|[1-9]\d*)(?:\.\d{1,8})?$/.test(value)) return false;
  return allowZero || !/^0(?:\.0+)?$/.test(value);
}

async function copyText(value, successMessage) {
  if (!value) {
    showToast("Nothing is available to copy", "error");
    return;
  }
  try {
    await navigator.clipboard.writeText(value);
    showToast(successMessage);
  } catch (error) {
    showToast(`Clipboard access failed: ${errorMessage(error)}`, "error");
  }
}

function renderBlocks(blocks) {
  const body = $("blocks-body");
  body.replaceChildren();
  if (blocks.length === 0) {
    const row = body.insertRow();
    const cell = row.insertCell();
    cell.colSpan = 5;
    cell.className = "empty-cell";
    cell.textContent = "No blocks are available";
    return;
  }

  for (const block of blocks) {
    const row = body.insertRow();
    const height = row.insertCell();
    const heightValue = document.createElement("strong");
    heightValue.textContent = `#${asNumber(block.height).toLocaleString()}`;
    height.append(heightValue);

    const hash = row.insertCell();
    const hashValue = document.createElement("code");
    hashValue.textContent = shortHash(block.hash);
    hashValue.title = String(block.hash || "");
    hash.append(hashValue);

    row.insertCell().textContent = formatTimestamp(block.timestamp);
    row.insertCell().textContent = asNumber(block.transactions).toLocaleString();
    row.insertCell().textContent = asNumber(block.difficulty).toLocaleString();
  }
}

function peerMeta(peer) {
  const parts = [peer.online && peer.active_outbound ? "Online" : peer.active_outbound ? "Connecting" : "Standby"];
  if (asNumber(peer.height) > 0) parts.push(`height #${asNumber(peer.height).toLocaleString()}`);
  parts.push(peer.bootstrap ? "Bootstrap" : peer.discovered ? peer.public ? "Public discovery" : "Local discovery" : "Manual");
  if (peer.active_outbound) parts.push("Outbound");
  if (asNumber(peer.failures) > 0) parts.push(`${asNumber(peer.failures)} failures`);
  return parts.join(" | ");
}

function renderPeers(peers) {
  const list = $("peer-list");
  list.replaceChildren();
  if (peers.length === 0) {
    const empty = document.createElement("li");
    empty.className = "empty-row";
    empty.textContent = "No peers configured or discovered";
    list.append(empty);
    return;
  }

  for (const peer of peers) {
    const row = document.createElement("li");
    if (peer.last_error) row.title = String(peer.last_error);

    const dot = document.createElement("span");
    dot.className = `peer-dot${peer.online && peer.active_outbound ? "" : " offline"}`;

    const main = document.createElement("div");
    main.className = "peer-main";
    const url = document.createElement("code");
    url.textContent = String(peer.url || "Unknown peer");
    const meta = document.createElement("span");
    meta.textContent = peerMeta(peer);
    main.append(url, meta);

    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "icon-button remove-peer";
    remove.title = `Remove ${peer.url || "peer"}`;
    remove.setAttribute("aria-label", `Remove ${peer.url || "peer"}`);
    remove.append(icon("trash-2"));
    remove.addEventListener("click", () => removePeer(String(peer.url || ""), remove));

    row.append(dot, main);
    if (!peer.bootstrap) row.append(remove);
    list.append(row);
  }
}

function renderMining(data) {
  const toggle = $("toggle-mining");
  toggle.replaceChildren(icon(data.mining ? "square" : "play"), document.createElement("span"));
  toggle.querySelector("span").textContent = data.mining ? "Stop mining" : "Start mining";
  toggle.classList.toggle("danger", Boolean(data.mining));
  $("mine-once").disabled = Boolean(data.mining);
  $("mining-indicator").classList.toggle("active", Boolean(data.mining));
  setText("mining-label", data.mining ? "Mining" : "Stopped");
}

function renderSync(data) {
  const localHeight = asNumber(data.height);
  const reportedBest = asNumber(data.best_peer_height);
  const bestHeight = Math.max(localHeight, reportedBest);
  setText("best-peer-height", bestHeight.toLocaleString());

  const progress = $("sync-progress");
  const syncIcon = $("sync-icon");
  progress.classList.remove("indeterminate");
  syncIcon.classList.toggle("active", Boolean(data.syncing));

  const onlinePeers = asNumber(data.peer_count);
  if (data.bootstrap_enabled && onlinePeers === 0) {
    setText("sync-label", "Connecting to the network");
    setText("sync-detail", data.bootstrap_error ? "Public seeds are being retried" : "Discovering public peers");
    progress.style.width = "35%";
    progress.classList.add("indeterminate");
    syncIcon.classList.add("active");
  } else if (data.syncing) {
    setText("sync-label", "Synchronizing chain");
    setText("sync-detail", reportedBest > localHeight
      ? `Local #${localHeight.toLocaleString()} of #${reportedBest.toLocaleString()}`
      : "Validating peer data");
    if (reportedBest > 0) {
      progress.style.width = `${Math.min(100, (localHeight / reportedBest) * 100)}%`;
    } else {
      progress.style.width = "35%";
      progress.classList.add("indeterminate");
    }
  } else if (reportedBest > localHeight) {
    setText("sync-label", "Peer chain is ahead");
    setText("sync-detail", `Local #${localHeight.toLocaleString()} | peer #${reportedBest.toLocaleString()}`);
    progress.style.width = `${Math.min(100, (localHeight / reportedBest) * 100)}%`;
  } else {
    setText("sync-label", "Chain synchronized");
    setText("sync-detail", `Validated through block #${localHeight.toLocaleString()}`);
    progress.style.width = "100%";
  }

  setText("network-sync", onlinePeers === 0 ? "Waiting for peers" : data.syncing ? `Syncing ${localHeight.toLocaleString()} / ${bestHeight.toLocaleString()}` : "Synchronized");
}

function renderDashboard(data) {
  state.dashboard = data;
  const peers = asArray(data.peers);
  const blocks = asArray(data.recent_blocks);
  const localHeight = asNumber(data.height);

  setText("confirmed-balance", formatAmount(data.confirmed_balance));
  setText("spendable-balance", formatAmount(data.spendable_balance));
  setText("wallet-address", data.address || "Unavailable");
  setText("wallet-page-address", data.address || "Unavailable");
  setText("height", localHeight.toLocaleString());
  setText("peer-count", asNumber(data.peer_count).toLocaleString());
  setText("pending-count", asNumber(data.pending_count).toLocaleString());
  setText("difficulty", asNumber(data.difficulty).toLocaleString());
  setText("target-seconds", asNumber(data.target_block_seconds).toLocaleString());
  setText("listen-address", data.listen_address || "Not listening");
  setText("issued", formatAmount(data.issued));
  setText("max-supply", formatAmount(data.max_supply));
  setText("next-reward", formatAmount(data.next_subsidy));
  setText("tip-hash", shortHash(data.tip_hash, 18));
  $("tip-hash").title = String(data.tip_hash || "");
  setText("peer-badge", `${asNumber(data.peer_count)}/${asNumber(data.configured_peer_count)}`);

  const issued = asNumber(data.issued);
  const maximum = asNumber(data.max_supply);
  const supplyPercent = maximum > 0 ? Math.min(100, (issued / maximum) * 100) : 0;
  $("supply-progress").style.width = `${Math.max(supplyPercent, localHeight > 0 ? 0.35 : 0)}%`;

  const statusDot = $("status-dot");
  statusDot.classList.toggle("error", Boolean(data.last_error));
  statusDot.classList.toggle("syncing", !data.last_error && Boolean(data.syncing));
  setText("node-state-label", data.last_error ? "Node warning" : asNumber(data.peer_count) === 0 && data.bootstrap_enabled ? "Finding network" : data.syncing ? "Synchronizing" : "Node active");

  renderSync(data);
  renderMining(data);
  renderBlocks(blocks);
  renderPeers(peers);

  setText("network-protocol", data.protocol || "Unknown");
  const bootstrapState = !data.bootstrap_enabled
    ? "Disabled"
    : asNumber(data.peer_count) > 0
      ? "Connected"
      : data.bootstrap_ready
        ? "Seeds loaded"
        : data.bootstrap_error
          ? "Retrying"
          : "Discovering";
  setText("network-bootstrap", bootstrapState);
  $("network-bootstrap").title = String(data.bootstrap_error || "");
  setText("network-listen", data.listen_address || "Not listening");
  setText("database-path", data.database_path || "Unavailable");
  $("database-path").title = String(data.database_path || "");
  setText("database-size", formatBytes(data.database_bytes));
  const storageMode = $("storage-mode");
  const pruneDepth = asNumber(data.prune_depth);
  const prunedThrough = asNumber(data.pruned_through);
  if (pruneDepth === 0 && prunedThrough === 0) {
    storageMode.textContent = "Archive";
    storageMode.classList.remove("pruned");
  } else if (pruneDepth === 0) {
    storageMode.textContent = "Archive going forward / previously pruned";
    storageMode.classList.add("pruned");
  } else {
    storageMode.textContent = `Pruned | keep ${pruneDepth.toLocaleString()}`;
    storageMode.classList.add("pruned");
  }
  setText("prune-depth", pruneDepth > 0 ? `${pruneDepth.toLocaleString()} recent blocks` : "No future pruning");
  setText("pruned-through", prunedThrough > 0
    ? `Block #${prunedThrough.toLocaleString()}`
    : pruneDepth > 0 ? "Retention enabled; no eligible blocks yet" : "Not pruned");
  setText("diagnostic-protocol", data.protocol || "Unknown");
  setText("diagnostic-listen", data.listen_address || "Not listening");
  setText("diagnostic-tip", data.tip_hash ? `#${localHeight.toLocaleString()} ${shortHash(data.tip_hash, 20)}` : "Unavailable");
  $("diagnostic-tip").title = String(data.tip_hash || "");
  setText("emission-blocks", `${asNumber(data.emission_blocks).toLocaleString()} blocks at ${asNumber(data.target_block_seconds).toLocaleString()} seconds`);
  setText("last-error", data.last_error || "None");
  $("last-error").classList.toggle("error-text", Boolean(data.last_error));

  const health = $("health-label");
  health.classList.toggle("error", Boolean(data.last_error));
  health.querySelector("span").textContent = data.last_error ? "Warning" : "Healthy";

  $("backup-alert").hidden = !data.wallet_needs_backup;
  const security = $("wallet-security-state");
  security.classList.toggle("warning", Boolean(data.wallet_needs_backup));
  security.querySelector("span").textContent = data.wallet_needs_backup ? "Backup needed" : "Recovery secured";

  $("open-restore-backup").disabled = Boolean(data.mining);
  $("open-restore-phrase").disabled = Boolean(data.mining);
  $("open-restore-backup").title = data.mining ? "Stop mining before restoring" : "Restore encrypted backup";
  $("open-restore-phrase").title = data.mining ? "Stop mining before restoring" : "Restore recovery phrase";
  $("switch-archive").disabled = pruneDepth === 0;
  $("switch-archive").title = pruneDepth === 0
    ? prunedThrough > 0 ? "Archive policy is active; previously deleted data remains unavailable" : "Archive policy is already active"
    : "Stop future pruning; previously deleted data will not be restored";

  createIcons({ icons: appIcons });
}

function transactionType(transaction) {
  if (transaction.coinbase) return { label: "Mining reward", className: "mined", icon: "pickaxe" };
  if (!/^0(?:\.0+)?$/.test(String(transaction.sent ?? "0"))) {
    return { label: "Sent transaction", className: "sent", icon: "arrow-up-right" };
  }
  return { label: "Received transaction", className: "received", icon: "arrow-down-left" };
}

function amountCell(value, className) {
  const cell = document.createElement("td");
  const amount = String(value ?? "0");
  if (/^0(?:\.0+)?$/.test(amount)) {
    cell.className = "zero-amount";
    cell.textContent = "--";
  } else {
    cell.className = className;
    cell.textContent = `${formatAmount(amount)} ENT`;
  }
  return cell;
}

function renderHistory(transactions) {
  const body = $("history-body");
  body.replaceChildren();
  setText("history-count", transactions.length.toLocaleString());

  if (transactions.length === 0) {
    const row = body.insertRow();
    const cell = row.insertCell();
    cell.colSpan = 5;
    cell.className = "empty-cell";
    cell.textContent = "No wallet transactions yet";
    return;
  }

  for (const transaction of transactions) {
    const row = body.insertRow();
    row.insertCell().textContent = formatTimestamp(transaction.timestamp);

    const identityCell = row.insertCell();
    const identity = document.createElement("div");
    identity.className = "tx-identity";
    const type = transactionType(transaction);
    const direction = document.createElement("span");
    direction.className = `tx-direction ${type.className}`;
    direction.append(icon(type.icon));
    const details = document.createElement("div");
    const title = document.createElement("strong");
    title.textContent = type.label;
    const id = document.createElement("code");
    id.textContent = shortHash(transaction.id, 22);
    id.title = String(transaction.id || "");
    details.append(title, id);
    identity.append(direction, details);
    identityCell.append(identity);

    row.append(amountCell(transaction.received, "tx-received"));
    row.append(amountCell(transaction.sent, "tx-sent"));

    const statusCell = document.createElement("td");
    const badge = document.createElement("span");
    if (transaction.pending) {
      badge.className = "status-badge pending";
      badge.append(icon("clock-3"), document.createTextNode("Pending"));
    } else {
      const confirmations = asNumber(transaction.confirmations);
      badge.className = "status-badge confirmed";
      badge.append(icon("circle-check"), document.createTextNode(`${confirmations.toLocaleString()} confirmation${confirmations === 1 ? "" : "s"}`));
      if (transaction.block_height != null) badge.title = `Block #${asNumber(transaction.block_height).toLocaleString()}`;
    }
    statusCell.append(badge);
    row.append(statusCell);
  }

  createIcons({ icons: appIcons });
}

async function refreshDashboard() {
  if (!state.ready || state.dashboardRefreshing) return;
  state.dashboardRefreshing = true;
  try {
    const dashboard = await invoke("GetDashboard");
    renderDashboard(dashboard);
    state.lastDashboardError = "";
  } catch (error) {
    const message = errorMessage(error);
    $("status-dot").classList.add("error");
    setText("node-state-label", "Node offline");
    if (message !== state.lastDashboardError) showToast(message, "error");
    state.lastDashboardError = message;
  } finally {
    state.dashboardRefreshing = false;
  }
}

async function refreshHistory(force = false) {
  if (!state.ready || state.historyRefreshing) return;
  state.historyRefreshing = true;
  const button = $("refresh-history");
  if (force) {
    button.disabled = true;
    button.classList.add("button-busy");
  }
  try {
    const history = asArray(await invoke("GetTransactionHistory", 100));
    state.history = history;
    state.lastHistoryRefresh = Date.now();
    renderHistory(history);
    setText("history-updated", `Updated ${new Intl.DateTimeFormat("en-GB", { hour: "2-digit", minute: "2-digit", second: "2-digit" }).format(new Date())}`);
  } catch (error) {
    if (force || state.history.length === 0) showToast(errorMessage(error), "error");
    if (state.history.length === 0) {
      const body = $("history-body");
      body.replaceChildren();
      const row = body.insertRow();
      const cell = row.insertCell();
      cell.colSpan = 5;
      cell.className = "empty-cell";
      cell.textContent = "Transaction history is unavailable";
    }
  } finally {
    state.historyRefreshing = false;
    if (force) {
      button.disabled = false;
      button.classList.remove("button-busy");
    }
  }
}

function showStartupSection(section) {
  $("startup-loading").hidden = section !== "loading";
  $("migration-form").hidden = section !== "migration";
  $("startup-failed").hidden = section !== "failed";
}

async function checkStartup() {
  if (state.startupChecking || state.ready) return;
  state.startupChecking = true;
  try {
    const startup = await invoke("GetStartupState");
    state.startupCode = startup.code || "starting";
    setText("startup-message", startup.message || "Opening wallet and ledger...");

    if (startup.ready || startup.code === "ready") {
      state.ready = true;
      $("startup-overlay").classList.remove("visible");
      await Promise.all([refreshDashboard(), refreshHistory()]);
      return;
    }
    $("startup-overlay").classList.add("visible");
    if (startup.code === "wallet_migration_required") {
      showStartupSection("migration");
    } else if (startup.code === "startup_failed") {
      setText("startup-error", startup.message || "Unknown startup error");
      showStartupSection("failed");
    } else {
      showStartupSection("loading");
    }
  } catch (error) {
    state.startupCode = "startup_failed";
    setText("startup-error", errorMessage(error));
    showStartupSection("failed");
    $("startup-overlay").classList.add("visible");
  } finally {
    state.startupChecking = false;
  }
}

function activateView(name) {
  for (const tab of document.querySelectorAll("[data-view]")) {
    tab.classList.toggle("active", tab.dataset.view === name);
  }
  for (const panel of document.querySelectorAll("[data-view-panel]")) {
    const active = panel.dataset.viewPanel === name;
    panel.hidden = !active;
    panel.classList.toggle("active", active);
  }
  if (name === "transactions") refreshHistory(true);
}

function clearSensitive(dialog) {
  for (const input of dialog.querySelectorAll('input[type="password"], textarea')) input.value = "";
  for (const input of dialog.querySelectorAll('input[type="checkbox"]')) input.checked = false;
  if (dialog.id === "recovery-dialog") {
    state.recoveryPhrase = "";
    $("recovery-grid").replaceChildren();
    $("recovery-placeholder").hidden = false;
    $("recovery-content").hidden = true;
    $("confirm-recovery").disabled = true;
  }
  if (dialog.id === "prune-dialog") {
    state.pendingPruneRetain = 0;
    $("confirm-prune").disabled = true;
  }
  updateAllSensitiveCounters();
}

function openDialog(id) {
  const dialog = $(id);
  clearSensitive(dialog);
  if (!dialog.open) dialog.showModal();
}

function closeDialog(id) {
  const dialog = $(id);
  if (dialog.open) dialog.close();
}

function updateAllSensitiveCounters() {
  updatePasswordCount("migration-password", "migration-password-bytes");
  updatePasswordCount("export-password", "export-password-bytes");
  updatePasswordCount("restore-backup-password", "restore-password-bytes");
  const words = $("restore-phrase").value.trim().split(/\s+/).filter(Boolean).length;
  setText("restore-word-count", `${words} / 24 words`);
}

async function removePeer(peer, button) {
  if (!peer) return;
  setButtonBusy(button, true, "Removing");
  try {
    const result = await invoke("RemovePeer", peer);
    showToast(result.message || "Peer removed");
    await refreshDashboard();
  } catch (error) {
    showToast(errorMessage(error), "error");
    setButtonBusy(button, false);
  }
}

document.querySelectorAll("[data-view]").forEach((tab) => {
  tab.addEventListener("click", () => activateView(tab.dataset.view));
});

document.querySelectorAll("[data-close-dialog]").forEach((button) => {
  button.addEventListener("click", () => closeDialog(button.dataset.closeDialog));
});

document.querySelectorAll("dialog").forEach((dialog) => {
  dialog.addEventListener("click", (event) => {
    if (event.target === dialog) dialog.close();
  });
  dialog.addEventListener("close", () => clearSensitive(dialog));
});

$("copy-address").addEventListener("click", () => copyText(state.dashboard?.address, "Address copied"));
$("copy-wallet-page-address").addEventListener("click", () => copyText(state.dashboard?.address, "Address copied"));
$("copy-database-path").addEventListener("click", () => copyText(state.dashboard?.database_path, "Database path copied"));

$("send-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const button = form.querySelector('button[type="submit"]');
  const to = $("send-to").value.trim();
  const amount = $("send-amount").value.trim();
  const fee = $("send-fee").value.trim();
  if (!to) {
    showToast("Recipient address is required", "error");
    return;
  }
  if (!isAmount(amount, false)) {
    showToast("Amount must be positive with no more than 8 decimal places", "error");
    return;
  }
  if (!isAmount(fee, true)) {
    showToast("Fee must be zero or positive with no more than 8 decimal places", "error");
    return;
  }

  setButtonBusy(button, true, "Broadcasting");
  try {
    const result = await invoke("SendTransaction", to, amount, fee);
    showToast(`${result.message || "Transaction submitted"}: ${shortHash(result.id)}`);
    $("send-amount").value = "";
    await Promise.all([refreshDashboard(), refreshHistory()]);
  } catch (error) {
    showToast(errorMessage(error), "error");
  } finally {
    setButtonBusy(button, false);
  }
});

$("toggle-mining").addEventListener("click", async () => {
  if (!state.dashboard) return;
  const button = $("toggle-mining");
  const enabling = !state.dashboard.mining;
  setButtonBusy(button, true, enabling ? "Starting" : "Stopping");
  try {
    const result = await invoke("SetMining", enabling);
    showToast(result.message || (enabling ? "Mining started" : "Mining stopping"));
    await refreshDashboard();
  } catch (error) {
    showToast(errorMessage(error), "error");
  } finally {
    setButtonBusy(button, false);
  }
});

$("mine-once").addEventListener("click", async () => {
  const button = $("mine-once");
  setButtonBusy(button, true, "Mining block");
  try {
    const result = await invoke("MineOneBlock");
    showToast(`${result.message || "Block mined"}: ${shortHash(result.id)}`);
    await Promise.all([refreshDashboard(), refreshHistory()]);
  } catch (error) {
    showToast(errorMessage(error), "error");
  } finally {
    setButtonBusy(button, false);
  }
});

$("peer-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const input = $("peer-url");
  const button = form.querySelector('button[type="submit"]');
  const peer = input.value.trim();
  if (!peer) return;
  button.disabled = true;
  try {
    const result = await invoke("AddPeer", peer);
    showToast(result.message || "Peer added");
    input.value = "";
    await refreshDashboard();
  } catch (error) {
    showToast(errorMessage(error), "error");
  } finally {
    button.disabled = false;
  }
});

$("refresh-history").addEventListener("click", () => refreshHistory(true));
$("backup-alert-action").addEventListener("click", () => openDialog("recovery-dialog"));
$("open-recovery").addEventListener("click", () => openDialog("recovery-dialog"));
$("open-export").addEventListener("click", () => openDialog("export-dialog"));
$("open-restore-backup").addEventListener("click", () => openDialog("restore-backup-dialog"));
$("open-restore-phrase").addEventListener("click", () => openDialog("restore-phrase-dialog"));

$("prune-form").addEventListener("submit", (event) => {
  event.preventDefault();
  const text = $("prune-retain").value.trim();
  if (!/^[1-9]\d*$/.test(text)) {
    showToast("Retention must be a whole number", "error");
    return;
  }
  const retain = Number(text);
  if (!Number.isSafeInteger(retain) || retain < 120 || retain > 31536000) {
    showToast("Retention must be between 120 and 31,536,000 blocks", "error");
    return;
  }

  const height = asNumber(state.dashboard?.height);
  const currentHorizon = asNumber(state.dashboard?.pruned_through);
  const requestedHorizon = Math.max(0, height - retain);
  state.pendingPruneRetain = retain;
  if (currentHorizon > requestedHorizon) {
    setText("prune-impact", `The ledger is already pruned through block #${currentHorizon.toLocaleString()}. Deleted data will not be restored; future pruning will retain the newest ${retain.toLocaleString()} blocks.`);
  } else if (requestedHorizon === 0) {
    setText("prune-impact", `Current height is #${height.toLocaleString()}. No existing body is eligible yet; future pruning will retain the newest ${retain.toLocaleString()} blocks.`);
  } else {
    setText("prune-impact", `Block and transaction bodies plus undo records through block #${requestedHorizon.toLocaleString()} will be permanently removed. The newest ${retain.toLocaleString()} blocks remain complete.`);
  }
  openDialog("prune-dialog");
  state.pendingPruneRetain = retain;
});

$("prune-confirm-check").addEventListener("change", (event) => {
  $("confirm-prune").disabled = !event.currentTarget.checked;
});

$("switch-archive").addEventListener("click", async () => {
  const button = $("switch-archive");
  setButtonBusy(button, true, "Switching");
  try {
    const result = await invoke("PruneLedger", 0);
    showToast(`${result.message || "Archive policy enabled"}. Previously pruned data remains unavailable.`, "info");
    await refreshDashboard();
  } catch (error) {
    showToast(errorMessage(error), "error");
  } finally {
    setButtonBusy(button, false);
    if (asNumber(state.dashboard?.prune_depth) === 0) button.disabled = true;
  }
});

$("confirm-prune").addEventListener("click", async () => {
  const button = $("confirm-prune");
  const retain = state.pendingPruneRetain;
  if (!retain || !$("prune-confirm-check").checked) return;
  setButtonBusy(button, true, "Pruning");
  try {
    const result = await invoke("PruneLedger", retain);
    showToast(result.message || "Ledger pruning completed");
    closeDialog("prune-dialog");
    await refreshDashboard();
  } catch (error) {
    showToast(errorMessage(error), "error");
    setButtonBusy(button, false);
  }
});

$("reveal-recovery").addEventListener("click", async () => {
  const button = $("reveal-recovery");
  setButtonBusy(button, true, "Decrypting");
  try {
    const phrase = String(await invoke("GetRecoveryPhrase"));
    const words = phrase.trim().split(/\s+/).filter(Boolean);
    if (words.length !== 24) throw new Error("Backend returned an invalid recovery phrase");
    state.recoveryPhrase = words.join(" ");
    const grid = $("recovery-grid");
    grid.replaceChildren();
    words.forEach((word, index) => {
      const item = document.createElement("li");
      const number = document.createElement("span");
      number.textContent = String(index + 1);
      const value = document.createElement("code");
      value.textContent = word;
      item.append(number, value);
      grid.append(item);
    });
    $("recovery-placeholder").hidden = true;
    $("recovery-content").hidden = false;
  } catch (error) {
    const message = errorMessage(error);
    setText("recovery-action-detail", message);
    showToast(message, "error");
  } finally {
    setButtonBusy(button, false);
  }
});

$("copy-recovery").addEventListener("click", () => copyText(state.recoveryPhrase, "Recovery phrase copied"));
$("recovery-confirm-check").addEventListener("change", (event) => {
  $("confirm-recovery").disabled = !event.currentTarget.checked;
});
$("confirm-recovery").addEventListener("click", async () => {
  const button = $("confirm-recovery");
  if (!state.recoveryPhrase || !$("recovery-confirm-check").checked) return;
  setButtonBusy(button, true, "Saving");
  try {
    const result = await invoke("ConfirmWalletRecovery");
    showToast(result.message || "Wallet recovery confirmed");
    closeDialog("recovery-dialog");
    await refreshDashboard();
  } catch (error) {
    showToast(errorMessage(error), "error");
    setButtonBusy(button, false);
  }
});

$("export-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const button = form.querySelector('button[type="submit"]');
  const password = $("export-password").value;
  const confirmation = $("export-confirm").value;
  try {
    validatePassword(password);
    if (password !== confirmation) throw new Error("Password confirmation does not match");
  } catch (error) {
    showToast(errorMessage(error), "error");
    return;
  }
  setButtonBusy(button, true, "Encrypting");
  try {
    const result = await invoke("ExportWalletBackup", password);
    showToast(result.message || "Wallet backup exported", result.message?.toLowerCase().includes("cancel") ? "info" : "success");
    closeDialog("export-dialog");
    await refreshDashboard();
  } catch (error) {
    showToast(errorMessage(error), "error");
  } finally {
    $("export-password").value = "";
    $("export-confirm").value = "";
    setButtonBusy(button, false);
  }
});

$("restore-backup-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const button = form.querySelector('button[type="submit"]');
  const password = $("restore-backup-password").value;
  try {
    validatePassword(password);
    if (!$("restore-backup-check").checked) throw new Error("Confirm active wallet replacement");
  } catch (error) {
    showToast(errorMessage(error), "error");
    return;
  }
  setButtonBusy(button, true, "Restoring");
  try {
    const result = await invoke("RestoreWalletBackup", password);
    showToast(result.id ? `${result.message}: ${shortHash(result.id, 18)}` : result.message, result.message?.toLowerCase().includes("cancel") ? "info" : "success");
    closeDialog("restore-backup-dialog");
    state.history = [];
    await Promise.all([refreshDashboard(), refreshHistory()]);
  } catch (error) {
    showToast(errorMessage(error), "error");
  } finally {
    $("restore-backup-password").value = "";
    setButtonBusy(button, false);
  }
});

$("restore-phrase-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const button = form.querySelector('button[type="submit"]');
  const words = $("restore-phrase").value.trim().split(/\s+/).filter(Boolean);
  if (words.length !== 24) {
    showToast("Recovery phrase must contain exactly 24 words", "error");
    return;
  }
  if (!$("restore-phrase-check").checked) {
    showToast("Confirm active wallet replacement", "error");
    return;
  }
  const phrase = words.join(" ");
  setButtonBusy(button, true, "Restoring");
  try {
    const result = await invoke("RestoreWalletMnemonic", phrase);
    showToast(`${result.message || "Wallet restored"}: ${shortHash(result.id, 18)}`);
    closeDialog("restore-phrase-dialog");
    state.history = [];
    await Promise.all([refreshDashboard(), refreshHistory()]);
  } catch (error) {
    showToast(errorMessage(error), "error");
  } finally {
    $("restore-phrase").value = "";
    setButtonBusy(button, false);
  }
});

$("migration-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const button = form.querySelector('button[type="submit"]');
  const password = $("migration-password").value;
  const confirmation = $("migration-confirm").value;
  try {
    validatePassword(password);
    if (password !== confirmation) throw new Error("Password confirmation does not match");
  } catch (error) {
    showToast(errorMessage(error), "error");
    return;
  }
  setButtonBusy(button, true, "Encrypting");
  try {
    const result = await invoke("MigrateLegacyWallet", password);
    const cancelled = result.message?.toLowerCase().includes("cancel");
    showToast(result.message || "Legacy wallet migrated", cancelled ? "info" : "success");
    if (!cancelled) {
      state.startupCode = "starting";
      showStartupSection("loading");
      await checkStartup();
    }
  } catch (error) {
    showToast(errorMessage(error), "error");
  } finally {
    $("migration-password").value = "";
    $("migration-confirm").value = "";
    updateAllSensitiveCounters();
    setButtonBusy(button, false);
  }
});

$("retry-startup").addEventListener("click", async () => {
  state.startupCode = "starting";
  showStartupSection("loading");
  try {
    await invoke("RetryStartup");
  } catch (error) {
    showToast(errorMessage(error), "error");
  }
  await checkStartup();
});

for (const input of [$("migration-password"), $("export-password"), $("restore-backup-password")]) {
  input.addEventListener("input", updateAllSensitiveCounters);
}
$("restore-phrase").addEventListener("input", updateAllSensitiveCounters);

window.addEventListener("beforeunload", () => {
  document.querySelectorAll('input[type="password"], textarea').forEach((input) => { input.value = ""; });
  state.recoveryPhrase = "";
});

async function heartbeat() {
  if (!state.ready) {
    if (state.startupCode === "starting") await checkStartup();
    return;
  }
  await refreshDashboard();
  if (Date.now() - state.lastHistoryRefresh >= 5000) await refreshHistory();
}

createIcons({ icons: appIcons });
updateAllSensitiveCounters();
checkStartup();
setInterval(heartbeat, 1200);
