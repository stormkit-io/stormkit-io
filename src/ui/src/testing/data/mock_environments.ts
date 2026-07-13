export default ({ app }: { app: App }): Environment[] => [
  {
    id: "1429333243019",
    env: "production",
    name: "production",
    branch: "master",
    build: {
      distFolder: "packages/console/dist",
      buildCmd: "yarn test \u0026\u0026 yarn run build:console",
      previewLinks: true,
      vars: {
        BABEL_ENV: "production",
        NODE_ENV: "production",
      },
    },
    autoDeploy: false,
    autoPublish: true,
    appId: app.id,
    lastDeploy: { id: "10556341488718", createdAt: 1588622630, exit: 0 },
    preview: "https://app.stormkit.io",
  },
  {
    id: "863521234275",
    env: "development",
    name: "development",
    branch: "env/dev",
    build: {
      distFolder: "packages/console/dist",
      buildCmd: "yarn run build:console",
      previewLinks: true,
      vars: {
        API_DOMAIN: "https://api.stormkit.io",
        BABEL_ENV: "production",
        NODE_ENV: "production",
        ROOT: "\u003croot\u003e/packages/console",
      },
    },
    autoDeploy: false,
    autoPublish: false,
    appId: app.id,
    lastDeploy: { id: "3414609680676", createdAt: 1568463533, exit: 0 },
    preview: "https://app.stormkit.io",
  },
];
