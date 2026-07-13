export const buildStatusChecks = (
  existingChecks: StatusCheck[],
  modifiedCheck: StatusCheck | undefined,
  index: number
): StatusCheck[] => {
  const copy = [...existingChecks];

  if (typeof modifiedCheck === "undefined") {
    copy.splice(index, 1);
    return copy;
  }

  if (index > -1) {
    copy[index] = modifiedCheck;
  } else {
    copy.push(modifiedCheck);
  }

  if (new Set(copy.map(i => i.cmd.trim())).size !== copy.length) {
    throw new Error("duplicate");
  }

  return copy;
};
