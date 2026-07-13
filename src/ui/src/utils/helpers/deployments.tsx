import React from "react";

/**
 * Formats a date.
 *
 * @param {*} ts
 */
export const formattedDate = (ts: number): string => {
  const date = new Date(ts * 1000);
  const now = new Date();

  if (
    date.getDate() === now.getDate() &&
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth()
  ) {
    return `Today at ${date.toLocaleTimeString("de-CH", {
      hour: "2-digit",
      minute: "2-digit",
    })}`;
  }

  return date.toLocaleDateString("de-CH", {
    month: "2-digit",
    day: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
};

export interface Commit {
  author: string;
  branch: string;
  msg: React.ReactNode;
}

interface PayloadCommit {
  branch: string;
  commit: {
    message: string;
    author: string;
  };
}

/**
 * Parses a deployment object and extracts useful info.
 *
 * @param {object} deployment
 */
export const parseCommit = (deployment?: Deployment): Commit => {
  if (deployment?.commit?.author) {
    return {
      author: deployment.commit.author.split("<")[0],
      msg: deployment.commit.message,
      branch: deployment.branch,
    };
  }

  if (deployment?.logs?.length) {
    for (let i = 0; i < deployment.logs.length; i++) {
      const payload = deployment.logs[i].payload as PayloadCommit;
      const commit = payload?.commit || {};
      const branch = payload?.branch || "";
      const msg = (commit.message || "").split("\n")[0];
      const author = (commit.author || "").split("<")[0];

      if (msg) {
        return { author, branch, msg };
      }
    }
  }

  const exitCode = deployment?.exit;

  let msg = (
    <div>
      Your deployment is queued up
      <br /> We will process it immediately as soon as we have available slots
    </div>
  );

  if (exitCode && exitCode > 0) {
    msg = (
      <div>
        Deployment has failed
        <br />
        Mostly this happens when Stormkit cannot checkout your repository
      </div>
    );
  } else if (exitCode === -1) {
    msg = <div>Deployment has been stopped manually</div>;
  } else if (exitCode === 0) {
    msg = <div>Deployment is completed successfully</div>;
  }

  return {
    author: "",
    branch: "",
    msg,
  };
};

/**
 * Converts bytes to MB
 * @param {number} byte
 */
export const bytesToMB = (byte: string | number): string =>
  `${(+byte / 1000000).toFixed(2)}MB`;
