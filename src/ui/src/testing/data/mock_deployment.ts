interface Props {
  id?: string;
  appId?: string;
  envId?: string;
  isRunning?: boolean;
}

export default ({ id, appId, envId, isRunning }: Props = {}): DeploymentV2 => ({
  id: id || "15",
  apiPathPrefix: "/api",
  appId: appId || "3",
  branch: "main",
  uploadResult: {
    clientBytes: 297439,
    serverBytes: 89491,
    serverlessBytes: 5915,
  },
  commit: {
    author: "Savas Vedova \u003csavas@stormkit.io\u003e",
    message: "chore: add node version",
    sha: "14d38d2b978aeaaf1629403eef831c42b7b9cb6e",
  },
  createdAt: "1765653731",
  detailsUrl: "/apps/3/environments/3/deployments/15",
  displayName: "sample",
  duration: 17,
  envId: envId || "3",
  envName: "production",
  error: "",
  isAutoDeploy: false,
  isAutoPublish: true,
  logs: [
    {
      title: "checkout main",
      message:
        "Cloning into '/var/folders/ql/hblw2_s95ms9rjljd5x8przw0000gn/T/deployment-15/repo'...\nremote: Enumerating objects: 36, done.        \nremote: Counting objects: 100% (36/36), done.        \nremote: Compressing objects: 100% (35/35), done.        \nReceiving objects: 100% (36/36), 28.65 KiB | 156.00 KiB/s, done.\nremote: Total 36 (delta 2), reused 18 (delta 0), pack-reused 0 (from 0)        \nResolving deltas: 100% (2/2), done.\n",
      status: true,
      payload: null,
      duration: 1,
    },
    {
      title: "mise install",
      message: "mise all tools are installed\n",
      status: true,
      payload: null,
      duration: 0,
    },
    {
      title: "node --version",
      message: "v24.6.0\n",
      status: true,
      payload: null,
      duration: 1,
    },
    {
      title: "environment variables",
      message:
        "NODE_ENV=production\nSK_APP_ID=3\nSK_BRANCH_NAME=main\nSK_COMMIT_SHA=14d38d2b978aeaaf1629403eef831c42b7b9cb6e\nSK_DEPLOYMENT_ID=15\nSK_DEPLOYMENT_URL=http://sample--15.localhost:8888\nSK_ENV=production\nSK_ENV_ID=3\nSK_ENV_URL=http://sample.localhost:8888\nSTORMKIT=true\n",
      status: true,
      payload: null,
      duration: 0,
    },
    {
      title: "npm ci",
      message:
        "registry: \nhttps://registry.npmjs.org/\nadded 298 packages in 8s\n25 packages are looking for funding\n  run `npm fund` for details\n",
      status: true,
      payload: null,
      duration: 8,
    },
    {
      title: "npm run build",
      message:
        "\u003e sample-project@0.1.0 build\n\u003e vite build\nvite v6.0.6 building for production...\n(!) outDir /private/var/folders/ql/hblw2_s95ms9rjljd5x8przw0000gn/T/deployment-15/repo/build is not inside project root and will not be emptied.\nUse --emptyOutDir to override.\ntransforming...\n✓ 57 modules transformed.\nrendering chunks...\ncomputing gzip size...\n../build/index.html                   2.07 kB │ gzip:   1.01 kB\n../build/assets/logo-C26LKBEw.png     6.17 kB\n../build/assets/index-BwhdVVS8.css    1.02 kB │ gzip:   0.57 kB\n../build/assets/index-QfnpvQ4A.js   597.63 kB │ gzip: 193.26 kB\n(!) Some chunks are larger than 500 kB after minification. Consider:\n- Using dynamic import() to code-split the application\n- Use build.rollupOptions.output.manualChunks to improve chunking: https://rollupjs.org/configuration-options/#output-manualchunks\n- Adjust chunk size limit for this warning via build.chunkSizeWarningLimit.\n✓ built in 674ms\n",
      status: true,
      payload: null,
      duration: 2,
    },
    {
      title: "build api",
      message:
        "We found `api` dir. We'll try to build it automatically.\nYou can turn off automatic api builds by specifying the following environment variable: `SK_BUILD_API=off`\nadded 4 packages, and audited 5 packages in 1s\n3 packages are looking for funding\n  run `npm fund` for details\nfound 0 vulnerabilities\n\n",
      status: true,
      payload: null,
      duration: 2,
    },
    {
      title: "deploy",
      message:
        "\nSuccessfully deployed client side.\nTotal bytes uploaded: 207.9kB\n\n\nSuccessfully deployed api.\nPackage size: 89.5kB",
      status: true,
      payload: null,
      duration: 2,
    },
  ],
  previewUrl: "http://sample--15.localhost:8888",
  published: [
    {
      envId: "3",
      percentage: 100,
    },
  ],
  repo: "github/stormkit-io/sample-project",
  snapshot: {
    build: {
      previewLinks: null,
      vars: {
        NODE_ENV: "production",
      },
    },
    env: "production",
    envId: "3",
  },
  status: isRunning ? "running" : "success",
  statusChecks: [],
  statusChecksPassed: null,
  stoppedAt: "1765653748",
  stoppedManually: false,
});
