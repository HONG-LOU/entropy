import test from "node:test";
import assert from "node:assert/strict";

globalThis.window = {
  localStorage: {
    getItem: () => "zh",
    setItem: () => {},
  },
};

const { currentLanguage, currentLocale, translate, translateError } = await import("../src/i18n.js");

test("saved Chinese language controls desktop locale", () => {
  assert.equal(currentLanguage(), "zh");
  assert.equal(currentLocale(), "zh-CN");
});

test("desktop copy uses natural Chinese translations", () => {
  assert.equal(translate("Software update"), "软件更新");
  assert.equal(translate("Update and restart"), "更新并重启");
  assert.equal(translate("Your wallet has not been marked as backed up."), "这个钱包还没有完成备份确认。");
});

test("dynamic version and block messages preserve their values", () => {
  assert.equal(translate("Entcoin v1.0.9 is available"), "Entcoin v1.0.9 可以更新");
  assert.equal(translate("Block 68012 mined"), "区块 #68012 已挖出");
  assert.equal(translate("6 confirmations"), "6 次确认");
});

test("unmapped backend failures remain natural in Chinese", () => {
  assert.equal(translateError("download release checksums: connection reset"), "更新没有完成，请检查网络连接后重试。");
  assert.equal(translateError("open protected wallet: keyring locked"), "钱包操作没有完成，请检查输入和系统密钥服务后重试。");
  assert.equal(translateError("unexpected internal failure"), "操作没有完成，请稍后重试。");
});
