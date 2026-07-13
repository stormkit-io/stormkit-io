import mockEnvs from "./mock_environments";

export default ({ app }: { app: App }): Environment => mockEnvs({ app })[0];
