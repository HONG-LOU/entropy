const worker = new Worker(new URL("./wallet-worker.js", import.meta.url));
const pending = new Map();
let sequence = 0;

worker.addEventListener("message", ({ data }) => {
  const callback = pending.get(data.id);
  if (!callback) return;
  pending.delete(data.id);
  callback(data.result);
});

export function walletCall(payload) {
  return new Promise((resolve) => {
    const id = ++sequence;
    pending.set(id, resolve);
    worker.postMessage({ id, payload });
  });
}
