import test from "node:test";
import assert from "node:assert/strict";
import { detectLanguage, messages, translate } from "../src/i18n.js";

test("中英文词典键保持一致", () => {
  assert.deepEqual(Object.keys(messages.en).sort(), Object.keys(messages.zh).sort());
});

test("优先使用已保存语言并按浏览器语言回退", () => {
  assert.equal(detectLanguage({ getItem: () => "en" }, ["zh-CN"]), "en");
  assert.equal(detectLanguage({ getItem: () => null }, ["zh-Hans", "en"]), "zh");
  assert.equal(detectLanguage({ getItem: () => null }, ["en-US"]), "en");
});

test("翻译支持变量替换", () => {
  assert.equal(translate("zh", "nodesAgree", { count: 2 }), "2 个节点一致");
  assert.equal(translate("en", "blockHeight", { height: "1,234" }), "Block 1,234");
});
