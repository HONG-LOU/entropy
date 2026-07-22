let ready;

async function loadWasm() {
  const baseURL = import.meta.env.BASE_URL;
  importScripts(`${baseURL}wasm/wasm_exec.js`);
  const go = new Go();
  const response = await fetch(`${baseURL}wasm/entcoin.wasm`);
  const bytes = await response.arrayBuffer();
  const instance = await WebAssembly.instantiate(bytes, go.importObject);
  go.run(instance.instance);
  while (typeof self.entcoinWasmCall !== "function") {
    await new Promise((resolve) => setTimeout(resolve, 0));
  }
}

ready = loadWasm();

self.addEventListener("message", async (event) => {
  const { id, payload } = event.data;
  try {
    await ready;
    const result = JSON.parse(self.entcoinWasmCall(JSON.stringify(payload)));
    self.postMessage({ id, result });
  } catch (error) {
    self.postMessage({ id, result: { ok: false, error: error?.message || "无法启动本地钱包引擎" } });
  }
});
