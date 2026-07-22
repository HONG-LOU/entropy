import "./style.css";
import QRCode from "qrcode";
import QrScanner from "qr-scanner";
import {
  ArrowDownLeft, ArrowUpRight, CircleDot, Copy, Download, KeyRound, Lock,
  LockKeyhole, Plus, ReceiptText, RefreshCw, ScanLine, ShieldCheck,
  TriangleAlert, Unlock, X, createIcons,
} from "lucide";
import { loadVault, saveVault } from "./storage.js";
import { walletCall } from "./vault-client.js";
import { broadcastTransaction, loadWalletSnapshot } from "./api.js";
import { formatAmount, parseAmount } from "./amount.js";

createIcons({ icons: { ArrowDownLeft, ArrowUpRight, CircleDot, Copy, Download, KeyRound, Lock, LockKeyhole, Plus, ReceiptText, RefreshCw, ScanLine, ShieldCheck, TriangleAlert, Unlock, X } });

const $ = (selector) => document.querySelector(selector);
const views = ["#loadingView", "#onboardingView", "#unlockView", "#walletView"];
let vaultRecord;
let snapshot;
let pendingSetup;
let pendingTransaction;
let scanner;

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

function setNodeState(kind, label) {
  $("#nodeState").className = `node-state ${kind}`;
  $("#nodeState span").textContent = label;
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
    toast(`无法读取本地钱包：${error.message}`, true);
  }
  registerServiceWorker();
}

function openSetup(mode) {
  pendingSetup = { mode };
  $("#passwordPanel").hidden = false;
  $("#mnemonicPanel").hidden = true;
  $("#setupStep").textContent = mode === "create" ? "创建钱包" : "恢复钱包";
  $("#setupTitle").textContent = mode === "create" ? "设置钱包密码" : "输入恢复信息";
  $("#mnemonicInputLabel").hidden = mode !== "restore";
  $("#setupPassword").value = "";
  $("#setupPasswordConfirm").value = "";
  $("#mnemonicInput").value = "";
  $("#setupDialog").showModal();
}

async function continueSetup() {
  const button = $("#setupContinue");
  const password = $("#setupPassword").value;
  if (password.length < 12 || password !== $("#setupPasswordConfirm").value) {
    toast(password.length < 12 ? "密码至少需要 12 个字符" : "两次输入的密码不一致", true);
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
  $("#setupTitle").textContent = "备份恢复词";
}

async function finishSetup() {
  const { result } = pendingSetup;
  vaultRecord = { address: result.address, backup: result.backup, savedAt: new Date().toISOString() };
  await saveVault(vaultRecord);
  $("#setupDialog").close();
  $("#receiveAddress").textContent = result.address;
  showView("#walletView");
  toast(pendingSetup.mode === "create" ? "钱包已创建" : "钱包已恢复");
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
  toast("钱包已锁定");
}

async function refreshWallet() {
  const button = $("#refreshButton");
  setBusy(button, true);
  setNodeState("loading", "正在同步");
  try {
    snapshot = await loadWalletSnapshot(vaultRecord.address);
    $("#balanceValue").textContent = formatAmount(snapshot.spendable_balance);
    $("#confirmedValue").textContent = `已确认 ${formatAmount(snapshot.confirmed_balance)} ENT`;
    $("#tipValue").textContent = `区块 ${snapshot.height.toLocaleString("zh-CN")}`;
    $("#syncTime").textContent = new Date().toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" });
    setNodeState(snapshot.tipsAgree ? "online" : "warning", snapshot.tipsAgree ? `${snapshot.nodesOnline} 个节点一致` : `${snapshot.nodesOnline} 个节点在线`);
    renderHistory(snapshot.history || []);
    if (snapshot.utxos_truncated) toast("可花费输出超过 256 个，请先在桌面钱包中归集", true);
  } catch (error) {
    setNodeState("offline", "节点离线");
    toast(error.message, true);
  } finally {
    setBusy(button, false);
  }
}

function renderHistory(history) {
  const container = $("#transactions");
  container.replaceChildren(...history.map((record) => {
    const sent = Number(record.sent) > 0;
    const amount = sent ? Math.max(0, Number(record.sent) - Number(record.received)) : Number(record.received);
    const row = document.createElement("div");
    row.className = "transaction-row";
    const icon = document.createElement("span");
    icon.className = sent ? "tx-icon sent" : "tx-icon received";
    icon.textContent = sent ? "↗" : "↙";
    const details = document.createElement("div");
    const title = document.createElement("strong");
    title.textContent = record.coinbase ? "挖矿奖励" : sent ? "已发送" : "已收到";
    const meta = document.createElement("span");
    meta.textContent = record.pending ? "等待确认" : new Date(record.timestamp * 1000).toLocaleString("zh-CN", { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
    details.append(title, meta);
    const value = document.createElement("div");
    value.className = sent ? "tx-value sent" : "tx-value";
    value.textContent = `${sent ? "−" : "+"}${formatAmount(amount)} ENT`;
    row.append(icon, details, value);
    return row;
  }));
  $("#emptyHistory").hidden = history.length > 0;
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
    toast("请先刷新钱包余额", true);
    return;
  }
  try {
    const to = $("#sendAddress").value.trim();
    const valid = await walletCall({ action: "validate_address", to });
    if (!valid.ok) throw new Error("收款地址无效");
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
    toast(error.message, true);
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
    toast(`交易已广播：${accepted.transaction_id.slice(0, 10)}…`);
    await refreshWallet();
  } catch (error) {
    toast(error.message, true);
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
  toast("加密备份已导出");
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
    toast("无法使用相机，请检查 Safari 相机权限", true);
  }
}

function stopScanner() {
  scanner?.destroy();
  scanner = undefined;
  $("#scannerDialog").close();
}

function localizeWalletError(message = "") {
  if (message.includes("authentication failed")) return "密码错误或钱包数据已损坏";
  if (message.includes("password does not meet policy")) return "密码至少需要 12 个字符";
  if (message.includes("invalid 24-word")) return "恢复词无效，请核对 24 个英文单词";
  if (message.includes("insufficient funds")) return "可用余额不足以支付金额和手续费";
  return message || "钱包操作失败";
}

async function registerServiceWorker() {
  if ("serviceWorker" in navigator && import.meta.env.PROD) {
    await navigator.serviceWorker.register(`${import.meta.env.BASE_URL}sw.js`, { scope: import.meta.env.BASE_URL }).catch(() => undefined);
  }
}

$("#createStart").addEventListener("click", () => openSetup("create"));
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
$("#copyAddress").addEventListener("click", async () => { await navigator.clipboard.writeText(vaultRecord.address); toast("地址已复制"); });
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

initialize();
