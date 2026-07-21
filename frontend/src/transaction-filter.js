export function transactionKind(transaction) {
  if (transaction?.coinbase) return "mining";
  if (!/^0(?:\.0+)?$/.test(String(transaction?.sent ?? "0"))) return "sent";
  return "received";
}
