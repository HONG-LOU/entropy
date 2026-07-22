import "./style.css";
import QRCode from "qrcode";
import QrScanner from "qr-scanner";
import {
  ArrowDownLeft, ArrowUpRight, ChevronRight, CircleDot, Copy, Download, KeyRound, Languages, Lock,
  LockKeyhole, Plus, ReceiptText, RefreshCw, ScanLine, ShieldCheck,
  TriangleAlert, Unlock, X, createIcons,
} from "lucide";
import { detectLanguage, saveLanguage, translate } from "./i18n.js";
import { loadVault, saveVault } from "./storage.js";
import { walletCall } from "./vault-client.js";
import { broadcastTransaction, loadWalletSnapshot } from "./api.js";
import { formatAmount, parseAmount } from "./amount.js";

createIcons({ icons: { ArrowDownLeft, ArrowUpRight, ChevronRight, CircleDot, Copy, Download, KeyRound, Languages, Lock, LockKeyhole, Plus, ReceiptText, RefreshCw, ScanLine, ShieldCheck, TriangleAlert, Unlock, X } });

const $ = (selector) => document.querySelector(selector);
const views = ["#loadingView", "#onboardingView", "#unlockView", "#walletView"];
let vaultRecord;
let snapshot;
let pendingSetup;
let pendingTransaction;
let scanner;
let language = detectLanguage();
let lastSyncTime;
let currentNodeState = { kind: "", key: "connecting", values: {} };
let selectedTransaction;

const tr = (key, values) => translate(language, key, values);
const locale = () => language === "zh" ? "zh-CN" : "en-US";

function applyLanguage() {
  document.documentElement.lang = language === "zh" ? "zh-CN" : "en";
  document.title = tr("documentTitle");
  document.querySelectorAll("[data-i18n]").forEach((element) => { element.textContent = tr(element.dataset.i18n); });
  for (const attribute of ["title", "aria-label", "placeholder"]) {
    document.querySelectorAll(`[data-i18n-${attribute}]`).forEach((element) => {
      element.setAttribute(attribute, tr(element.dataset[`i18n${attribute.split("-").map((part) => part[0].toUpperCase() + part.slice(1)).join("")}`]));
    });
  }
  const button = $("#languageButton");
  button.querySelector("span").textContent = language === "zh" ? "EN" : "中";
  button.title = tr("language");
  button.setAttribute("aria-label", tr("language"));
  updateNodeState();
  renderSnapshot();
  if (pendingSetup) updateSetupHeadings();
  if (selectedTransaction && $("#transactionDialog").open) renderTransactionDetails(selectedTransaction);
}

function switchLanguage() {
  language = language === "zh" ? "en" : "zh";
  saveLanguage(language);
  applyLanguage();
}

function showView(selector) {
  for (const view of views) $(view).hidden = view !== selector;
}

function toast(message, error = false) {
  const element = $("#toast");
  element.textContent = message;
  element.classList.toggle("error", error);
  element.hidden = false;
  clearTimeout(toast.timer);
  toast.timer = setTimeout(() => { element.hidden = true; }, 3200);
}

function setBusy(button, busy) {
  button.disabled = busy;
  button.classList.toggle("busy", busy);
}

function setNodeState(kind, key, values = {}) {
  currentNodeState = { kind, key, values };
  updateNodeState();
}

function updateNodeState() {
  const { kind, key, values } = currentNodeState;
  $("#nodeState").className = `node-state ${kind}`;
  $("#nodeState span").textContent = tr(key, values);
}

async function initialize() {
  try {
    vaultRecord = await loadVault();
    if (vaultRecord) {
      $("#unlockAddress").textContent = vaultRecord.address;
      showView("#unlockView");
    } else {
      showView("#onboardingView");
    }
  } catch (error) {
    showView("#onboardingView");
    toast(tr("readVaultError", { message: error.message }), true);
  }
  registerServiceWorker();
}

function openSetup(mode) {
  pendingSetup = { mode };
  $("#passwordPanel").hidden = false;
  $("#mnemonicPanel").hidden = true;
  updateSetupHeadings();
  $("#mnemonicInputLabel").hidden = mode !== "restore";
  $("#setupPassword").value = "";
  $("#setupPasswordConfirm").value = "";
  $("#mnemonicInput").value = "";
  $("#setupDialog").showModal();
}

function updateSetupHeadings() {
  $("#setupStep").textContent = tr(pendingSetup.mode === "create" ? "createWallet" : "restoreWallet");
  $("#setupTitle").textContent = tr(pendingSetup.result ? "backupWords" : pendingSetup.mode === "create" ? "setPassword" : "restoreInfo");
}

async function continueSetup() {
  const button = $("#setupContinue");
  const password = $("#setupPassword").value;
  if (password.length < 12 || password !== $("#setupPasswordConfirm").value) {
    toast(tr(password.length < 12 ? "passwordTooShort" : "passwordMismatch"), true);
    return;
  }
  setBusy(button, true);
  const payload = pendingSetup.mode === "create"
    ? { action: "create", password }
    : { action: "restore", password, mnemonic: $("#mnemonicInput").value };
  const result = await walletCall(payload);
  setBusy(button, false);
  if (!result.ok) {
    toast(localizeWalletError(result.error), true);
    return;
  }
  pendingSetup = { ...pendingSetup, password, result };
  if (pendingSetup.mode === "restore") {
    await finishSetup();
    return;
  }
  $("#mnemonicWords").replaceChildren(...result.mnemonic.split(" ").map((word) => {
    const item = document.createElement("li");
    item.textContent = word;
    return item;
  }));
  $("#mnemonicConfirmed").checked = false;
  $("#finishCreate").disabled = true;
  $("#passwordPanel").hidden = true;
  $("#mnemonicPanel").hidden = false;
  updateSetupHeadings();
}

async function finishSetup() {
  const { result } = pendingSetup;
  vaultRecord = { address: result.address, backup: result.backup, savedAt: new Date().toISOString() };
  await saveVault(vaultRecord);
  $("#setupDialog").close();
  $("#receiveAddress").textContent = result.address;
  showView("#walletView");
  toast(tr(pendingSetup.mode === "create" ? "walletCreated" : "walletRestored"));
  pendingSetup = undefined;
  await refreshWallet();
}

async function unlock(event) {
  event.preventDefault();
  const button = event.currentTarget.querySelector("button");
  setBusy(button, true);
  const result = await walletCall({ action: "unlock", password: $("#unlockPassword").value, backup: vaultRecord.backup });
  setBusy(button, false);
  $("#unlockPassword").value = "";
  if (!result.ok) {
    toast(localizeWalletError(result.error), true);
    return;
  }
  $("#receiveAddress").textContent = result.address;
  showView("#walletView");
  await refreshWallet();
}

async function lockWallet() {
  await walletCall({ action: "lock" });
  snapshot = undefined;
  $("#unlockAddress").textContent = vaultRecord.address;
  showView("#unlockView");
  toast(tr("walletLocked"));
}

async function refreshWallet() {
  const button = $("#refreshButton");
  setBusy(button, true);
  setNodeState("loading", "syncing");
  try {
    snapshot = await loadWalletSnapshot(vaultRecord.address);
    lastSyncTime = new Date();
    renderSnapshot();
    setNodeState(snapshot.tipsAgree ? "online" : "warning", snapshot.tipsAgree ? "nodesAgree" : "nodesOnline", { count: snapshot.nodesOnline });
    if (snapshot.utxos_truncated) toast(tr("utxosTruncated"), true);
  } catch (error) {
    setNodeState("offline", "nodeOffline");
    toast(localizeError(error.message), true);
  } finally {
    setBusy(button, false);
  }
}

function renderSnapshot() {
  if (!snapshot) return;
  $("#balanceValue").textContent = formatAmount(snapshot.spendable_balance);
  $("#confirmedValue").textContent = tr("confirmedBalance", { amount: formatAmount(snapshot.confirmed_balance) });
  $("#tipValue").textContent = tr("blockHeight", { height: Number(snapshot.height).toLocaleString(locale()) });
  if (lastSyncTime) $("#syncTime").textContent = lastSyncTime.toLocaleTimeString(locale(), { hour: "2-digit", minute: "2-digit" });
  renderHistory(snapshot.history || []);
}

function renderHistory(history) {
  const container = $("#transactions");
  container.replaceChildren(...history.map((record) => {
    const sent = Number(record.sent) > 0;
    const amount = sent ? Math.max(0, Number(record.sent) - Number(record.received)) : Number(record.received);
    const row = document.createElement("button");
    row.type = "button";
    row.className = "transaction-row";
    row.addEventListener("click", () => showTransactionDetails(record));
    const icon = document.createElement("span");
    icon.className = sent ? "tx-icon sent" : "tx-icon received";
    icon.textContent = sent ? "↗" : "↙";
    const details = document.createElement("div");
    const title = document.createElement("strong");
    title.textContent = tr(record.coinbase ? "miningReward" : sent ? "sent" : "received");
    const meta = document.createElement("span");
    meta.textContent = record.pending ? tr("pending") : formatTransactionTime(record.timestamp, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
    details.append(title, meta);
    const value = document.createElement("div");
    value.className = sent ? "tx-value sent" : "tx-value";
    value.textContent = `${sent ? "−" : "+"}${formatAmount(amount)} ENT`;
    const arrow = document.createElement("i");
    arrow.dataset.lucide = "chevron-right";
    arrow.className = "tx-chevron";
    row.append(icon, details, value, arrow);
    return row;
  }));
  createIcons({ icons: { ChevronRight } });
  $("#emptyHistory").hidden = history.length > 0;
}

function formatTransactionTime(timestamp, options = {}) {
  if (!timestamp) return tr("unavailable");
  return new Date(Number(timestamp) * 1000).toLocaleString(locale(), options);
}

function appendDetailRow(container, labelKey, value, mono = false) {
  const row = document.createElement("div");
  row.className = "detail-row";
  const label = document.createElement("dt");
  label.textContent = tr(labelKey);
  const content = document.createElement("dd");
  content.textContent = value ?? tr("unavailable");
  if (mono) content.classList.add("mono");
  row.append(label, content);
  container.append(row);
  return content;
}

function appendTransactionSection(container, titleKey, items, kind) {
  const section = document.createElement("section");
  const title = document.createElement("h3");
  title.textContent = tr(titleKey);
  section.append(title);
  if (!items.length) {
    const empty = document.createElement("p");
    empty.className = "detail-empty";
    empty.textContent = tr(kind === "input" ? "noInputs" : "noOutputs");
    section.append(empty);
  }
  items.forEach((item, index) => {
    const group = document.createElement("dl");
    group.className = "detail-group";
    const heading = document.createElement("strong");
    heading.textContent = kind === "input" ? `${tr("inputs")} ${index + 1}` : tr("output", { index: index + 1 });
    group.append(heading);
    if (kind === "input") {
      appendDetailRow(group, "outpoint", `${item.tx_id}:${item.output_index}`, true);
    } else {
      appendDetailRow(group, "amount", `${formatAmount(item.amount)} ENT`);
      appendDetailRow(group, "address", item.address, true);
    }
    section.append(group);
  });
  container.append(section);
}

function renderTransactionDetails(record) {
  const container = $("#transactionDetails");
  container.replaceChildren();
  const transaction = record.transaction || {};
  const transactionId = transaction.id || record.id || tr("unavailable");
  const sent = Number(record.sent) > 0;
  const summary = document.createElement("dl");
  summary.className = "detail-summary";
  appendDetailRow(summary, "status", `${tr(record.coinbase ? "miningReward" : sent ? "sent" : "received")} · ${tr(record.pending ? "pending" : "confirmed")}`);
  appendDetailRow(summary, "time", formatTransactionTime(record.timestamp, { dateStyle: "medium", timeStyle: "short" }));
  appendDetailRow(summary, "confirmations", tr("confirmationsCount", { count: Number(record.confirmations || 0).toLocaleString(locale()) }));
  const idContent = appendDetailRow(summary, "transactionId", transactionId, true);
  if (transactionId !== tr("unavailable")) {
    const copy = document.createElement("button");
    copy.type = "button";
    copy.className = "detail-copy";
    copy.title = tr("copyTransactionId");
    copy.setAttribute("aria-label", tr("copyTransactionId"));
    copy.innerHTML = '<i data-lucide="copy"></i>';
    copy.addEventListener("click", async () => { await navigator.clipboard.writeText(transactionId); toast(tr("transactionIdCopied")); });
    idContent.append(copy);
  }
  if (record.block_height !== undefined && record.block_height !== null) appendDetailRow(summary, "block", Number(record.block_height).toLocaleString(locale()));
  if (record.block_hash) appendDetailRow(summary, "blockHash", record.block_hash, true);
  if (record.position !== undefined && record.position !== null) appendDetailRow(summary, "position", Number(record.position).toLocaleString(locale()));
  appendDetailRow(summary, "receivedTotal", `${formatAmount(record.received)} ENT`);
  appendDetailRow(summary, "sentTotal", `${formatAmount(record.sent)} ENT`);
  container.append(summary);
  if (record.pruned) {
    const notice = document.createElement("p");
    notice.className = "detail-notice";
    notice.textContent = tr("prunedDetails");
    container.append(notice);
  } else {
    appendTransactionSection(container, "inputs", transaction.inputs || [], "input");
    appendTransactionSection(container, "outputs", transaction.outputs || [], "output");
  }
  createIcons({ icons: { Copy } });
}

function showTransactionDetails(record) {
  selectedTransaction = record;
  renderTransactionDetails(record);
  $("#transactionDialog").showModal();
}

async function showReceive() {
  $("#receiveAddress").textContent = vaultRecord.address;
  await QRCode.toCanvas($("#receiveQR"), vaultRecord.address, { width: 220, margin: 2, color: { dark: "#0b0f12", light: "#fafaf6" } });
  $("#receiveDialog").showModal();
}

function openSend() {
  $("#sendFields").hidden = false;
  $("#sendReview").hidden = true;
  pendingTransaction = undefined;
  $("#sendDialog").showModal();
}

async function reviewSend(event) {
  event.preventDefault();
  if (!snapshot) {
    toast(tr("refreshFirst"), true);
    return;
  }
  try {
    const to = $("#sendAddress").value.trim();
    const valid = await walletCall({ action: "validate_address", to });
    if (!valid.ok) throw new Error(tr("invalidAddress"));
    const amount = parseAmount($("#sendAmount").value);
    const fee = parseAmount($("#sendFee").value);
    pendingTransaction = { to, amount, fee };
    $("#reviewAmount").textContent = formatAmount(amount);
    $("#reviewFee").textContent = formatAmount(fee);
    $("#reviewAddress").textContent = to;
    $("#nodeWarning").hidden = snapshot.tipsAgree;
    $("#confirmSend").disabled = !snapshot.tipsAgree || snapshot.utxos_truncated;
    $("#sendFields").hidden = true;
    $("#sendReview").hidden = false;
  } catch (error) {
    toast(localizeError(error.message), true);
  }
}

async function confirmSend() {
  const button = $("#confirmSend");
  setBusy(button, true);
  try {
    const result = await walletCall({ action: "sign", ...pendingTransaction, utxos: snapshot.utxos });
    if (!result.ok) throw new Error(localizeWalletError(result.error));
    const accepted = await broadcastTransaction(result.transaction);
    $("#sendDialog").close();
    $("#sendForm").reset();
    $("#sendFee").value = "0.00001";
    toast(tr("broadcasted", { id: accepted.transaction_id.slice(0, 10) }));
    await refreshWallet();
  } catch (error) {
    toast(localizeError(error.message), true);
  } finally {
    setBusy(button, false);
  }
}

async function exportBackup() {
  const button = $("#exportBackup");
  setBusy(button, true);
  const result = await walletCall({ action: "export", password: $("#backupPassword").value });
  setBusy(button, false);
  $("#backupPassword").value = "";
  if (!result.ok) {
    toast(localizeWalletError(result.error), true);
    return;
  }
  const bytes = Uint8Array.from(atob(result.backup), (character) => character.charCodeAt(0));
  const link = document.createElement("a");
  link.href = URL.createObjectURL(new Blob([bytes], { type: "application/json" }));
  link.download = `entcoin-${result.address.slice(0, 12)}.entwallet`;
  link.click();
  URL.revokeObjectURL(link.href);
  $("#backupDialog").close();
  toast(tr("backupExported"));
}

async function startScanner() {
  const dialog = $("#scannerDialog");
  dialog.showModal();
  scanner = new QrScanner($("#scannerVideo"), (result) => {
    const value = result.data.replace(/^entcoin:/i, "").split("?")[0];
    $("#sendAddress").value = value;
    stopScanner();
  }, { preferredCamera: "environment", highlightScanRegion: true, returnDetailedScanResult: true });
  try {
    await scanner.start();
  } catch {
    stopScanner();
    toast(tr("cameraError"), true);
  }
}

function stopScanner() {
  scanner?.destroy();
  scanner = undefined;
  $("#scannerDialog").close();
}

function localizeWalletError(message = "") {
  if (message.includes("authentication failed")) return tr("authFailed");
  if (message.includes("password does not meet policy")) return tr("passwordTooShort");
  if (message.includes("invalid 24-word")) return tr("invalidMnemonic");
  if (message.includes("insufficient funds")) return tr("insufficientFunds");
  return localizeError(message) || tr("walletOperationFailed");
}

function localizeError(message = "") {
  if (message.includes("金额格式无效")) return tr("invalidAmount");
  if (message.includes("金额超出允许范围")) return tr("amountOutOfRange");
  if (message.includes("暂时无法连接 Entcoin 节点")) return tr("nodesUnavailable");
  if (message.includes("交易广播失败")) return tr("broadcastFailed");
  const httpStatus = message.match(/节点返回 HTTP (\d+)/)?.[1];
  if (httpStatus) return tr("nodeHttpError", { status: httpStatus });
  return message;
}

async function registerServiceWorker() {
  if ("serviceWorker" in navigator && import.meta.env.PROD) {
    await navigator.serviceWorker.register(`${import.meta.env.BASE_URL}sw.js`, { scope: import.meta.env.BASE_URL }).catch(() => undefined);
  }
}

$("#createStart").addEventListener("click", () => openSetup("create"));
$("#languageButton").addEventListener("click", switchLanguage);
$("#restoreStart").addEventListener("click", () => openSetup("restore"));
$("#setupContinue").addEventListener("click", continueSetup);
$("#mnemonicConfirmed").addEventListener("change", (event) => { $("#finishCreate").disabled = !event.target.checked; });
$("#finishCreate").addEventListener("click", finishSetup);
$("#setupDialog").addEventListener("close", () => { if (pendingSetup?.result) walletCall({ action: "lock" }); pendingSetup = undefined; });
$("#unlockForm").addEventListener("submit", unlock);
$("#refreshButton").addEventListener("click", refreshWallet);
$("#receiveButton").addEventListener("click", showReceive);
$("#sendButton").addEventListener("click", openSend);
$("#backupButton").addEventListener("click", () => $("#backupDialog").showModal());
$("#lockButton").addEventListener("click", lockWallet);
$("#sendForm").addEventListener("submit", reviewSend);
$("#confirmSend").addEventListener("click", confirmSend);
$("#editSend").addEventListener("click", () => { $("#sendFields").hidden = false; $("#sendReview").hidden = true; });
$("#exportBackup").addEventListener("click", exportBackup);
$("#scanButton").addEventListener("click", startScanner);
$("#copyAddress").addEventListener("click", async () => { await navigator.clipboard.writeText(vaultRecord.address); toast(tr("addressCopied")); });
document.querySelectorAll(".dialog-close").forEach((button) => button.addEventListener("click", () => button.closest("dialog").close()));
$("#scannerDialog").addEventListener("close", () => scanner?.destroy());
document.addEventListener("visibilitychange", () => {
  if (document.hidden && !$("#walletView").hidden) {
    clearTimeout(lockWallet.timer);
    lockWallet.timer = setTimeout(lockWallet, 120000);
  } else {
    clearTimeout(lockWallet.timer);
  }
});

applyLanguage();
initialize();
