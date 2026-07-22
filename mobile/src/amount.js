const UNITS = 100000000n;

export function parseAmount(value) {
  const normalized = value.trim();
  if (!/^\d+(\.\d{1,8})?$/.test(normalized)) throw new Error("金额格式无效，最多支持 8 位小数");
  const [whole, fraction = ""] = normalized.split(".");
  const units = BigInt(whole) * UNITS + BigInt(fraction.padEnd(8, "0"));
  if (units <= 0n || units > 18446744073709551615n) throw new Error("金额超出允许范围");
  return units.toString();
}

export function formatAmount(value) {
  const units = BigInt(value || 0);
  const whole = units / UNITS;
  const fraction = (units % UNITS).toString().padStart(8, "0").replace(/0+$/, "");
  return fraction ? `${whole}.${fraction}` : whole.toString();
}
